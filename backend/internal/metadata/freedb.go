package metadata

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"mpd-client-modern/internal/models"
)

const (
	freeDBAPIURL    = "https://freedb.freedb.org/~cddb/cddb.cgi"
	freeDBUserAgent = "mpd-client-modern/1.0"
)

// FreeDBProvider implements the Provider interface for FreeDB
type FreeDBProvider struct {
	client *http.Client
}

// NewFreeDBProvider creates a new FreeDB provider
func NewFreeDBProvider() *FreeDBProvider {
	return &FreeDBProvider{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Name returns the provider name
func (p *FreeDBProvider) Name() string {
	return "FreeDB"
}

// Search searches for albums on FreeDB
// Note: FreeDB uses a different protocol (CDDB). We use the HTTP API.
func (p *FreeDBProvider) Search(artist, album string) ([]models.MetadataCandidate, error) {
	// FreeDB HTTP API requires specific parameters
	// Using the simple search endpoint
	params := url.Values{
		"cmd":   {"cddb search all * " + album},
		"proto": {"5"},
	}

	req, err := p.newRequest("POST", params)
	if err != nil {
		return nil, NewProviderError("FreeDB", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, NewProviderError("FreeDB", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, NewProviderError("FreeDB", fmt.Errorf("API returned status %d", resp.StatusCode))
	}

	// Parse FreeDB response format
	// FreeDB returns: code category discid artist title
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, NewProviderError("FreeDB", err)
	}

	lines := strings.Split(string(body), "\n")
	var candidates []models.MetadataCandidate

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, ".") {
			continue
		}

		parts := strings.SplitN(line, " ", 4)
		if len(parts) < 4 {
			continue
		}

		// Parse: code category discid artist title
		category := parts[1]
		discid := parts[2]
		artistName := parts[3]
		title := parts[4]

		// If artist name is in "Artist - Title" format, split it
		if idx := strings.LastIndex(title, " - "); idx != -1 {
			artistName = title[:idx]
			title = title[idx+3:]
		}

		candidates = append(candidates, models.MetadataCandidate{
			Source:     "FreeDB",
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

// GetReleaseDetails fetches detailed metadata for a FreeDB release
func (p *FreeDBProvider) GetReleaseDetails(externalID string) (*models.MetadataCandidate, error) {
	// Parse externalID (format: category/discid)
	parts := strings.SplitN(externalID, "/", 2)
	if len(parts) != 2 {
		return nil, NewProviderError("FreeDB", fmt.Errorf("invalid external ID format"))
	}

	category := parts[0]
	discid := parts[1]

	params := url.Values{
		"cmd":   {"cddb read " + category + " " + discid},
		"proto": {"5"},
	}

	req, err := p.newRequest("POST", params)
	if err != nil {
		return nil, NewProviderError("FreeDB", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, NewProviderError("FreeDB", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, NewProviderError("FreeDB", fmt.Errorf("API returned status %d", resp.StatusCode))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, NewProviderError("FreeDB", err)
	}

	// Parse FreeDB database format
	// DTITLE, DYEAR, DGENRE, TTITLE0-9, etc.
	lines := strings.Split(string(body), "\n")

	artist := ""
	album := ""
	year := ""
	genre := ""
	var tracks []models.Song

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, ".") {
			continue
		}

		if strings.HasPrefix(line, "DTITLE=") {
			title := strings.TrimPrefix(line, "DTITLE=")
			if idx := strings.LastIndex(title, " / "); idx != -1 {
				artist = title[:idx]
				album = title[idx+3:]
			} else {
				album = title
			}
		} else if strings.HasPrefix(line, "DYEAR=") {
			year = strings.TrimPrefix(line, "DYEAR=")
		} else if strings.HasPrefix(line, "DGENRE=") {
			genre = strings.TrimPrefix(line, "DGENRE=")
		} else if strings.HasPrefix(line, "TTITLE") {
			trackLine := strings.TrimPrefix(line, "TTITLE")
			if idx := strings.Index(trackLine, "="); idx != -1 {
				trackNum := trackLine[:idx]
				trackTitle := trackLine[idx+1:]
				if n, err := strconv.Atoi(trackNum); err == nil {
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

	return &models.MetadataCandidate{
		Source:     "FreeDB",
		Artist:     artist,
		Album:      album,
		Year:       year,
		Genre:      genre,
		Tracks:     tracks,
		ExternalID: externalID,
	}, nil
}

// GetCoverArt fetches cover art from FreeDB
// Note: FreeDB doesn't typically provide cover art, return empty
func (p *FreeDBProvider) GetCoverArt(artist, album string) ([]models.CoverArtCandidate, error) {
	// FreeDB doesn't provide cover art
	return []models.CoverArtCandidate{}, nil
}

// newRequest creates a new HTTP request for FreeDB
func (p *FreeDBProvider) newRequest(method string, params url.Values) (*http.Request, error) {
	// FreeDB uses POST with form-encoded body
	req, err := http.NewRequest(method, freeDBAPIURL, strings.NewReader(params.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", freeDBUserAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return req, nil
}
