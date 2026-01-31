package metadata

import (
	"context"
	"log"
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
	log.Printf("[METADATA SEARCH] Starting search - Artist: '%s', Album: '%s'", artist, album)
	log.Printf("[METADATA SEARCH] Available providers: %d", len(a.providers))
	for _, p := range a.providers {
		log.Printf("[METADATA SEARCH]   - %s", p.Name())
	}

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

	log.Printf("[METADATA SEARCH] Active providers after filtering: %d", len(activeProviders))
	for _, p := range activeProviders {
		log.Printf("[METADATA SEARCH]   - %s", p.Name())
	}

	if len(activeProviders) == 0 {
		log.Printf("[METADATA SEARCH] No active providers, returning empty results")
		return []models.MetadataCandidate{}, nil
	}

	// Search all providers in parallel
	var wg sync.WaitGroup
	mu := sync.Mutex{}
	results := make([]models.MetadataCandidate, 0)

	log.Printf("[METADATA SEARCH] Starting parallel search across %d providers", len(activeProviders))
	
	for _, p := range activeProviders {
		wg.Add(1)
		go func(provider Provider) {
			defer wg.Done()

			log.Printf("[METADATA SEARCH] [%s] Starting search", provider.Name())
			candidates, err := provider.Search(artist, album)
			if err != nil {
				log.Printf("[METADATA SEARCH] [%s] Error: %v", provider.Name(), err)
				return
			}

			log.Printf("[METADATA SEARCH] [%s] Found %d candidates", provider.Name(), len(candidates))
			for i, c := range candidates {
				log.Printf("[METADATA SEARCH] [%s]   Candidate %d: Artist='%s', Album='%s', Year='%s', ExternalID='%s'", 
					provider.Name(), i, c.Artist, c.Album, c.Year, c.ExternalID)
			}

			mu.Lock()
			for i := range candidates {
				// Calculate confidence score
				candidates[i].Confidence = calculateConfidence(candidates[i], artist, album)
				log.Printf("[METADATA SEARCH] [%s]   Candidate %d confidence: %.2f", 
					provider.Name(), i, candidates[i].Confidence)
			}
			results = append(results, candidates...)
			mu.Unlock()
		}(p)
	}

	wg.Wait()
	log.Printf("[METADATA SEARCH] All providers completed. Total candidates before deduplication: %d", len(results))

	// Deduplicate by external ID
	results = deduplicate(results)
	log.Printf("[METADATA SEARCH] After deduplication: %d candidates", len(results))

	// Sort by confidence (highest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Confidence > results[j].Confidence
	})

	// Log final results
	log.Printf("[METADATA SEARCH] Final results (sorted by confidence):")
	for i, r := range results {
		log.Printf("[METADATA SEARCH]   %d. [%.2f] %s - %s (%s) from %s", 
			i+1, r.Confidence, r.Artist, r.Album, r.Year, r.Source)
	}

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
	normalizedCandidateArtist := normalizeString(candidate.Artist)
	normalizedQueryArtist := normalizeString(queryArtist)
	artistScore := stringSimilarity(normalizedCandidateArtist, normalizedQueryArtist)
	score += artistScore * 30

	// Album similarity (0-30 points)
	normalizedCandidateAlbum := normalizeString(candidate.Album)
	normalizedQueryAlbum := normalizeString(queryAlbum)
	albumScore := stringSimilarity(normalizedCandidateAlbum, normalizedQueryAlbum)
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

	log.Printf("[CONFIDENCE] Artist: '%s' vs '%s' = %.4f (%.1f pts)", 
		normalizedCandidateArtist, normalizedQueryArtist, artistScore, artistScore*30)
	log.Printf("[CONFIDENCE] Album: '%s' vs '%s' = %.4f (%.1f pts)", 
		normalizedCandidateAlbum, normalizedQueryAlbum, albumScore, albumScore*30)
	log.Printf("[CONFIDENCE] Source bonus: %s = %.1f pts", candidate.Source, score-50-artistScore*30-albumScore*30)

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
