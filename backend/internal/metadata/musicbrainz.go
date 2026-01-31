package metadata

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"mpd-client-modern/internal/models"
)

const (
	musicBrainzAPIURL     = "https://musicbrainz.org/ws/2"
	musicBrainzCoverURL   = "https://coverartarchive.org"
	musicBrainzUserAgent  = "mpd-client-modern/1.0 (contact@example.com)"
	musicBrainzRateLimit  = time.Second / 1 // 1 request per second
)

// MusicBrainzProvider implements the Provider interface for MusicBrainz
type MusicBrainzProvider struct {
	client  *http.Client
	lastReq time.Time
}

// NewMusicBrainzProvider creates a new MusicBrainz provider
func NewMusicBrainzProvider() *MusicBrainzProvider {
	return &MusicBrainzProvider{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name returns the provider name
func (p *MusicBrainzProvider) Name() string {
	return "MusicBrainz"
}

// Search searches for releases on MusicBrainz
func (p *MusicBrainzProvider) Search(artist, album string) ([]models.MetadataCandidate, error) {
	// Rate limiting
	if time.Since(p.lastReq) < musicBrainzRateLimit {
		time.Sleep(musicBrainzRateLimit - time.Since(p.lastReq))
	}

	query := fmt.Sprintf(`artist:"%s" AND release:"%s"`, artist, album)
	params := url.Values{
		"query":  {query},
		"type":   {"album"},
		"limit":  {"20"},
		"fmt":    {"json"},
	}

	req, err := p.newRequest("GET", "/release-group/", params)
	if err != nil {
		return nil, NewProviderError("MusicBrainz", err)
	}

	var result struct {
		ReleaseGroups []struct {
			ID                string `json:"id"`
			Title             string `json:"title"`
			FirstReleaseDate  string `json:"first-release-date"`
			ArtistCredits []struct {
				Artist struct {
					ID   string `json:"id"`
					Name string `json:"name"`
				} `json:"artist"`
			} `json:"artist-credits"`
			Releases []struct {
				ID string `json:"id"`
			} `json:"releases"`
		} `json:"release-groups"`
		Count int `json:"count"`
	}

	p.lastReq = time.Now()
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, NewProviderError("MusicBrainz", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, NewProviderError("MusicBrainz", fmt.Errorf("API returned status %d", resp.StatusCode))
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, NewProviderError("MusicBrainz", err)
	}

	var candidates []models.MetadataCandidate
	for _, rg := range result.ReleaseGroups {
		artistName := artist
		if len(rg.ArtistCredits) > 0 {
			artistName = rg.ArtistCredits[0].Artist.Name
		}

		releaseID := ""
		if len(rg.Releases) > 0 {
			releaseID = rg.Releases[0].ID
		}

		candidates = append(candidates, models.MetadataCandidate{
			Source:     "MusicBrainz",
			Artist:     artistName,
			Album:      rg.Title,
			Year:       rg.FirstReleaseDate[:4], // Take first 4 chars for year
			ExternalID: rg.ID,
			Metadata: map[string]interface{}{
				"releaseGroupID": rg.ID,
				"releaseID":      releaseID,
			},
		})
	}

	return candidates, nil
}

// GetReleaseDetails fetches detailed metadata for a MusicBrainz release
func (p *MusicBrainzProvider) GetReleaseDetails(externalID string) (*models.MetadataCandidate, error) {
	// Rate limiting
	if time.Since(p.lastReq) < musicBrainzRateLimit {
		time.Sleep(musicBrainzRateLimit - time.Since(p.lastReq))
	}

	// Remove "release-group:" prefix if present
	id := strings.TrimPrefix(externalID, "release-group:")

	params := url.Values{
		"inc":  {"recordings+artist-credits+release-groups"},
		"fmt":  {"json"},
	}

	req, err := p.newRequest("GET", "/release/"+id, params)
	if err != nil {
		return nil, NewProviderError("MusicBrainz", err)
	}

	p.lastReq = time.Now()
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, NewProviderError("MusicBrainz", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, NewProviderError("MusicBrainz", fmt.Errorf("API returned status %d", resp.StatusCode))
	}

	var release struct {
		ID           string `json:"id"`
		Title        string `json:"title"`
		Date         string `json:"date"`
		Country      string `json:"country"`
		Barcode      string `json:"barcode"`
		LabelInfo    []struct {
			Label struct {
				Name string `json:"name"`
			} `json:"label"`
		} `json:"label-info"`
		ArtistCredits []struct {
			Artist struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"artist"`
			JoinPhrase string `json:"join-phrase"`
		} `json:"artist-credits"`
		Media []struct {
			Position int `json:"position"`
			Format   string `json:"format"`
			Tracks   []struct {
				ID          string `json:"id"`
				Title       string `json:"title"`
				Length      int    `json:"length"` // in milliseconds
				TrackNumber int    `json:"number"`
			} `json:"tracks"`
		} `json:"media"`
		ReleaseGroup struct {
			ID   string `json:"id"`
			Type string `json:"type"`
		} `json:"release-group"`
		Genre []struct {
			Name string `json:"name"`
		} `json:"genres"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, NewProviderError("MusicBrainz", err)
	}

	// Build artist name from credits
	var artistName strings.Builder
	for i, ac := range release.ArtistCredits {
		if i > 0 {
			artistName.WriteString(ac.JoinPhrase)
		}
		artistName.WriteString(ac.Artist.Name)
	}

	// Build track list
	var tracks []models.Song
	discNumber := 1
	for _, media := range release.Media {
		for _, track := range media.Tracks {
			tracks = append(tracks, models.Song{
				Title:    track.Title,
				Artist:   artistName.String(),
				Album:    release.Title,
				Track:    fmt.Sprintf("%d", track.TrackNumber),
				Disc:     fmt.Sprintf("%d", discNumber),
				Duration: track.Length / 1000, // Convert to seconds
			})
		}
		discNumber++
	}

	// Extract genres
	var genres []string
	for _, g := range release.Genre {
		genres = append(genres, g.Name)
	}

	// Extract label
	label := ""
	if len(release.LabelInfo) > 0 {
		label = release.LabelInfo[0].Label.Name
	}

	return &models.MetadataCandidate{
		Source:  "MusicBrainz",
		Artist:  artistName.String(),
		Album:   release.Title,
		Year:    release.Date[:4],
		Genre:   strings.Join(genres, "; "),
		Tracks:  tracks,
		ExternalID: release.ID,
		Metadata: map[string]interface{}{
			"releaseGroupID":   release.ReleaseGroup.ID,
			"releaseGroupType": release.ReleaseGroup.Type,
			"country":          release.Country,
			"barcode":          release.Barcode,
			"label":            label,
		},
	}, nil
}

// GetCoverArt fetches cover art from Cover Art Archive
func (p *MusicBrainzProvider) GetCoverArt(artist, album string) ([]models.CoverArtCandidate, error) {
	// First search for the release
	candidates, err := p.Search(artist, album)
	if err != nil {
		return nil, err
	}

	if len(candidates) == 0 {
		return []models.CoverArtCandidate{}, nil
	}

	// Get the first release ID
	releaseID := ""
	if meta, ok := candidates[0].Metadata["releaseID"]; ok {
		releaseID = meta.(string)
	}
	if releaseID == "" {
		return []models.CoverArtCandidate{}, nil
	}

	// Try to fetch from Cover Art Archive
	url := fmt.Sprintf("%s/release/%s/front", musicBrainzCoverURL, releaseID)
	resp, err := http.Head(url)
	if err != nil {
		return []models.CoverArtCandidate{}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []models.CoverArtCandidate{}, nil
	}

	return []models.CoverArtCandidate{
		{
			Source:    "MusicBrainz",
			URL:       url,
			Thumbnail: url + "-250", // Cover Art Archive supports thumbnails
			Size:      "full",
		},
	}, nil
}

// newRequest creates a new HTTP request with proper headers
func (p *MusicBrainzProvider) newRequest(method, path string, params url.Values) (*http.Request, error) {
	url := musicBrainzAPIURL + path + "?" + params.Encode()
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", musicBrainzUserAgent)
	return req, nil
}
