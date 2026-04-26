package metadata

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"sloppy-disk/internal/models"
)

const (
	gnuDBAPIURL    = "https://gnudb.gnudb.org/~cddb/cddb.cgi"
	gnuDBUserAgent = "sloppy-disk/1.0"
)

// GNUDbProvider implements the Provider interface for GNUDb
type GNUDbProvider struct {
	client *http.Client
}

// NewGNUDbProvider creates a new GNUDb provider
func NewGNUDbProvider() *GNUDbProvider {
	return &GNUDbProvider{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Name returns the provider name
func (p *GNUDbProvider) Name() string {
	return "GNUDb"
}

// Search searches for albums on GNUDb
// Note: GNUDb uses a different protocol (CDDB). We use the HTTP API.
func (p *GNUDbProvider) Search(artist, album string) ([]models.MetadataCandidate, error) {
	// GNUDb HTTP API requires specific parameters
	// Using the simple search endpoint
	params := url.Values{
		"cmd":   {"cddb search all " + artist + " " + album},
		"hello": {"anonymous localhost mpd-client 1.0"},
		"proto": {"6"},
	}

	req, err := p.newRequest("POST", params)
	if err != nil {
		return nil, NewProviderError("GNUDb", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, NewProviderError("GNUDb", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, NewProviderError("GNUDb", fmt.Errorf("API returned status %d", resp.StatusCode))
	}

	// Parse GNUDb response format
	// GNUDb returns: code category discid artist / title
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, NewProviderError("GNUDb", err)
	}

	lines := strings.Split(string(body), "\n")
	if len(lines) == 0 {
		return nil, nil
	}

	// Check status code in first line
	firstLine := strings.TrimSpace(lines[0])
	if !strings.HasPrefix(firstLine, "200") {
		// If 202 (No match), return empty
		if strings.HasPrefix(firstLine, "202") {
			return []models.MetadataCandidate{}, nil
		}
		// Otherwise, treat as error (e.g. 403, 409, etc which might come as body even with 200 OK HTTP)
		return nil, NewProviderError("GNUDb", fmt.Errorf("CDDB error: %s", firstLine))
	}

	var candidates []models.MetadataCandidate

	// Skip first line (status)
	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, ".") {
			continue
		}

		parts := strings.SplitN(line, " ", 4)
		if len(parts) < 4 {
			continue
		}

		// Parse: category discid artist / title
		// Note: The search output format (after 200 OK) is: category discid Artist / Title
		category := parts[0]
		discid := parts[1]
		rest := parts[2] + " " + parts[3] // Rejoin if split wrongly or just take parts[3]?
		// Wait, SplitN(4) -> [cat, discid, part3, part4] NO.
		// Line: rock 12345678 Artist / Title
		// parts[0] = rock
		// parts[1] = 12345678
		// parts[2]... we want the rest.

		// Let's reuse the SplitN 4 logic but careful.
		// Actually, sometimes standard CDDB output is: category discid Artist / Title
		// SplitN(line, " ", 3) -> [cat, discid, title_info]

		parts = strings.SplitN(line, " ", 3)
		if len(parts) < 3 {
			continue
		}
		category = parts[0]
		discid = parts[1]
		rest = parts[2]

		// Parse "Artist / Title"
		artistName := ""
		title := rest
		if idx := strings.Index(rest, " / "); idx != -1 {
			artistName = rest[:idx]
			title = rest[idx+3:]
		}

		candidates = append(candidates, models.MetadataCandidate{
			Source:     "GNUDb",
			Artist:     artistName,
			Album:      title,
			ExternalID: category + "/" + discid,
			Metadata: map[string]interface{}{
				"category": category,
				"discid":   discid,
			},
		})
	}

	return candidates, nil
}

// GetReleaseDetails fetches detailed metadata for a GNUDb release
func (p *GNUDbProvider) GetReleaseDetails(externalID string) (*models.MetadataCandidate, error) {
	// Parse externalID (format: category/discid)
	parts := strings.SplitN(externalID, "/", 2)
	if len(parts) != 2 {
		return nil, NewProviderError("GNUDb", fmt.Errorf("invalid external ID format"))
	}

	category := parts[0]
	discid := parts[1]

	params := url.Values{
		"cmd":   {"cddb read " + category + " " + discid},
		"hello": {"anonymous localhost mpd-client 1.0"},
		"proto": {"6"},
	}

	req, err := p.newRequest("POST", params)
	if err != nil {
		return nil, NewProviderError("GNUDb", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, NewProviderError("GNUDb", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, NewProviderError("GNUDb", fmt.Errorf("API returned status %d", resp.StatusCode))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, NewProviderError("GNUDb", err)
	}

	// Parse GNUDb response format
	// DTITLE, DYEAR, DGENRE, TTITLE0-9, etc.
	lines := strings.Split(string(body), "\n")
	if len(lines) == 0 {
		return nil, NewProviderError("GNUDb", fmt.Errorf("empty response"))
	}

	// Check status in first line (should be 210)
	firstLine := strings.TrimSpace(lines[0])
	if !strings.HasPrefix(firstLine, "210") {
		return nil, NewProviderError("GNUDb", fmt.Errorf("CDDB error: %s", firstLine))
	}

	artist := ""
	album := ""
	year := ""
	genre := ""
	var tracks []models.Song

	// Skip status line
	for _, line := range lines[1:] {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, ".") {
			continue
		}

		if strings.HasPrefix(line, "DTITLE=") {
			title := strings.TrimPrefix(line, "DTITLE=")
			if idx := strings.Index(title, " / "); idx != -1 {
				artist = title[:idx]
				album = title[idx+3:]
			} else if artist == "" { // Only set if not already found (sometimes multiple DTITLE lines?)
				album = title
			}
		} else if strings.HasPrefix(line, "DYEAR=") {
			year = strings.TrimPrefix(line, "DYEAR=")
		} else if strings.HasPrefix(line, "DGENRE=") {
			genre = strings.TrimPrefix(line, "DGENRE=")
		} else if strings.HasPrefix(line, "TTITLE") {
			trackLine := strings.TrimPrefix(line, "TTITLE")
			if idx := strings.Index(trackLine, "="); idx != -1 {
				trackNumStr := trackLine[:idx]
				trackTitle := trackLine[idx+1:]
				if n, err := strconv.Atoi(trackNumStr); err == nil {
					tracks = append(tracks, models.Song{
						Title:  trackTitle,
						Artist: artist,
						Album:  album,
						Track:  fmt.Sprintf("%d", n+1),
					})
				}
			}
		}
	}

	// Post-processing to ensure all tracks have artist/album if set later
	for i := range tracks {
		if tracks[i].Artist == "" {
			tracks[i].Artist = artist
		}
		if tracks[i].Album == "" {
			tracks[i].Album = album
		}
	}

	return &models.MetadataCandidate{
		Source:     "GNUDb",
		Artist:     artist,
		Album:      album,
		Year:       year,
		Genre:      genre,
		Tracks:     tracks,
		ExternalID: externalID,
	}, nil
}

// GetCoverArt fetches cover art from GNUDb
// Note: GNUDb doesn't typically provide cover art, return empty
func (p *GNUDbProvider) GetCoverArt(artist, album string) ([]models.CoverArtCandidate, error) {
	// GNUDb doesn't provide cover art
	return []models.CoverArtCandidate{}, nil
}

// newRequest creates a new HTTP request for GNUDb
func (p *GNUDbProvider) newRequest(method string, params url.Values) (*http.Request, error) {
	// GNUDb can use POST with form-encoded body
	req, err := http.NewRequest(method, gnuDBAPIURL, strings.NewReader(params.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", gnuDBUserAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req, nil
}
