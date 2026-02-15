package albumcache

import (
	"fmt"
	"log"
	"math/rand"
	"mpd-client-modern/internal/models"
	"mpd-client-modern/internal/mpd"
	"net/url"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/sahilm/fuzzy"
)

// AlbumCache holds the in-memory album database
type AlbumCache struct {
	albums       []models.Album
	searchSource *AlbumSource
	mu           sync.RWMutex

	randomBuffer []models.Album
	bufferMutex  sync.Mutex

	// Page Cache for sorted lists
	pageCache      map[string]cachedPage
	pageCacheMutex sync.RWMutex

	// Album Details Cache (tracklists + summary)
	detailsCache      map[string]cachedDetails
	detailsCacheMutex sync.RWMutex

	// Inflight enrichment tracking
	inflight      map[string]bool
	inflightMutex sync.Mutex

	// Prevent concurrent refreshes
	refreshMutex sync.Mutex
	refreshing  bool
}

type cachedDetails struct {
	Data      interface{}
	ExpiresAt time.Time
}

type cachedPage struct {
	Albums    []models.Album
	ExpiresAt time.Time
}

// AlbumSource implements fuzzy.Source
type AlbumSource struct {
	keywords []string
}

func (s *AlbumSource) String(i int) string {
	return s.keywords[i]
}

func (s *AlbumSource) Len() int {
	return len(s.keywords)
}

var (
	defaultCache *AlbumCache
	once         sync.Once
)

// GetCache returns the singleton cache instance
func GetCache() *AlbumCache {
	once.Do(func() {
		defaultCache = &AlbumCache{
			albums:       []models.Album{},
			pageCache:    make(map[string]cachedPage),
			detailsCache: make(map[string]cachedDetails),
			inflight:     make(map[string]bool),
		}
	})
	return defaultCache
}

// Refresh rebuilds the album cache from MPD
func (ac *AlbumCache) Refresh() error {
	// Prevent concurrent refreshes
	ac.refreshMutex.Lock()
	if ac.refreshing {
		ac.refreshMutex.Unlock()
		log.Println("Album cache refresh already in progress, skipping...")
		return nil
	}
	ac.refreshing = true
	ac.refreshMutex.Unlock()

	// Ensure we clear the refreshing flag even if there's an error
	defer func() {
		ac.refreshMutex.Lock()
		ac.refreshing = false
		ac.refreshMutex.Unlock()
	}()

	client := mpd.GetClient()
	if err := client.Execute(func(conn *mpd.Connection) error {
		return conn.EnsureConnection()
	}); err != nil {
		return err
	}

	log.Println("Starting album cache refresh...")
	start := time.Now()

	// 1. Get all album keys
	// We fetch all at once since MPD 'list' command windowing can be problematic
	allKeys, err := client.GetAllAlbumKeys()
	if err != nil {
		return err
	}

	log.Printf("Fetched %d album keys in %v", len(allKeys), time.Since(start))

	var tempAlbums []models.Album
	for _, key := range allKeys {
		id := fmt.Sprintf("%x", fmt.Sprintf("%s|%s", key.AlbumArtist, key.Album))
		tempAlbums = append(tempAlbums, models.Album{
			ID:     id,
			Album:  key.Album,
			Artist: key.AlbumArtist,
			Date:   key.Date,
			Genre:  key.Genre,
		})
	}

	// 4. Sort
	sort.Slice(tempAlbums, func(i, j int) bool {
		if tempAlbums[i].Album != tempAlbums[j].Album {
			return tempAlbums[i].Album < tempAlbums[j].Album
		}
		return tempAlbums[i].Artist < tempAlbums[j].Artist
	})

	// 5. Build fuzzy index
	keywords := make([]string, len(tempAlbums))
	for i, a := range tempAlbums {
		// Only use Album and Artist for initial fast index
		keywords[i] = a.Album + " " + a.Artist
	}

	newSource := &AlbumSource{keywords: keywords}

	ac.mu.Lock()
	ac.albums = tempAlbums
	ac.searchSource = newSource
	ac.mu.Unlock()

	log.Printf("Refreshed cache with %d albums in %v", len(tempAlbums), time.Since(start))

	// Trigger initial buffer fill
	go ac.MaintainRandomBuffer()

	return nil
}

// EnrichAlbums takes a slice of albums and fetches their missing metadata from MPD
// NOTE: This function NO LONGER calls GetAlbumStats() to avoid overwhelming MPD
// Stats (TrackCount, Duration) will remain 0 and should be loaded opportunistically
func (ac *AlbumCache) EnrichAlbums(albums []models.Album) ([]models.Album, error) {
	if len(albums) == 0 {
		return albums, nil
	}

	client := mpd.GetClient()

	// Process in smaller batches to avoid overwhelming MPD
	// Max batch size of 3 albums for opportunistic enrichment
	const maxEnrichBatchSize = 3

	enriched := make([]models.Album, len(albums))

	// Process albums in batches
	for i := 0; i < len(albums); i += maxEnrichBatchSize {
		end := i + maxEnrichBatchSize
		if end > len(albums) {
			end = len(albums)
		}
		batch := albums[i:end]

		// Create batch keys for this batch
		var batchKeys []models.AlbumKey
		for _, a := range batch {
			batchKeys = append(batchKeys, models.AlbumKey{
				Album:       a.Album,
				AlbumArtist: a.Artist,
			})
		}

		log.Printf("[AlbumCache] Enriching batch %d-%d (%d albums)", i, end, len(batch))

		// REMOVED: GetAlbumStats - no longer called to avoid MPD overload
		// TrackCount and Duration will remain 0 until opportunistically loaded

		reps, err := client.GetAlbumRepresentatives(batchKeys)
		if err != nil {
			log.Printf("[AlbumCache] Failed to get reps for batch %d-%d: %v", i, end, err)
			// Continue with partially enriched data
			for j, a := range batch {
				enriched[i+j] = a
			}
			continue
		}

		// Apply enrichment to this batch (only representative data)
		for j, a := range batch {
			key := models.AlbumKey{Album: a.Album, AlbumArtist: a.Artist}
			rep := reps[key]

			enriched[i+j] = a
			// NOTE: TrackCount and Duration are NOT set (remain 0)
			// These will be loaded opportunistically when album details are viewed

			if rep != nil {
				// Only overwrite Date and Genre if representative has non-empty values
				// This preserves values from GetAllAlbumKeys() which may be more accurate
				if rep.Date != "" {
					enriched[i+j].Date = rep.Date
				}
				if rep.Genre != "" {
					enriched[i+j].Genre = rep.Genre
				}
				enriched[i+j].Path = rep.Path
				if enriched[i+j].Artist == "" {
					enriched[i+j].Artist = rep.Artist
				}
				// Fallback for empty album name: use parent directory
				if enriched[i+j].Album == "" && rep.Path != "" {
					dir := filepath.Base(filepath.Dir(rep.Path))
					if dir != "." && dir != "/" {
						enriched[i+j].Album = dir
					} else {
						enriched[i+j].Album = "Unknown Album"
					}
				}

				// Populate CoverURL
				if enriched[i+j].Path != "" {
					albumPath := filepath.Dir(enriched[i+j].Path)
					if albumPath == "." {
						albumPath = ""
					}
					// Escape path for URL, keeping slashes
					pathParts := strings.Split(albumPath, "/")
					for idx, part := range pathParts {
						pathParts[idx] = url.PathEscape(part)
					}
					escapedPath := strings.Join(pathParts, "/")
					enriched[i+j].CoverURL = fmt.Sprintf("/api/coverart/%s", escapedPath)
				}
			}

			// Ensure every album has an ID
			if enriched[i+j].ID == "" {
				enriched[i+j].ID = fmt.Sprintf("%x", fmt.Sprintf("%s|%s", enriched[i+j].Artist, enriched[i+j].Album))
			}
		}

		log.Printf("[AlbumCache] Completed batch %d-%d", i, end)
	}

	return enriched, nil
}

// GetRandomAlbums returns a random selection of albums (up to count) from the pre-cached buffer.
// If buffer is insufficient, it falls back to synchronous enrichment.
func (ac *AlbumCache) GetRandomAlbums(count int, enrich bool) ([]models.Album, error) {
	if count <= 0 {
		count = 36
	}

	result := make([]models.Album, 0, count)

	// Step 1: Drain buffer as much as possible
	ac.bufferMutex.Lock()
	bufferedCount := len(ac.randomBuffer)
	if bufferedCount > 0 {
		take := count
		if bufferedCount < take {
			take = bufferedCount
		}

		result = append(result, ac.randomBuffer[:take]...)
		ac.randomBuffer = ac.randomBuffer[take:] // Remove taken items
	}
	ac.bufferMutex.Unlock()

	// Trigger refill in background regardless, as we consumed items
	go ac.MaintainRandomBuffer()

	// Step 2: If we have enough, return
	if len(result) >= count {
		return result, nil
	}

	// Step 3: Fetch remainder synchronously
	remaining := count - len(result)
	log.Printf("Buffer provided %d items, fetching %d synchronously", len(result), remaining)

	ac.mu.RLock()
	total := len(ac.albums)
	if total == 0 {
		ac.mu.RUnlock()
		return result, nil // Return what we have
	}

	indices := make([]int, total)
	for i := range indices {
		indices[i] = i
	}
	rand.Shuffle(total, func(i, j int) { indices[i], indices[j] = indices[j], indices[i] })

	limit := remaining
	if limit > total {
		limit = total
	}

	selected := make([]models.Album, limit)
	for i := 0; i < limit; i++ {
		selected[i] = ac.albums[indices[i]]
	}
	ac.mu.RUnlock()

	// Enrich if needed (requested or if we want to mimic buffer behavior)
	// The buffer always contains enriched albums.
	if enrich {
		enriched, err := ac.EnrichAlbums(selected)
		if err != nil {
			return nil, err
		}
		// Corrected: append enriched albums to result, not selected
		result = append(result, enriched...)
	} else {
		result = append(result, selected...)
	}

	return result, nil
}

// MaintainRandomBuffer ensures the random buffer contains enough enriched albums
const BufferSize = 120
const RefillBatchSize = 5 // Small batch size to avoid blocking MPD for long periods

func (ac *AlbumCache) MaintainRandomBuffer() {
	// Try to acquire a "refill lock" to prevent multiple goroutines from refilling simultaneously
	// Since we don't have a dedicated lock field for this, we rely on checking buffer size inside loop

	// If buffer is very low at start, boost urgency

	for {
		ac.bufferMutex.Lock()
		currentSize := len(ac.randomBuffer)
		ac.bufferMutex.Unlock()

		if currentSize >= BufferSize {
			return
		}

		// Calculate needed for this iteration
		needed := RefillBatchSize
		// If current size is low (startup or drained), boost fetch size slightly to recover faster
		if currentSize < 10 {
			needed = 10 - currentSize
			if needed < RefillBatchSize {
				needed = RefillBatchSize
			}
		}

		if currentSize+needed > BufferSize {
			needed = BufferSize - currentSize
		}

		// Pick random items from main cache
		ac.mu.RLock()
		total := len(ac.albums)
		if total == 0 {
			ac.mu.RUnlock()
			return
		}

		indices := make([]int, total)
		for i := range indices {
			indices[i] = i
		}
		rand.Shuffle(total, func(i, j int) { indices[i], indices[j] = indices[j], indices[i] })

		limit := needed
		if limit > total {
			limit = total
		}

		candidates := make([]models.Album, limit)
		for i := 0; i < limit; i++ {
			candidates[i] = ac.albums[indices[i]]
		}
		ac.mu.RUnlock()

		// Enrich them (this takes time, ~1s for 5 items)
		enriched, err := ac.EnrichAlbums(candidates)
		if err != nil {
			log.Printf("Failed to enrich buffer items: %v", err)
			time.Sleep(1 * time.Second) // Backoff on error
			continue
		}

		// Append to buffer
		ac.bufferMutex.Lock()
		ac.randomBuffer = append(ac.randomBuffer, enriched...)
		newSize := len(ac.randomBuffer)
		ac.bufferMutex.Unlock()

		log.Printf("Random buffer appended %d items. Size: %d", len(enriched), newSize)

		// If full, stop
		if newSize >= BufferSize {
			return
		}

		// Yield/Sleep to allow other MPD commands to interleave
		// Increased from 50ms to 200ms to reduce MPD load during startup
		time.Sleep(200 * time.Millisecond)
	}
}

func (ac *AlbumCache) GetAlbumsPage(offset, limit int, sortMode string) ([]models.Album, int) {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	total := len(ac.albums)

	if offset >= total {
		return []models.Album{}, total
	}

	// Create a copy of the albums slice for sorting/paging
	// We sort the entire list because cross-page sorting needs it
	// In a real DB we'd use ORDER BY
	albumList := make([]models.Album, len(ac.albums))
	copy(albumList, ac.albums)

	switch sortMode {
	case "date":
		sort.Slice(albumList, func(i, j int) bool {
			if albumList[i].Date != albumList[j].Date {
				return albumList[i].Date > albumList[j].Date // Descending
			}
			return albumList[i].Album < albumList[j].Album
		})
	case "name":
		sort.Slice(albumList, func(i, j int) bool {
			if albumList[i].Album != albumList[j].Album {
				return albumList[i].Album < albumList[j].Album
			}
			return albumList[i].Artist < albumList[j].Artist
		})
	}

	end := offset + limit
	if end > total {
		end = total
	}

	return albumList[offset:end], total
}

// SearchAlbums searches for albums using fuzzy matching
func (ac *AlbumCache) SearchAlbums(query string, offset, limit int) ([]models.Album, int) {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	if ac.searchSource == nil || len(ac.albums) == 0 {
		return []models.Album{}, 0
	}

	matches := fuzzy.FindFrom(query, ac.searchSource)

	total := len(matches)
	if offset >= total {
		return []models.Album{}, total
	}

	end := offset + limit
	if end > total {
		end = total
	}

	// Map matches back to albums
	result := make([]models.Album, end-offset)
	for i, match := range matches[offset:end] {
		if match.Index >= 0 && match.Index < len(ac.albums) {
			result[i] = ac.albums[match.Index]
		}
	}

	return result, total
}

// GetAllAlbums returns all albums from the cache
func (ac *AlbumCache) GetAllAlbums() []models.Album {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make([]models.Album, len(ac.albums))
	copy(result, ac.albums)
	return result
}

// SearchAlbumsByFields searches albums by any field (album, artist, genre, date)
// Returns albums matching the query in any of these fields
func (ac *AlbumCache) SearchAlbumsByFields(query string) []models.Album {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	if len(ac.albums) == 0 {
		return []models.Album{}
	}

	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return ac.albums
	}

	var results []models.Album
	for _, album := range ac.albums {
		if strings.Contains(strings.ToLower(album.Album), query) ||
			strings.Contains(strings.ToLower(album.Artist), query) ||
			strings.Contains(strings.ToLower(album.Genre), query) ||
			strings.Contains(strings.ToLower(album.Date), query) {
			results = append(results, album)
		}
	}

	return results
}

// GetCachedPage returns a cached page if it exists and hasn't expired
func (ac *AlbumCache) GetCachedPage(key string) ([]models.Album, bool) {
	ac.pageCacheMutex.RLock()
	defer ac.pageCacheMutex.RUnlock()

	item, ok := ac.pageCache[key]
	if !ok {
		return nil, false
	}

	if time.Now().After(item.ExpiresAt) {
		return nil, false
	}

	return item.Albums, true
}

// SetCachedPage stores a page in the cache with a 15-minute expiration
func (ac *AlbumCache) SetCachedPage(key string, albums []models.Album) {
	ac.pageCacheMutex.Lock()
	defer ac.pageCacheMutex.Unlock()

	ac.pageCache[key] = cachedPage{
		Albums:    albums,
		ExpiresAt: time.Now().Add(15 * time.Minute),
	}
}

// BackgroundEnrichAndCache enriches a page and caches it
func (ac *AlbumCache) BackgroundEnrichAndCache(offset, limit int, sortMode string) {
	// Generate key
	key := fmt.Sprintf("%d-%d-%s", offset, limit, sortMode)

	// Check if already cached (optimization to avoid work)
	if _, ok := ac.GetCachedPage(key); ok {
		return
	}

	// Check inflight
	ac.inflightMutex.Lock()
	if ac.inflight[key] {
		ac.inflightMutex.Unlock()
		return
	}
	ac.inflight[key] = true
	ac.inflightMutex.Unlock()

	// Ensure cleanup
	defer func() {
		ac.inflightMutex.Lock()
		delete(ac.inflight, key)
		ac.inflightMutex.Unlock()
	}()

	// fetch basic
	albums, total := ac.GetAlbumsPage(offset, limit, sortMode)
	if len(albums) == 0 {
		return
	}
	_ = total // ignored

	// enrich (this blocks the goroutine, not the caller)
	enriched, err := ac.EnrichAlbums(albums)
	if err != nil {
		log.Printf("Background enrich failed for %s: %v", key, err)
		return
	}

	// cache
	ac.SetCachedPage(key, enriched)
	log.Printf("Background enriched and cached page: %s (%d items)", key, len(enriched))
}

// GetAlbumDetails returns cached album details if they exist and haven't expired
func (ac *AlbumCache) GetAlbumDetails(artist, album string) (interface{}, bool) {
	key := fmt.Sprintf("%s|%s", artist, album)
	ac.detailsCacheMutex.RLock()
	defer ac.detailsCacheMutex.RUnlock()

	item, ok := ac.detailsCache[key]
	if !ok {
		return nil, false
	}

	if time.Now().After(item.ExpiresAt) {
		return nil, false
	}

	return item.Data, true
}

// SetAlbumDetails stores album details in the cache
func (ac *AlbumCache) SetAlbumDetails(artist, album string, data interface{}) {
	key := fmt.Sprintf("%s|%s", artist, album)
	ac.detailsCacheMutex.Lock()
	defer ac.detailsCacheMutex.Unlock()

	ac.detailsCache[key] = cachedDetails{
		Data:      data,
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
}
