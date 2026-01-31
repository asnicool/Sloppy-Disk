package metadata

import (
	"context"
	"sort"
	"strings"
	"sync"
	"unicode"

	"mpd-client-modern/internal/config"
	"mpd-client-modern/internal/models"
)

// Aggregator handles searching across multiple metadata providers
type Aggregator struct {
	providers []Provider
}

// NewAggregator creates a new metadata aggregator
func NewAggregator() *Aggregator {
	cfg := config.Get()
	providers := []Provider{}

	// Only add providers that are enabled in the config
	if cfg.MusicBrainzEnabled {
		providers = append(providers, NewMusicBrainzProvider())
	}
	if cfg.DiscogsEnabled {
		providers = append(providers, NewDiscogsProvider())
	}
	if cfg.FreeDBEnabled {
		providers = append(providers, NewFreeDBProvider())
	}
	if cfg.AlbumArtEnabled {
		providers = append(providers, NewAlbumArtProvider())
	}

	return &Aggregator{providers: providers}
}

// AddProvider adds a provider to the aggregator
func (a *Aggregator) AddProvider(p Provider) {
	a.providers = append(a.providers, p)
}

// Search searches all providers in parallel and aggregates results
func (a *Aggregator) Search(ctx context.Context, artist, album string, providers []string) ([]models.MetadataCandidate, error) {
	// Filter providers
	var activeProviders []Provider
	providerSet := make(map[string]bool)
	for _, p := range providers {
		providerSet[strings.ToLower(p)] = true
	}

	for _, p := range a.providers {
		if len(providers) == 0 || providerSet[strings.ToLower(p.Name())] {
			activeProviders = append(activeProviders, p)
		}
	}

	if len(activeProviders) == 0 {
		return []models.MetadataCandidate{}, nil
	}

	// Search all providers in parallel
	var wg sync.WaitGroup
	mu := sync.Mutex{}
	results := make([]models.MetadataCandidate, 0)

	for _, p := range activeProviders {
		wg.Add(1)
		go func(provider Provider) {
			defer wg.Done()

			candidates, err := provider.Search(artist, album)
			if err != nil {
				// Log error but continue
				return
			}

			mu.Lock()
			for i := range candidates {
				// Calculate confidence score
				candidates[i].Confidence = calculateConfidence(candidates[i], artist, album)
			}
			results = append(results, candidates...)
			mu.Unlock()
		}(p)
	}

	wg.Wait()

	// Deduplicate by external ID
	results = deduplicate(results)

	// Sort by confidence (highest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Confidence > results[j].Confidence
	})

	return results, nil
}

// GetReleaseDetails fetches detailed metadata from the appropriate provider
func (a *Aggregator) GetReleaseDetails(ctx context.Context, source, externalID string) (*models.MetadataCandidate, error) {
	for _, p := range a.providers {
		if strings.ToLower(p.Name()) == strings.ToLower(source) {
			return p.GetReleaseDetails(externalID)
		}
	}
	return nil, nil
}

// SearchCoverArt searches all providers for cover art
func (a *Aggregator) SearchCoverArt(ctx context.Context, artist, album string) ([]models.CoverArtCandidate, error) {
	var wg sync.WaitGroup
	mu := sync.Mutex{}
	results := make([]models.CoverArtCandidate, 0)

	for _, p := range a.providers {
		wg.Add(1)
		go func(provider Provider) {
			defer wg.Done()

			candidates, err := provider.GetCoverArt(artist, album)
			if err != nil {
				return
			}

			mu.Lock()
			results = append(results, candidates...)
			mu.Unlock()
		}(p)
	}

	wg.Wait()

	// Remove duplicates
	results = deduplicateCoverArt(results)

	return results, nil
}

// calculateConfidence calculates a confidence score for a candidate
func calculateConfidence(candidate models.MetadataCandidate, queryArtist, queryAlbum string) float64 {
	var score float64 = 50 // Base score

	// Artist similarity (0-30 points)
	artistScore := stringSimilarity(normalizeString(candidate.Artist), normalizeString(queryArtist))
	score += artistScore * 30

	// Album similarity (0-30 points)
	albumScore := stringSimilarity(normalizeString(candidate.Album), normalizeString(queryAlbum))
	score += albumScore * 30

	// Source reliability bonus (0-10 points)
	switch candidate.Source {
	case "MusicBrainz":
		score += 10
	case "Discogs":
		score += 7
	default:
		score += 5
	}

	return score
}

// normalizeString normalizes a string for comparison
func normalizeString(s string) string {
	// Convert to lowercase
	s = strings.ToLower(s)
	// Remove diacritics
	s = removeDiacritics(s)
	// Remove special characters
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// removeDiacritics removes diacritical marks from a string
func removeDiacritics(s string) string {
	result := make([]rune, 0, len(s))
	for _, r := range s {
		switch r {
		case 'á', 'à', 'â', 'ã', 'ä', 'å':
			result = append(result, 'a')
		case 'é', 'è', 'ê', 'ë':
			result = append(result, 'e')
		case 'í', 'ì', 'î', 'ï':
			result = append(result, 'i')
		case 'ó', 'ò', 'ô', 'õ', 'ö', 'ø':
			result = append(result, 'o')
		case 'ú', 'ù', 'û', 'ü':
			result = append(result, 'u')
		case 'ý', 'ÿ':
			result = append(result, 'y')
		case 'ñ':
			result = append(result, 'n')
		case 'ç':
			result = append(result, 'c')
		case 'ß':
			result = append(result, 's')
		case 'đ':
			result = append(result, 'd')
		default:
			result = append(result, r)
		}
	}
	return string(result)
}

// stringSimilarity calculates Levenshtein-based similarity (0-1)
func stringSimilarity(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}
	if len(s1) == 0 || len(s2) == 0 {
		return 0.0
	}

	// Use simple word overlap for efficiency
	words1 := strings.Fields(s1)
	words2 := strings.Fields(s2)

	if len(words1) == 0 || len(words2) == 0 {
		return 0.0
	}

	// Count matching words
	matchCount := 0
	wordSet2 := make(map[string]bool)
	for _, w := range words2 {
		wordSet2[w] = true
	}
	for _, w := range words1 {
		if wordSet2[w] {
			matchCount++
		}
	}

	// Jaccard-like similarity
	union := len(words1) + len(words2) - matchCount
	if union == 0 {
		return 1.0
	}
	return float64(matchCount) / float64(union)
}

// deduplicate removes duplicate candidates based on external ID
func deduplicate(candidates []models.MetadataCandidate) []models.MetadataCandidate {
	seen := make(map[string]bool)
	result := make([]models.MetadataCandidate, 0, len(candidates))

	for _, c := range candidates {
		key := c.Source + ":" + c.ExternalID
		if !seen[key] {
			seen[key] = true
			result = append(result, c)
		}
	}

	return result
}

// deduplicateCoverArt removes duplicate cover art based on URL
func deduplicateCoverArt(candidates []models.CoverArtCandidate) []models.CoverArtCandidate {
	seen := make(map[string]bool)
	result := make([]models.CoverArtCandidate, 0, len(candidates))

	for _, c := range candidates {
		if !seen[c.URL] {
			seen[c.URL] = true
			result = append(result, c)
		}
	}

	return result
}
