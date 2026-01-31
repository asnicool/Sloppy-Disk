package metadata

import (
	"encoding/json"
	"fmt"
	"log"
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
	log.Printf("[MUSICBRAINZ] Search called with artist='%s', album='%s'", artist, album)
	
	// Rate limiting
	if time.Since(p.lastReq) < musicBrainzRateLimit {
		sleepTime := musicBrainzRateLimit - time.Since(p.lastReq)
		log.Printf("[MUSICBRAINZ] Rate limiting: sleeping for %v", sleepTime)
		time.Sleep(sleepTime)
	}

	query := fmt.Sprintf(`artist:"%s" AND release:"%s"`, artist, album)
	params := url.Values{
		"query":  {query},
		"type":   {"album"},
		"limit":  {"20"},
		"fmt":    {"json"},
	}

	log.Printf("[MUSICBRAINZ] Query URL params: query='%s', type='%s', limit='%s'", query, params.Get("type"), params.Get("limit"))

	req, err := p.newRequest("GET", "/release-group/", params)
	if err != nil {
		log.Printf("[MUSICBRAINZ] Error creating request: %v", err)
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

	log.Printf("[MUSICBRAINZ] Sending request to: %s", req.URL.String())
	p.lastReq = time.Now()
	resp, err := p.client.Do(req)
	if err != nil {
		log.Printf("[MUSICBRAINZ] HTTP request failed: %v", err)
		return nil, NewProviderError("MusicBrainz", err)
	}
	defer resp.Body.Close()

	log.Printf("[MUSICBRAINZ] Response status: %d %s", resp.StatusCode, resp.Status)
	if resp.StatusCode != http.StatusOK {
		log.Printf("[MUSICBRAINZ] Non-OK status code received")
		return nil, NewProviderError("MusicBrainz", fmt.Errorf("API returned status %d", resp.StatusCode))
	}

	log.Printf("[MUSICBRAINZ] Parsing JSON response...")
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("[MUSICBRAINZ] JSON decode error: %v", err)
		return nil, NewProviderError("MusicBrainz", err)
	}

	log.Printf("[MUSICBRAINZ] API returned %d release groups (count=%d)", len(result.ReleaseGroups), result.Count)

	var candidates []models.MetadataCandidate
	for i, rg := range result.ReleaseGroups {
		log.Printf("[MUSICBRAINZ] Release group %d: Title='%s', Date='%s', ID='%s'", i, rg.Title, rg.FirstReleaseDate, rg.ID)
		if len(rg.ArtistCredits) > 0 {
			log.Printf("[MUSICBRAINZ]   Artist credits: %d artists", len(rg.ArtistCredits))
			for j, ac := range rg.ArtistCredits {
				log.Printf("[MUSICBRAINZ]     Artist %d: '%s' (ID: %s)", j, ac.Artist.Name, ac.Artist.ID)
			}
		}
		if len(rg.Releases) > 0 {
			log.Printf("[MUSICBRAINZ]   Releases: %d available", len(rg.Releases))
		}
		artistName := artist
		if len(rg.ArtistCredits) > 0 {
			artistName = rg.ArtistCredits[0].Artist.Name
		}

		releaseID := ""
		if len(rg.Releases) > 0 {
			releaseID = rg.Releases[0].ID
		}

		// Handle potential panic from [:4] on short date strings
		year := "????"
		if len(rg.FirstReleaseDate) >= 4 {
			year = rg.FirstReleaseDate[:4]
		}

		log.Printf("[MUSICBRAINZ] Creating candidate: Artist='%s', Album='%s', Year='%s', ReleaseID='%s'", 
			artistName, rg.Title, year, releaseID)

		candidates = append(candidates, models.MetadataCandidate{
			Source:     "MusicBrainz",
			Artist:     artistName,
			Album:      rg.Title,
			Year:       year,
			ExternalID: rg.ID,
			Metadata: map[string]interface{}{
				"releaseGroupID": rg.ID,
				"releaseID":      releaseID,
			},
		})
	}

	log.Printf("[MUSICBRAINZ] Returning %d candidates", len(candidates))
	return candidates, nil
}

// GetReleaseDetails fetches detailed metadata for a MusicBrainz release
func (p *MusicBrainzProvider) GetReleaseDetails(externalID string) (*models.MetadataCandidate, error) {
	log.Printf("[MUSICBRAINZ] GetReleaseDetails called with externalID='%s'", externalID)
	
	// Rate limiting
	if time.Since(p.lastReq) < musicBrainzRateLimit {
		sleepTime := musicBrainzRateLimit - time.Since(p.lastReq)
		log.Printf("[MUSICBRAINZ] Rate limiting: sleeping for %v", sleepTime)
		time.Sleep(sleepTime)
	}

	// The externalID is a release-group ID, but we need a release ID to get details
	// We'll use a different approach: search for the release-group again to get a release ID
	log.Printf("[MUSICBRAINZ] Fetching release-group to get release ID: '%s'", externalID)
	
	params := url.Values{
		"fmt": {"json"},
	}

	req, err := p.newRequest("GET", "/release-group/"+externalID, params)
	if err != nil {
		log.Printf("[MUSICBRAINZ] Error creating request for release-group: %v", err)
		return nil, NewProviderError("MusicBrainz", err)
	}

	log.Printf("[MUSICBRAINZ] Sending request to: %s", req.URL.String())
	p.lastReq = time.Now()
	resp, err := p.client.Do(req)
	if err != nil {
		log.Printf("[MUSICBRAINZ] HTTP request failed: %v", err)
		return nil, NewProviderError("MusicBrainz", err)
	}
	defer resp.Body.Close()

	log.Printf("[MUSICBRAINZ] Response status: %d %s", resp.StatusCode, resp.Status)
	if resp.StatusCode != http.StatusOK {
		log.Printf("[MUSICBRAINZ] Non-OK status code received when fetching release-group")
		return nil, NewProviderError("MusicBrainz", fmt.Errorf("API returned status %d", resp.StatusCode))
	}

	var releaseGroup struct {
		ID       string `json:"id"`
		Title    string `json:"title"`
		Releases []struct {
			ID string `json:"id"`
		} `json:"releases"`
	}

	log.Printf("[MUSICBRAINZ] Parsing JSON response...")
	if err := json.NewDecoder(resp.Body).Decode(&releaseGroup); err != nil {
		log.Printf("[MUSICBRAINZ] JSON decode error: %v", err)
		return nil, NewProviderError("MusicBrainz", err)
	}

	if len(releaseGroup.Releases) == 0 {
		log.Printf("[MUSICBRAINZ] No releases found for release-group '%s'", externalID)
		return nil, NewProviderError("MusicBrainz", fmt.Errorf("no releases found"))
	}

	releaseID := releaseGroup.Releases[0].ID
	log.Printf("[MUSICBRAINZ] Found release ID: '%s', now fetching release details", releaseID)

	// Now fetch the actual release details
	params = url.Values{
		"inc":  {"recordings+artist-credits+release-groups"},
		"fmt":  {"json"},
	}

	req, err = p.newRequest("GET", "/release/"+releaseID, params)
	if err != nil {
		log.Printf("[MUSICBRAINZ] Error creating request for release: %v", err)
		return nil, NewProviderError("MusicBrainz", err)
	}

	log.Printf("[MUSICBRAINZ] Sending request to: %s", req.URL.String())
	p.lastReq = time.Now()
	resp, err = p.client.Do(req)
	if err != nil {
		log.Printf("[MUSICBRAINZ] HTTP request failed: %v", err)
		return nil, NewProviderError("MusicBrainz", err)
	}
	defer resp.Body.Close()

	log.Printf("[MUSICBRAINZ] Response status: %d %s", resp.StatusCode, resp.Status)
	if resp.StatusCode != http.StatusOK {
		log.Printf("[MUSICBRAINZ] Non-OK status code received when fetching release")
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

	log.Printf("[MUSICBRAINZ] Parsing JSON response...")
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		log.Printf("[MUSICBRAINZ] JSON decode error: %v", err)
		return nil, NewProviderError("MusicBrainz", err)
	}

	log.Printf("[MUSICBRAINZ] Release details: Title='%s', Date='%s', Tracks=%d", release.Title, release.Date, countTracks(release.Media))

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

	// Handle potential panic from [:4] on short date strings
	year := "????"
	if len(release.Date) >= 4 {
		year = release.Date[:4]
	}

	log.Printf("[MUSICBRAINZ] Returning %d tracks for release", len(tracks))
	return &models.MetadataCandidate{
		Source:  "MusicBrainz",
		Artist:  artistName.String(),
		Album:   release.Title,
		Year:    year,
		Genre:   strings.Join(genres, "; "),
		Tracks:  tracks,
		ExternalID: externalID,
		Metadata: map[string]interface{}{
			"releaseGroupID":   release.ReleaseGroup.ID,
			"releaseGroupType": release.ReleaseGroup.Type,
			"country":          release.Country,
			"barcode":          release.Barcode,
			"label":            label,
			"releaseID":        release.ID,
		},
	}, nil
}

// countTracks counts total tracks across all media
func countTracks(media []struct {
	Position int `json:"position"`
	Format   string `json:"format"`
	Tracks   []struct {
		ID          string `json:"id"`
		Title       string `json:"title"`
		Length      int    `json:"length"`
		TrackNumber int    `json:"number"`
	} `json:"tracks"`
}) int {
	total := 0
	for _, m := range media {
		total += len(m.Tracks)
	}
	return total
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