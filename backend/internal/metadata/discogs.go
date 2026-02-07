package metadata

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"mpd-client-modern/internal/config"
	"mpd-client-modern/internal/models"
)

const (
	discogsAPIURL    = "https://api.discogs.com"
	discogsUserAgent = "mpd-client-modern/1.0"
)

// DiscogsProvider implements the Provider interface for Discogs
type DiscogsProvider struct {
	client *http.Client
}

// NewDiscogsProvider creates a new Discogs provider
func NewDiscogsProvider() *DiscogsProvider {
	return &DiscogsProvider{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Name returns the provider name
func (p *DiscogsProvider) Name() string {
	return "Discogs"
}

// Search searches for releases on Discogs
func (p *DiscogsProvider) Search(artist, album string) ([]models.MetadataCandidate, error) {
	cfg := config.Get()
	hasKeySecret := cfg.DiscogsKey != "" && cfg.DiscogsSecret != ""
	hasToken := cfg.DiscogsToken != ""

	if !hasKeySecret && !hasToken {
		return nil, fmt.Errorf("Discogs credentials not configured")
	}

	query := url.Values{}
	query.Set("release_title", album)
	query.Set("artist", artist)
	query.Set("type", "release")

	// Use token if Key/Secret are not available
	if !hasKeySecret && hasToken {
		query.Set("token", cfg.DiscogsToken)
	}

	req, err := p.newRequest("GET", "/database/search", query)
	if err != nil {
		return nil, NewProviderError("Discogs", err)
	}

	// Add Authorization header if Key/Secret are available
	if hasKeySecret {
		req.Header.Set("Authorization", fmt.Sprintf("Discogs key=%s, secret=%s", cfg.DiscogsKey, cfg.DiscogsSecret))
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, NewProviderError("Discogs", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, NewProviderError("Discogs", fmt.Errorf("API error: %s", resp.Status))
	}

	var result struct {
		Results []struct {
			Title      string   `json:"title"`
			Year       string   `json:"year"`
			ID         int      `json:"id"`
			Thumb      string   `json:"thumb"`
			CoverImage string   `json:"cover_image"`
			Genre      []string `json:"genre"`
			Style      []string `json:"style"`
		} `json:"results"`
		Pagination struct {
			Pages int `json:"pages"`
		} `json:"pagination"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, NewProviderError("Discogs", err)
	}

	var candidates []models.MetadataCandidate
	for _, r := range result.Results {
		// Parse title which is usually "Artist - Album"
		albumTitle := r.Title
		if idx := findLastIndex(r.Title, " - "); idx != -1 {
			albumTitle = r.Title[idx+3:]
		}

		genre := ""
		if len(r.Genre) > 0 {
			genre = r.Genre[0]
		}

		candidates = append(candidates, models.MetadataCandidate{
			Source:     "Discogs",
			Artist:     artist,
			Album:      albumTitle,
			Year:       r.Year,
			Genre:      genre,
			ExternalID: strconv.Itoa(r.ID),
			Metadata: map[string]interface{}{
				"fullTitle":  r.Title,
				"thumbnail":  r.Thumb,
				"coverImage": r.CoverImage,
				"genres":     r.Genre,
				"styles":     r.Style,
			},
		})
	}

	return candidates, nil
}

// GetReleaseDetails fetches detailed metadata for a Discogs release
func (p *DiscogsProvider) GetReleaseDetails(externalID string) (*models.MetadataCandidate, error) {
	cfg := config.Get()
	hasKeySecret := cfg.DiscogsKey != "" && cfg.DiscogsSecret != ""
	hasToken := cfg.DiscogsToken != ""

	if !hasKeySecret && !hasToken {
		return nil, fmt.Errorf("Discogs credentials not configured")
	}

	params := url.Values{}
	// Use token if Key/Secret are not available
	if !hasKeySecret && hasToken {
		params.Set("token", cfg.DiscogsToken)
	}

	req, err := p.newRequest("GET", "/releases/"+externalID, params)
	if err != nil {
		return nil, NewProviderError("Discogs", err)
	}

	// Add Authorization header if Key/Secret are available
	if hasKeySecret {
		req.Header.Set("Authorization", fmt.Sprintf("Discogs key=%s, secret=%s", cfg.DiscogsKey, cfg.DiscogsSecret))
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, NewProviderError("Discogs", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, NewProviderError("Discogs", fmt.Errorf("API error: %s", resp.Status))
	}

	var release struct {
		ID      int    `json:"id"`
		Title   string `json:"title"`
		Year    int    `json:"year"`
		Country string `json:"country"`
		Label   []struct {
			Name string `json:"name"`
		} `json:"label"`
		Genre     []string `json:"genre"`
		Style     []string `json:"style"`
		Tracklist []struct {
			Title    string `json:"title"`
			Position string `json:"position"`
			Duration string `json:"duration"`
			Type_    string `json:"type_"`
		} `json:"tracklist"`
		Artists []struct {
			Name string `json:"name"`
		} `json:"artists"`
		Images []struct {
			Type   string `json:"type"`
			URI    string `json:"uri"`
			Height int    `json:"height"`
			Width  int    `json:"width"`
		} `json:"images"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, NewProviderError("Discogs", err)
	}

	// Build artist name
	artistName := ""
	if len(release.Artists) > 0 {
		artistName = release.Artists[0].Name
	}

	// Build track list
	var tracks []models.Song
	for _, track := range release.Tracklist {
		if track.Type_ != "track" {
			continue
		}

		// Parse duration (format: MM:SS or HH:MM:SS)
		duration := parseDuration(track.Duration)

		// Parse track number from position
		trackNum := track.Position
		if idx := findLastIndex(track.Position, "."); idx != -1 {
			trackNum = track.Position[idx+1:]
		}

		// Parse disc number from position (format: A-1 or 1-1)
		discNum := "1"
		if idx := findLastIndex(track.Position, "-"); idx != -1 && idx > 0 {
			discNum = track.Position[:idx]
			// Convert letter discs to numbers (A=1, B=2, etc.)
			if len(discNum) == 1 && discNum[0] >= 'A' && discNum[0] <= 'Z' {
				discNum = fmt.Sprintf("%d", int(discNum[0]-'A'+1))
			}
		}

		tracks = append(tracks, models.Song{
			Title:    track.Title,
			Artist:   artistName,
			Album:    release.Title,
			Track:    trackNum,
			Disc:     discNum,
			Duration: duration,
		})
	}

	// Extract genres
	genre := ""
	if len(release.Genre) > 0 {
		genre = release.Genre[0]
	}

	// Extract label
	label := ""
	if len(release.Label) > 0 {
		label = release.Label[0].Name
	}

	// Find primary image
	coverImage := ""
	for _, img := range release.Images {
		if img.Type == "primary" {
			coverImage = img.URI
			break
		}
	}
	if coverImage == "" && len(release.Images) > 0 {
		coverImage = release.Images[0].URI
	}

	return &models.MetadataCandidate{
		Source:     "Discogs",
		Artist:     artistName,
		Album:      release.Title,
		Year:       fmt.Sprintf("%d", release.Year),
		Genre:      genre,
		Tracks:     tracks,
		ExternalID: strconv.Itoa(release.ID),
		Metadata: map[string]interface{}{
			"country":    release.Country,
			"label":      label,
			"genres":     release.Genre,
			"styles":     release.Style,
			"coverImage": coverImage,
		},
	}, nil
}

// GetCoverArt fetches cover art candidates from Discogs
func (p *DiscogsProvider) GetCoverArt(artist, album string) ([]models.CoverArtCandidate, error) {
	candidates, err := p.Search(artist, album)
	if err != nil {
		return nil, err
	}

	var results []models.CoverArtCandidate
	// For each candidate, if we want dimensions we'd need to fetch details for EACH.
	// To avoid rate limiting, let's just use what we have in Search or maybe just fetch details for the TOP result.
	// Actually, Discogs Search results include 'cover_image' but NO dimensions.
	// But our API allows choosing.

	for _, c := range candidates {
		if coverImage, ok := c.Metadata["coverImage"].(string); ok && coverImage != "" {
			candidate := models.CoverArtCandidate{
				Source:    "Discogs",
				URL:       coverImage,
				Thumbnail: coverImage,
				Size:      "full",
			}

			// If we ever add dimensions to Metadata in Search, we'd pick them up here.
			// Discogs doesn't provide them in search results.

			results = append(results, candidate)
		}
		if thumb, ok := c.Metadata["thumbnail"].(string); ok && thumb != "" {
			results = append(results, models.CoverArtCandidate{
				Source:    "Discogs",
				URL:       thumb,
				Thumbnail: thumb,
				Size:      "small",
			})
		}
	}

	return results, nil
}

// newRequest creates a new HTTP request with proper headers
func (p *DiscogsProvider) newRequest(method, path string, params url.Values) (*http.Request, error) {
	url := discogsAPIURL + path + "?" + params.Encode()
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", discogsUserAgent)
	return req, nil
}

// Helper function to find last index of a substring
func findLastIndex(s, substr string) int {
	for i := len(s) - len(substr); i >= 0; i-- {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// Helper function to parse duration string to seconds
func parseDuration(dur string) int {
	parts := splitDuration(dur, ":")
	switch len(parts) {
	case 3: // HH:MM:SS
		h, _ := strconv.Atoi(parts[0])
		m, _ := strconv.Atoi(parts[1])
		s, _ := strconv.Atoi(parts[2])
		return h*3600 + m*60 + s
	case 2: // MM:SS
		m, _ := strconv.Atoi(parts[0])
		s, _ := strconv.Atoi(parts[1])
		return m*60 + s
	case 1: // SS
		s, _ := strconv.Atoi(parts[0])
		return s
	}
	return 0
}

func splitDuration(s, sep string) []string {
	var result []string
	start := 0
	for i := 0; i <= len(s)-len(sep); i++ {
		if s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
		}
	}
	result = append(result, s[start:])
	return result
}
