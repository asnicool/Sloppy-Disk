package metadata

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"sloppy-disk/internal/config"
	"sloppy-disk/internal/models"
)

const (
	albumArtAPIURL    = "https://api.albumart.digital/v1"
	albumArtUserAgent = "sloppy-disk/1.0"
)

// AlbumArtProvider implements the Provider interface for AlbumArt.digital
type AlbumArtProvider struct {
	client *http.Client
	apiKey string
}

// NewAlbumArtProvider creates a new AlbumArt.digital provider
func NewAlbumArtProvider() *AlbumArtProvider {
	cfg := config.Get()
	return &AlbumArtProvider{
		client: &http.Client{Timeout: 30 * time.Second},
		apiKey: cfg.AlbumArtAPIKey,
	}
}

// Name returns the provider name
func (p *AlbumArtProvider) Name() string {
	return "AlbumArt.digital"
}

// Search searches for albums on AlbumArt.digital
func (p *AlbumArtProvider) Search(artist, album string) ([]models.MetadataCandidate, error) {
	// AlbumArt.digital is primarily for cover art, not metadata
	// Return empty candidates
	return []models.MetadataCandidate{}, nil
}

// GetReleaseDetails fetches detailed metadata (limited for AlbumArt.digital)
func (p *AlbumArtProvider) GetReleaseDetails(externalID string) (*models.MetadataCandidate, error) {
	return nil, fmt.Errorf("AlbumArt.digital does not provide detailed metadata")
}

// GetCoverArt fetches cover art from AlbumArt.digital
func (p *AlbumArtProvider) GetCoverArt(artist, album string) ([]models.CoverArtCandidate, error) {
	if p.apiKey == "" {
		return []models.CoverArtCandidate{}, nil
	}

	query := fmt.Sprintf("%s %s", artist, album)
	params := url.Values{
		"q": {query},
	}

	req, err := p.newRequest("GET", "/search", params)
	if err != nil {
		return nil, NewProviderError("AlbumArt.digital", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, NewProviderError("AlbumArt.digital", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, NewProviderError("AlbumArt.digital", fmt.Errorf("API returned status %d", resp.StatusCode))
	}

	var result struct {
		Results []struct {
			Album        string `json:"album"`
			Artist       string `json:"artist"`
			ImageURL     string `json:"imageUrl"`
			ThumbnailURL string `json:"thumbnailUrl"`
			Width        int    `json:"width"`
			Height       int    `json:"height"`
			Source       string `json:"source"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, NewProviderError("AlbumArt.digital", err)
	}

	var candidates []models.CoverArtCandidate
	for _, r := range result.Results {
		size := "full"
		if r.Width < 300 {
			size = "small"
		} else if r.Width < 600 {
			size = "medium"
		}

		candidates = append(candidates, models.CoverArtCandidate{
			Source:    "AlbumArt.digital",
			URL:       r.ImageURL,
			Thumbnail: r.ThumbnailURL,
			Width:     r.Width,
			Height:    r.Height,
			Size:      size,
		})
	}

	return candidates, nil
}

// newRequest creates a new HTTP request with proper headers
func (p *AlbumArtProvider) newRequest(method, path string, params url.Values) (*http.Request, error) {
	url := albumArtAPIURL + path + "?" + params.Encode()
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", albumArtUserAgent)
	req.Header.Set("X-API-Key", p.apiKey)
	return req, nil
}
