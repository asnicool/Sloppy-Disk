package metadata

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"sloppy-disk/internal/models"

	"github.com/michiwend/gomusicbrainz"
)

const (
	musicBrainzAPIURL    = "https://musicbrainz.org"
	musicBrainzCoverURL  = "https://coverartarchive.org"
	musicBrainzUserAgent = "sloppy-disk"
	musicBrainzVersion   = "1.0"
	musicBrainzContact   = "contact@example.com"
)

// Ensure valid transport is set globally for the library
func init() {
	// Configure robust transport (IPv4 only, No HTTP/2, Retries via library loop)
	robustTransport := &http.Transport{
		MaxIdleConns:          10,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   20 * time.Second,
		ResponseHeaderTimeout: 20 * time.Second,
		DisableKeepAlives:     false,
		ForceAttemptHTTP2:     false,
		TLSNextProto:          make(map[string]func(string, *tls.Conn) http.RoundTripper),
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
			MinVersion:         tls.VersionTLS12,
		},
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			dialer := &net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}
			return dialer.DialContext(ctx, "tcp4", addr)
		},
	}
	http.DefaultTransport = robustTransport
}

// MusicBrainzProvider implements the Provider interface for MusicBrainz
type MusicBrainzProvider struct {
	client *gomusicbrainz.WS2Client
}

// NewMusicBrainzProvider creates a new MusicBrainz provider
func NewMusicBrainzProvider() *MusicBrainzProvider {
	client, err := gomusicbrainz.NewWS2Client(
		musicBrainzAPIURL,
		musicBrainzUserAgent,
		musicBrainzVersion,
		musicBrainzContact,
	)
	if err != nil {
		log.Printf("[MUSICBRAINZ] Failed to create client: %v", err)
		return nil
	}

	return &MusicBrainzProvider{
		client: client,
	}
}

// Name returns the provider name
func (p *MusicBrainzProvider) Name() string {
	return "MusicBrainz"
}

// Search searches for releases on MusicBrainz
func (p *MusicBrainzProvider) Search(artist, album string) ([]models.MetadataCandidate, error) {
	query := fmt.Sprintf(`artist:"%s" AND release:"%s" AND primarytype:"Album"`, artist, album)
	log.Printf("[MUSICBRAINZ] Search called with query='%s'", query)

	var resp *gomusicbrainz.ReleaseGroupSearchResponse
	var err error

	// Retry loop
	for i := 0; i < 5; i++ {
		resp, err = p.client.SearchReleaseGroup(query, 20, -1)
		if err == nil {
			break
		}
		log.Printf("[MUSICBRAINZ] Search attempt %d failed: %v. Retrying...", i+1, err)
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		return nil, NewProviderError("MusicBrainz", err)
	}

	var candidates []models.MetadataCandidate
	for _, rg := range resp.ReleaseGroups {
		year := "????"
		if !rg.FirstReleaseDate.IsZero() {
			year = fmt.Sprintf("%d", rg.FirstReleaseDate.Year())
		}

		artistName := artist
		if len(rg.ArtistCredit.NameCredits) > 0 {
			artistName = rg.ArtistCredit.NameCredits[0].Artist.Name
		}

		// The library might not fetch releases in SearchReleaseGroup, so we might need a separate lookup or just use ReleaseGroup ID
		// But our data model expects a release ID for detailed lookup later.
		// SearchReleaseGroup response usually includes some releases if we ask? Use Lookup for details.

		candidates = append(candidates, models.MetadataCandidate{
			Source:     "MusicBrainz",
			Artist:     artistName,
			Album:      rg.Title,
			Year:       year,
			ExternalID: string(rg.ID),
			Metadata: map[string]interface{}{
				"releaseGroupID": string(rg.ID),
			},
		})
	}

	log.Printf("[MUSICBRAINZ] Returning %d candidates", len(candidates))
	return candidates, nil
}

// GetReleaseDetails fetches detailed metadata for a MusicBrainz release
func (p *MusicBrainzProvider) GetReleaseDetails(externalID string) (*models.MetadataCandidate, error) {
	// externalID is a ReleaseGroup ID from Search
	log.Printf("[MUSICBRAINZ] GetReleaseDetails called with externalID='%s'", externalID)

	// 1. We have a ReleaseGroup ID. We need to find the best Release (album) in this group.
	// We'll perform a lookup on the ReleaseGroup and include releases.
	var rg *gomusicbrainz.ReleaseGroup
	var err error

	for i := 0; i < 5; i++ {
		rg, err = p.client.LookupReleaseGroup(gomusicbrainz.MBID(externalID), "releases")
		if err == nil {
			break
		}
		log.Printf("[MUSICBRAINZ] LookupReleaseGroup attempt %d failed: %v", i+1, err)
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		return nil, NewProviderError("MusicBrainz", err)
	}

	if len(rg.Releases) == 0 {
		return nil, NewProviderError("MusicBrainz", fmt.Errorf("no releases found for group"))
	}

	// Pick the first release (simplification, ideally we'd filter by country/date)
	releaseID := rg.Releases[0].ID

	// 2. Lookup the specific Release to get tracks
	var release *gomusicbrainz.Release
	for i := 0; i < 5; i++ {
		release, err = p.client.LookupRelease(releaseID, "recordings", "artist-credits") // 'recordings' gives tracks
		if err == nil {
			break
		}
		log.Printf("[MUSICBRAINZ] LookupRelease attempt %d failed: %v", i+1, err)
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		return nil, NewProviderError("MusicBrainz", err)
	}

	var tracks []models.Song
	for _, medium := range release.Mediums {
		for _, track := range medium.Tracks {
			// Resolve artist name
			trackArtist := artistCreditString(track.Recording.ArtistCredit)
			if trackArtist == "" {
				trackArtist = artistCreditString(release.ArtistCredit)
			}

			tracks = append(tracks, models.Song{
				Title:    track.Recording.Title,
				Artist:   trackArtist,
				Album:    release.Title,
				Track:    track.Number,
				Disc:     fmt.Sprintf("%d", medium.Position),
				Duration: track.Length / 1000,
			})
		}
	}

	// Genre? Library doesn't seem to have easy genre support in core structs based on what I saw, ignoring for now or fetching tags if possible.
	// The library `Tag` struct exists.

	year := "????"
	if !release.Date.IsZero() {
		year = fmt.Sprintf("%d", release.Date.Year())
	}

	artistName := artistCreditString(release.ArtistCredit)

	return &models.MetadataCandidate{
		Source:     "MusicBrainz",
		Artist:     artistName,
		Album:      release.Title,
		Year:       year,
		Genre:      "", // Skip for now
		Tracks:     tracks,
		ExternalID: externalID,
		Metadata: map[string]interface{}{
			"releaseGroupID": string(rg.ID),
			"releaseID":      string(release.ID),
			"barcode":        release.Barcode,
		},
	}, nil
}

func artistCreditString(ac gomusicbrainz.ArtistCredit) string {
	var sb strings.Builder
	for i, nc := range ac.NameCredits {
		if i > 0 {
			sb.WriteString(" / ")
		}
		sb.WriteString(nc.Artist.Name)
	}
	return sb.String()
}

// GetArtistImage fetches artist images from MusicBrainz
// Note: MusicBrainz doesn't directly host artist images, but provides links to external sources
// This implementation is a placeholder - Discogs is the primary source for artist images
func (p *MusicBrainzProvider) GetArtistImage(artistName string) ([]models.ArtistImageCandidate, error) {
	// MusicBrainz primarily provides metadata and links to external resources
	// The main artist images come from Discogs
	// For now, return empty - Discogs is the primary source
	return []models.ArtistImageCandidate{}, nil
}

// GetCoverArt fetches cover art from Cover Art Archive
func (p *MusicBrainzProvider) GetCoverArt(artist, album string) ([]models.CoverArtCandidate, error) {
	// Re-using exiting Search logic
	candidates, err := p.Search(artist, album)
	if err != nil {
		return nil, err
	}

	if len(candidates) == 0 {
		return []models.CoverArtCandidate{}, nil
	}

	// Search gave us a ReleaseGroup ID (ExternalID).
	// We want to find ALL releases in this group that have cover art.
	rgID := candidates[0].ExternalID

	// Lookup ReleaseGroup with "releases" inc to get the list of releases
	rg, err := p.client.LookupReleaseGroup(gomusicbrainz.MBID(rgID), "releases")
	if err != nil {
		return nil, err
	}

	var results []models.CoverArtCandidate
	seen := make(map[string]bool)

	// Iterate over ALL releases in the group to find unique cover art
	for _, release := range rg.Releases {
		// The library version we use is old and doesn't have CoverArtArchive field on Release struct.
		// We optimistically add the candidate. The frontend will handle 404s if the image doesn't exist.

		releaseID := string(release.ID)

		// We will try to fetch the "front" image directly from CAA.
		// We use the direct URL pattern which redirects to the best available image.
		// https://coverartarchive.org/release/{release-id}/front
		frontURL := fmt.Sprintf("%s/release/%s/front", musicBrainzCoverURL, releaseID)

		// To avoid duplicates (if multiple releases share the same cover image ID internally,
		// the URL will still be unique by release ID, but the visual content might be the same.
		// However, without downloading we can't be sure.
		// But often different releases have slightly different covers (remaster, different region).
		// We will treat them as separate candidates for now, as users might prefer one over another.

		// Optimization: Check if we haven't already added this exact URL (unlikely to happen with release ID in URL)
		if seen[frontURL] {
			continue
		}
		seen[frontURL] = true

		// Create a candidate
		// We use the "front" endpoint which is a 307 redirect to the actual file.
		// For the thumbnail, we can append "-250" or "-500" to the *redirected* URL,
		// but standard CAA behavior for /front/{size} is:
		// /release/{mbid}/front-250
		// /release/{mbid}/front-500
		thumbURL := fmt.Sprintf("%s/release/%s/front-250", musicBrainzCoverURL, releaseID)

		results = append(results, models.CoverArtCandidate{
			Source:    "MusicBrainz",
			URL:       frontURL,
			Thumbnail: thumbURL,
			Size:      "full", // We don't know the exact dimensions without fetching
		})

		// Limit the number of candidates to avoid overwhelming the UI
		if len(results) >= 20 {
			break
		}
	}

	if len(results) == 0 {
		// Fallback: If no release explicitly flagged "Front", try the first one anyway just in case
		if len(rg.Releases) > 0 {
			firstReleaseID := string(rg.Releases[0].ID)
			frontURL := fmt.Sprintf("%s/release/%s/front", musicBrainzCoverURL, firstReleaseID)
			return []models.CoverArtCandidate{
				{
					Source:    "MusicBrainz",
					URL:       frontURL,
					Thumbnail: frontURL + "-250",
					Size:      "full",
				},
			}, nil
		}
		return []models.CoverArtCandidate{}, nil
	}

	return results, nil
}
