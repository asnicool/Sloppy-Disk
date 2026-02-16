package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"mpd-client-modern/internal/albumcache"
	"mpd-client-modern/internal/config"
	"mpd-client-modern/internal/models"
	"mpd-client-modern/internal/mpd"

	"github.com/gorilla/mux"
)

var (
	randomCache      []models.Album
	randomCacheLock  sync.Mutex
	randomCacheTime  time.Time
	randomCacheCount int
)

// HandleAlbumList handles /api/albums
func HandleAlbumList(w http.ResponseWriter, r *http.Request) {
	page := 0
	limit := 50

	if p := r.URL.Query().Get("page"); p != "" {
		if val, err := strconv.Atoi(p); err == nil {
			page = val
		}
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil {
			limit = val
		}
	}

	// Calculate offset
	offset := page * limit
	sortMode := r.URL.Query().Get("sort")

	// Generate cache key
	cacheKey := fmt.Sprintf("%d-%d-%s", offset, limit, sortMode)

	cache := albumcache.GetCache()

	var albums []models.Album
	var total int

	// Check cache first
	if cached, ok := cache.GetCachedPage(cacheKey); ok {
		albums = cached
		// We still need total count, which GetAlbumsPage returns.
		// Since GetAlbumsPage is fast (in-memory slice), we can call it just for total,
		// or simpler: just get the slice again to get total, but use cached albums.
		_, total = cache.GetAlbumsPage(offset, limit, sortMode)

		// REMOVED: Background enrichment calls - no longer called
		// Album details will be loaded opportunistically when cards are viewed
	} else {
		// Cache miss: Get basic albums
		albums, total = cache.GetAlbumsPage(offset, limit, sortMode)

		// Return basic albums immediately (FAST)

		// REMOVED: Background enrichment goroutines
		// Album details will be loaded opportunistically when cards are viewed
	}

	response := models.APIResponse{
		Success: true,
		Data:    albums,
		Meta: models.PaginationMeta{
			Page:    page,
			Limit:   limit,
			Total:   total,
			HasMore: offset+len(albums) < total,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleAlbumSearch handles /api/albums/search
func HandleAlbumSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		HandleAlbumList(w, r)
		return
	}

	page := 0
	limit := 50

	if p := r.URL.Query().Get("page"); p != "" {
		if val, err := strconv.Atoi(p); err == nil {
			page = val
		}
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil {
			limit = val
		}
	}

	// Calculate offset
	offset := page * limit

	cache := albumcache.GetCache()
	albums, total := cache.SearchAlbums(query, offset, limit)

	// Enrich albums before sending
	enriched, err := cache.EnrichAlbums(albums)
	if err != nil {
		log.Printf("Error enriching search results: %v", err)
	} else {
		albums = enriched
	}

	response := models.APIResponse{
		Success: true,
		Data:    albums,
		Meta: models.PaginationMeta{
			Page:    page,
			Limit:   limit,
			Total:   total,
			HasMore: offset+len(albums) < total,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleAlbumEnrich handles POST /api/albums/enrich
func HandleAlbumEnrich(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Albums []models.Album `json:"albums"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(models.APIResponse{
			Success: false,
			Error:   "Invalid request body",
		})
		return
	}

	cache := albumcache.GetCache()
	enriched, err := cache.EnrichAlbums(request.Albums)
	if err != nil {
		log.Printf("Error enriching albums on demand: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(models.APIResponse{
			Success: false,
			Error:   "Failed to enrich albums",
		})
		return
	}

	response := models.APIResponse{
		Success: true,
		Data:    enriched,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleAlbumDetails handles GET /api/album/{artist}/{album}
func HandleAlbumDetails(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	artist, _ := url.PathUnescape(vars["artist"])
	albumName, _ := url.PathUnescape(vars["album"])

	if artist == "" || albumName == "" {
		SendError(w, http.StatusBadRequest, "Artist and Album are required")
		return
	}

	cache := albumcache.GetCache()
	if data, ok := cache.GetAlbumDetails(artist, albumName); ok {
		SendJSON(w, models.APIResponse{Success: true, Data: data})
		return
	}

	var data map[string]interface{}
	err := MPDCircuitBreaker.Execute(r.Context(), func() error {
		var execErr error
		data, execErr = fetchDetailedAlbumInfo(artist, albumName)
		return execErr
	})

if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	cache.SetAlbumDetails(artist, albumName, data)
	SendJSON(w, models.APIResponse{Success: true, Data: data})
}

type BatchDetailsRequest struct {
	Albums []struct {
		Artist string `json:"artist"`
		Album  string `json:"album"`
	} `json:"albums"`
}

// HandleAlbumDetailsBatch handles /api/albums/details/batch
// Uses MPD command lists to fetch multiple albums in a single round-trip
// This is significantly faster than sequential individual requests
func HandleAlbumDetailsBatch(w http.ResponseWriter, r *http.Request) {
	var req BatchDetailsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Limit batch size to prevent overwhelming MPD and ensure status/playlist availability
	const maxBatchSize = 10
	if len(req.Albums) > maxBatchSize {
		SendError(w, http.StatusBadRequest, fmt.Sprintf("Batch size exceeds maximum of %d", maxBatchSize))
		return
	}

	start := time.Now()
	results := make(map[string]interface{})
	cache := albumcache.GetCache()

	// Separate cached from uncached albums
	type albumRequest struct {
		Artist string
		Album  string
		Key    string
	}
	var toFetch []albumRequest

	for _, a := range req.Albums {
		key := fmt.Sprintf("%s|%s", a.Artist, a.Album)
		if data, ok := cache.GetAlbumDetails(a.Artist, a.Album); ok {
			results[key] = data
		} else {
			toFetch = append(toFetch, albumRequest{
				Artist: a.Artist,
				Album:  a.Album,
				Key:    key,
			})
		}
	}

	// Batch fetch using MPD command list for efficiency
	if len(toFetch) > 0 {
		log.Printf("[BATCH] Fetching %d albums using command list", len(toFetch))

		// Build command list - all find commands in one batch
		var commands []string
		for _, item := range toFetch {
			albumEsc := strings.ReplaceAll(item.Album, "\"", "\\\"")
			artistEsc := strings.ReplaceAll(item.Artist, "\"", "\\\"")
			// Try artist first, then albumartist if artist doesn't work
			cmd := fmt.Sprintf("find album \"%s\" artist \"%s\"", albumEsc, artistEsc)
			commands = append(commands, cmd)
		}

		// Execute all commands in ONE round-trip using command list
		// This is the key optimization - MPD processes all commands in a single network call
		responses, err := mpd.GetClient().SendCommandList(commands)
		if err != nil {
			log.Printf("[BATCH] Command list failed: %v (took %v)", err, time.Since(start))
			SendError(w, http.StatusInternalServerError, err.Error())
			return
		}

		// Process responses - parse songs and build album info
		for i, resp := range responses {
			if i >= len(toFetch) {
				break
			}
			item := toFetch[i]

			// Parse songs from response
			songs := parseSongs(resp)

			// If artist filter didn't work, try albumartist
			if len(songs) == 0 {
				log.Printf("[BATCH] No songs with artist, trying albumartist for %s - %s", item.Artist, item.Album)
				// We need to fetch this one individually with albumartist
				data, fetchErr := fetchDetailedAlbumInfo(item.Artist, item.Album)
				if fetchErr == nil && data != nil {
					cache.SetAlbumDetails(item.Artist, item.Album, data)
					results[item.Key] = data
				} else {
					log.Printf("[BATCH] Failed to fetch %s - %s: %v", item.Artist, item.Album, fetchErr)
				}
				continue
			}

			// Build album info from songs
			if len(songs) > 0 {
				data := buildAlbumInfoFromSongs(item.Artist, item.Album, songs)
				cache.SetAlbumDetails(item.Artist, item.Album, data)
				results[item.Key] = data
			}
		}

		log.Printf("[BATCH] Fetched %d albums in %v (avg: %v per album)",
			len(toFetch), time.Since(start), time.Since(start)/time.Duration(len(toFetch)))
	}

	SendJSON(w, models.APIResponse{
		Success: true,
		Data:    results,
	})
}

func fetchDetailedAlbumInfo(artist, albumName string) (map[string]interface{}, error) {
	// Fetch songs for this album
	// Use find as it's more precise than search
	albumEsc := strings.ReplaceAll(albumName, "\"", "\\\"")
	artistEsc := strings.ReplaceAll(artist, "\"", "\\\"")

	// We'll try to find by album and artist
	cmd := fmt.Sprintf("find album \"%s\" artist \"%s\"", albumEsc, artistEsc)
	resp, err := mpd.GetClient().SendCommand(cmd)
	if err != nil {
		return nil, fmt.Errorf("error searching songs for album %s by %s: %v", albumName, artist, err)
	}

	songs := parseSongs(resp)
	if len(songs) == 0 {
		// Try again with albumartist if artist didn't work
		cmd = fmt.Sprintf("find album \"%s\" albumartist \"%s\"", albumEsc, artistEsc)
		resp, err = mpd.GetClient().SendCommand(cmd)
		if err == nil {
			songs = parseSongs(resp)
		}
	}

	if len(songs) == 0 {
		return nil, fmt.Errorf("album not found")
	}

	// Compute stats
	totalDuration := 0
	genre := ""
	date := ""
	path := ""

	for _, s := range songs {
		totalDuration += s.Duration
		if genre == "" && s.Genre != "" {
			genre = s.Genre
		}
		if date == "" && s.Date != "" {
			date = s.Date
		}
		if path == "" {
			path = filepath.Dir(s.Path)
		}
	}

	// Escape path for URL, keeping slashes
	pathParts := strings.Split(path, "/")
	for i, part := range pathParts {
		pathParts[i] = url.PathEscape(part)
	}
	escapedPath := strings.Join(pathParts, "/")

	albumInfo := models.Album{
		Album:      albumName,
		Artist:     artist,
		TrackCount: len(songs),
		Duration:   totalDuration,
		Genre:      genre,
		Date:       date,
		Path:       path,
		CoverURL:   fmt.Sprintf("/api/coverart/%s", escapedPath),
	}

	// Prepare data to include both summary and tracklist
	return map[string]interface{}{
		"album":  albumInfo,
		"tracks": songs,
	}, nil
}

// buildAlbumInfoFromSongs constructs album details from a list of songs
// This is used by both fetchDetailedAlbumInfo and HandleAlbumDetailsBatch
func buildAlbumInfoFromSongs(artist, albumName string, songs []models.Song) map[string]interface{} {
	// Compute stats
	totalDuration := 0
	genre := ""
	date := ""
	path := ""

	for _, s := range songs {
		totalDuration += s.Duration
		if genre == "" && s.Genre != "" {
			genre = s.Genre
		}
		if date == "" && s.Date != "" {
			date = s.Date
		}
		if path == "" {
			path = filepath.Dir(s.Path)
		}
	}

	// Escape path for URL, keeping slashes
	pathParts := strings.Split(path, "/")
	for i, part := range pathParts {
		pathParts[i] = url.PathEscape(part)
	}
	escapedPath := strings.Join(pathParts, "/")

	albumInfo := models.Album{
		Album:      albumName,
		Artist:     artist,
		TrackCount: len(songs),
		Duration:   totalDuration,
		Genre:      genre,
		Date:       date,
		Path:       path,
		CoverURL:   fmt.Sprintf("/api/coverart/%s", escapedPath),
	}

	// Prepare data to include both summary and tracklist
	return map[string]interface{}{
		"album":  albumInfo,
		"tracks": songs,
	}
}

// HandleAllAlbums handles /api/albums/all
// Returns all albums from the cache for local search/filtering
// Enriches albums with track count, duration, and path
func HandleAllAlbums(w http.ResponseWriter, r *http.Request) {
	cache := albumcache.GetCache()

	// Get all albums from cache (basic info only)
	// NO enrichment called - this avoids overwhelming MPD with GetAlbumStats queries
	// Track count, duration, and path will be loaded opportunistically when needed
	allAlbums := cache.GetAllAlbums()

	response := models.APIResponse{
		Success: true,
		Data:    allAlbums,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleRandomAlbums handles /api/albums/random
func HandleRandomAlbums(w http.ResponseWriter, r *http.Request) {
	count := config.Get().RandomAlbumCount
	if count <= 0 {
		if count <= 0 {
			count = 36
		}
	}

	if c := r.URL.Query().Get("count"); c != "" {
		if val, err := strconv.Atoi(c); err == nil {
			count = val
		}
	}

	forceRefresh := r.URL.Query().Get("refresh") == "true"

	randomCacheLock.Lock()
	// Check if cache is valid (5 minutes and same count)
	refresh := r.URL.Query().Get("refresh") == "true"
	if !forceRefresh && !refresh && len(randomCache) > 0 && time.Since(randomCacheTime) < 5*time.Minute && randomCacheCount == count {
		randomCacheLock.Unlock()
		SendJSON(w, models.APIResponse{
			Success: true,
			Data:    randomCache,
		})
		return
	}
	randomCacheLock.Unlock()

	cache := albumcache.GetCache()
	// Use buffer (enrich=true)
	albums, err := cache.GetRandomAlbums(count, true)
	if err != nil {
		SendError(w, http.StatusInternalServerError, "Failed to fetch random albums: "+err.Error())
		return
	}

	// Update cache
	randomCacheLock.Lock()
	randomCache = albums
	randomCacheTime = time.Now()
	randomCacheCount = count
	randomCacheLock.Unlock()

	SendJSON(w, models.APIResponse{
		Success: true,
		Data:    albums,
	})
}

// HandlePlaylistAlbum handles POST /api/playlist/album
func HandlePlaylistAlbum(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Artist string `json:"artist"`
		Album  string `json:"album"`
		Mode   string `json:"mode"` // append, insert, replace
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	// Fetch tracks for the album
	albumEsc := strings.ReplaceAll(req.Album, "\"", "\\\"")
	artistEsc := strings.ReplaceAll(req.Artist, "\"", "\\\"")

	cmd := fmt.Sprintf("find album \"%s\" artist \"%s\"", albumEsc, artistEsc)
	client := mpd.GetClient()
	resp, err := client.SendCommand(cmd)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	songs := parseSongs(resp)
	if len(songs) == 0 {
		cmd = fmt.Sprintf("find album \"%s\" albumartist \"%s\"", albumEsc, artistEsc)
		resp, err = client.SendCommand(cmd)
		if err == nil {
			songs = parseSongs(resp)
		}
	}

	if len(songs) == 0 {
		SendError(w, http.StatusNotFound, "Album not found")
		return
	}

	// Perform playlist operation
	switch req.Mode {
	case "replace":
		commands := []string{"clear"}
		for _, s := range songs {
			commands = append(commands, fmt.Sprintf("add \"%s\"", strings.ReplaceAll(s.Path, "\"", "\\\"")))
		}
		commands = append(commands, "play")
		client.SendCommandList(commands)
	case "insert":
		err := client.Execute(func(conn *mpd.Connection) error {
			status, err := conn.GetStatus()
			if err != nil {
				return err
			}
			pos := status.PlaylistPos + 1
			commands := make([]string, len(songs))
			for i, s := range songs {
				commands[i] = fmt.Sprintf("addid \"%s\" %d", strings.ReplaceAll(s.Path, "\"", "\\\""), pos+i)
			}
			_, err = conn.SendCommandList(commands)
			return err
		})
		if err != nil {
			SendError(w, http.StatusInternalServerError, err.Error())
			return
		}
	case "append":
		fallthrough
	default:
		commands := make([]string, len(songs))
		for i, s := range songs {
			commands[i] = fmt.Sprintf("add \"%s\"", strings.ReplaceAll(s.Path, "\"", "\\\""))
		}
		client.SendCommandList(commands)
	}

	SendJSON(w, models.APIResponse{Success: true})
}
