package metadata

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"mpd-client-modern/internal/config"
	"mpd-client-modern/internal/models"
)

type DiscogsProvider struct {
	client *http.Client
}

func NewDiscogsProvider() *DiscogsProvider {
	return &DiscogsProvider{
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *DiscogsProvider) Search(artist, album string) ([]models.MetadataCandidate, error) {
	cfg := config.Get()
	if cfg.DiscogsToken == "" {
		return nil, fmt.Errorf("Discogs token not configured")
	}

	query := url.Values{}
	query.Set("release_title", album)
	query.Set("artist", artist)
	query.Set("type", "release")
	query.Set("token", cfg.DiscogsToken)

	req, _ := http.NewRequest("GET", "https://api.discogs.com/database/search?"+query.Encode(), nil)
	req.Header.Set("User-Agent", "MPDClientModern/1.0")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Discogs API error: %s", resp.Status)
	}

	var result struct {
		Results []struct {
			Title string `json:"title"`
			Year  string `json:"year"`
			ID    int    `json:"id"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var candidates []models.MetadataCandidate
	for _, r := range result.Results {
		candidates = append(candidates, models.MetadataCandidate{
			Source:     "Discogs",
			Album:      r.Title, // Discogs title is usually "Artist - Album"
			Year:       r.Year,
			ExternalID: fmt.Sprintf("%d", r.ID),
		})
	}

	return candidates, nil
}
