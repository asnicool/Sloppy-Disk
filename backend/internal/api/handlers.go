package api

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"mpd-client-modern/internal/artistimage"
	"mpd-client-modern/internal/config"
	"mpd-client-modern/internal/coverart"
	"mpd-client-modern/internal/metadata"
	"mpd-client-modern/internal/models"
	"mpd-client-modern/internal/mpd"
	"mpd-client-modern/internal/n50"
	"mpd-client-modern/internal/sync"

	"github.com/gorilla/mux"
)

func RegisterRoutes(r *mux.Router) {
	fmt.Println("Registering API routes...")
	api := r.PathPrefix("/api").Subrouter()

	// Config
	api.HandleFunc("/config", GetConfig).Methods("GET")
	api.HandleFunc("/config", UpdateConfig).Methods("POST")
	api.HandleFunc("/connection-status", GetConnectionStatus).Methods("GET")

	// MPD Status
	api.HandleFunc("/status", GetStatus).Methods("GET")

	// Playback Controls (wrapped with N50 check)
	api.HandleFunc("/play", wrapPlaybackWithN50Check(Play)).Methods("POST")
	api.HandleFunc("/play/{pos}", wrapPlaybackWithN50Check(PlayPos)).Methods("POST")
	api.HandleFunc("/pause", Pause).Methods("POST")
	api.HandleFunc("/stop", Stop).Methods("POST")
	api.HandleFunc("/next", wrapPlaybackWithN50Check(Next)).Methods("POST")
	api.HandleFunc("/previous", wrapPlaybackWithN50Check(Previous)).Methods("POST")
	api.HandleFunc("/volume/{volume}", SetVolume).Methods("POST")

	// Playlist
	api.HandleFunc("/playlist", GetPlaylist).Methods("GET")
	api.HandleFunc("/playlist/move", MovePlaylistTrack).Methods("POST")
	api.HandleFunc("/playlist/remove/{pos}", RemovePlaylistTrack).Methods("POST")

	// Sync
	api.HandleFunc("/sync/status", GetSyncStatus).Methods("GET")
	api.HandleFunc("/sync/start", StartSync).Methods("POST")

	// Albums & Artists
	api.HandleFunc("/albums", HandleAlbumList).Methods("GET")
	api.HandleFunc("/albums/all", HandleAllAlbums).Methods("GET")
	api.HandleFunc("/albums/random", HandleRandomAlbums).Methods("GET")
	api.HandleFunc("/albums/details/batch", HandleAlbumDetailsBatch).Methods("POST")
	api.HandleFunc("/albums/search", HandleAlbumSearch).Methods("GET")
	api.HandleFunc("/albums/enrich", HandleAlbumEnrich).Methods("POST")
	api.HandleFunc("/playlist/album", HandlePlaylistAlbum).Methods("POST")
	api.HandleFunc("/artists", ListArtists).Methods("GET")
	api.HandleFunc("/dates", ListDates).Methods("GET")
	api.HandleFunc("/genres", ListGenres).Methods("GET")
	api.HandleFunc("/genres/matrix", GetGenreDateMatrix).Methods("GET")
	api.HandleFunc("/album/{artist}/{album}", HandleAlbumDetails).Methods("GET")
	api.HandleFunc("/album/{artist}/{album}/tags", GetAlbumTags).Methods("GET")
	api.HandleFunc("/album/{artist}/{album}/tags", UpdateAlbumTags).Methods("POST")

	// Search
	api.HandleFunc("/search", Search).Methods("GET")

	// Metadata
	api.HandleFunc("/metadata/search", SearchMetadata).Methods("GET")
	api.HandleFunc("/metadata/details", GetMetadataDetails).Methods("GET")
	api.HandleFunc("/metadata/apply", ApplyMetadata).Methods("POST")

	// Cover Art - specific routes first, then catch-all
	api.HandleFunc("/coverart/candidates", GetCoverArtCandidates).Methods("GET")
	api.HandleFunc("/coverart/apply", ApplyCoverArt).Methods("POST")
	api.HandleFunc("/coverart/upload", UploadCoverArt).Methods("POST")
	api.HandleFunc("/coverart/{path:.*}", GetCoverArt).Methods("GET")

	// Artist Art
	api.HandleFunc("/artistart/candidates", GetArtistArtCandidates).Methods("GET")
	api.HandleFunc("/artistart/apply", ApplyArtistArt).Methods("POST")
	api.HandleFunc("/artistart/{artist}/candidates", GetArtistArtCandidatesByArtist).Methods("GET")

	// WebSockets
	r.HandleFunc("/ws", WebsocketHandler)
	r.HandleFunc("/ws/search", SearchWebSocketHandler)
	r.HandleFunc("/ws/logs", LogWebsocketHandler)

	// Status refresh (for explicit user requests)
	api.HandleFunc("/status/refresh", RefreshStatus).Methods("POST")

	// Circuit breaker stats (for monitoring)
	api.HandleFunc("/circuit-breaker/stats", GetCircuitBreakerStats).Methods("GET")
}

func GetConfig(w http.ResponseWriter, r *http.Request) {
	SendJSON(w, models.APIResponse{Success: true, Data: config.Get()})
}

func UpdateConfig(w http.ResponseWriter, r *http.Request) {
	// Check if request body is empty
	bodyBytes := []byte{}
	if r.Body != nil {
		bodyBytes, _ = io.ReadAll(r.Body)
	}

	if len(bodyBytes) == 0 {
		SendError(w, http.StatusBadRequest, "Request body is empty")
		return
	}

	// Restore the body for decoding
	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	var newCfg config.ConfigDTO
	if err := json.NewDecoder(r.Body).Decode(&newCfg); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

	// Load current config and apply changes
	currentCfg := config.Get()
	currentCfg.ApplyDTO(&newCfg)

	if err := currentCfg.Validate(); err != nil {
		SendError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := config.Save(currentCfg); err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Reset the MPD client to use the new configuration in a goroutine to avoid blocking
	go func() {
		mpd.ResetClient()
		n50.ResetClient()
	}()

	SendJSON(w, models.APIResponse{Success: true, Data: currentCfg})
}

// GetConnectionStatus returns the current connection status to the MPD server
func GetConnectionStatus(w http.ResponseWriter, r *http.Request) {
	client := mpd.GetClient()

	isConnected := client.IsConnected()

	SendJSON(w, models.APIResponse{
		Success: true,
		Data: map[string]interface{}{
			"connected": isConnected,
			"config":    config.Get(),
		},
	})
}

func GetStatus(w http.ResponseWriter, r *http.Request) {
	log.Printf("[API] GetStatus called")
	start := time.Now()

	// Use pooled client to avoid head-of-line blocking
	status, err := mpd.GetClient().GetStatus()
	if err != nil {
		log.Printf("[API] GetStatus failed: %v (took %v). Response status: 500", err, time.Since(start))
		SendError(w, http.StatusInternalServerError, "MPD Status Error: "+err.Error())
		return
	}
	log.Printf("[API] GetStatus success (took %v)", time.Since(start))
	SendJSON(w, models.APIResponse{Success: true, Data: status})
}

func RefreshStatus(w http.ResponseWriter, r *http.Request) {
	// Get current status using pooled client to avoid head-of-line blocking
	status, err := mpd.GetClient().GetStatus()
	if err != nil {
		log.Printf("[API] RefreshStatus failed: %v. Response status: 500", err)
		SendError(w, http.StatusInternalServerError, "MPD Refresh Status Error: "+err.Error())
		return
	}

	// Broadcast to all WebSocket clients if broadcaster is available
	if GlobalBroadcaster != nil {
		GlobalBroadcaster.Broadcast(status)
	}

	SendJSON(w, models.APIResponse{Success: true, Data: status})
}

func Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		SendError(w, http.StatusBadRequest, "Search query is required")
		return
	}

	// Legacy search endpoint (returns empty/limited results to encourage WS usage)
	SendJSON(w, models.APIResponse{Success: true, Data: []interface{}{}, Error: "Please use WebSocket streaming search for results"})
}

func GetSyncStatus(w http.ResponseWriter, r *http.Request) {
	status := sync.GetManager().GetStatus()
	SendJSON(w, models.APIResponse{Success: true, Data: status})
}

func StartSync(w http.ResponseWriter, r *http.Request) {
	if err := sync.GetManager().StartSync(r.Context()); err != nil {
		SendError(w, http.StatusConflict, err.Error())
		return
	}
	SendJSON(w, models.APIResponse{Success: true})
}

// ListAlbums implementation removed in favor of HandleAlbumList in albums.go

func ListArtists(w http.ResponseWriter, r *http.Request) {
	page := parseInt(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	limit := parseInt(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	// Get all artists and their albums from MPD
	resp, err := mpd.GetClient().SendCommand("list album group artist")
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	lines := strings.Split(strings.TrimSpace(resp), "\n")
	var artists []models.ArtistGroup
	var currentArtist *models.ArtistGroup

	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) != 2 {
			continue
		}

		key, value := parts[0], parts[1]
		if key == "Artist" {
			if currentArtist != nil {
				artists = append(artists, *currentArtist)
			}
			currentArtist = &models.ArtistGroup{
				Artist: value,
				Albums: make([]string, 0),
			}
		} else if key == "Album" && currentArtist != nil {
			if value != "" {
				currentArtist.Albums = append(currentArtist.Albums, value)
			}
		}
	}
	if currentArtist != nil {
		artists = append(artists, *currentArtist)
	}

	// Apply pagination
	start := (page - 1) * limit
	end := start + limit
	var paginated []models.ArtistGroup
	if start >= len(artists) {
		paginated = []models.ArtistGroup{}
	} else if end > len(artists) {
		paginated = artists[start:]
	} else {
		paginated = artists[start:end]
	}

	total := len(artists)
	meta := models.PaginationMeta{
		Page:    page,
		Limit:   limit,
		Total:   total,
		HasMore: end < total,
	}

	SendJSON(w, models.APIResponse{Success: true, Data: paginated, Meta: meta})
}

func ListDates(w http.ResponseWriter, r *http.Request) {
	page := parseInt(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	limit := parseInt(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	resp, err := mpd.GetClient().SendCommand("list album group date")
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	lines := strings.Split(strings.TrimSpace(resp), "\n")
	var groups []models.GroupedResult
	var currentGroup *models.GroupedResult

	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) != 2 {
			continue
		}

		key, value := parts[0], parts[1]
		if key == "Date" {
			if currentGroup != nil {
				groups = append(groups, *currentGroup)
			}
			currentGroup = &models.GroupedResult{
				Key:    value,
				Albums: make([]string, 0),
			}
		} else if key == "Album" && currentGroup != nil {
			if value != "" {
				currentGroup.Albums = append(currentGroup.Albums, value)
			}
		}
	}
	if currentGroup != nil {
		groups = append(groups, *currentGroup)
	}

	// Sort dates descending
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Key > groups[j].Key
	})

	start := (page - 1) * limit
	end := start + limit
	var paginated []models.GroupedResult
	if start >= len(groups) {
		paginated = []models.GroupedResult{}
	} else if end > len(groups) {
		paginated = groups[start:]
	} else {
		paginated = groups[start:end]
	}

	SendJSON(w, models.APIResponse{
		Success: true,
		Data:    paginated,
		Meta: models.PaginationMeta{
			Page:    page,
			Limit:   limit,
			Total:   len(groups),
			HasMore: end < len(groups),
		},
	})
}

func ListGenres(w http.ResponseWriter, r *http.Request) {
	page := parseInt(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	limit := parseInt(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	resp, err := mpd.GetClient().SendCommand("list album group genre")
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	lines := strings.Split(strings.TrimSpace(resp), "\n")
	var groups []models.GroupedResult
	var currentGroup *models.GroupedResult

	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) != 2 {
			continue
		}

		key, value := parts[0], parts[1]
		if key == "Genre" {
			if currentGroup != nil {
				groups = append(groups, *currentGroup)
			}
			currentGroup = &models.GroupedResult{
				Key:    value,
				Albums: make([]string, 0),
			}
		} else if key == "Album" && currentGroup != nil {
			if value != "" {
				currentGroup.Albums = append(currentGroup.Albums, value)
			}
		}
	}
	if currentGroup != nil {
		groups = append(groups, *currentGroup)
	}

	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Key < groups[j].Key
	})

	start := (page - 1) * limit
	end := start + limit
	var paginated []models.GroupedResult
	if start >= len(groups) {
		paginated = []models.GroupedResult{}
	} else if end > len(groups) {
		paginated = groups[start:]
	} else {
		paginated = groups[start:end]
	}

	SendJSON(w, models.APIResponse{
		Success: true,
		Data:    paginated,
		Meta: models.PaginationMeta{
			Page:    page,
			Limit:   limit,
			Total:   len(groups),
			HasMore: end < len(groups),
		},
	})
}

func GetAlbumTags(w http.ResponseWriter, r *http.Request) {
	SendJSON(w, models.APIResponse{Success: true, Data: []models.Song{}})
}

func UpdateAlbumTags(w http.ResponseWriter, r *http.Request) {
	SendJSON(w, models.APIResponse{Success: true})
}

func SearchMetadata(w http.ResponseWriter, r *http.Request) {
	artist := r.URL.Query().Get("artist")
	album := r.URL.Query().Get("album")
	providersParam := r.URL.Query().Get("providers")
	trackCountStr := r.URL.Query().Get("trackCount")
	durationStr := r.URL.Query().Get("duration")

	log.Printf("[METADATA HANDLER] SearchMetadata called")
	log.Printf("[METADATA HANDLER]   Artist: '%s'", artist)
	log.Printf("[METADATA HANDLER]   Album: '%s'", album)
	log.Printf("[METADATA HANDLER]   Providers param: '%s'", providersParam)
	log.Printf("[METADATA HANDLER]   TrackCount: '%s', Duration: '%s'", trackCountStr, durationStr)

	// Parse optional params
	var trackCount int
	var duration int
	if trackCountStr != "" {
		fmt.Sscanf(trackCountStr, "%d", &trackCount)
	}
	if durationStr != "" {
		fmt.Sscanf(durationStr, "%d", &duration)
	}

	// Parse providers list
	var providers []string
	if providersParam != "" {
		providers = strings.Split(providersParam, ",")
		log.Printf("[METADATA HANDLER]   Parsed providers: %v", providers)
	} else {
		log.Printf("[METADATA HANDLER]   No providers specified, will use all available providers")
	}

	// Use aggregator for multi-provider search
	log.Printf("[METADATA HANDLER] Creating aggregator and starting search...")
	aggregator := metadata.NewAggregator()
	candidates, err := aggregator.Search(r.Context(), artist, album, providers, trackCount, duration)
	if err != nil {
		log.Printf("[METADATA HANDLER] Search failed with error: %v", err)
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	log.Printf("[METADATA HANDLER] Search completed successfully. Returning %d candidates", len(candidates))
	SendJSON(w, models.APIResponse{Success: true, Data: candidates})
}

func GetMetadataDetails(w http.ResponseWriter, r *http.Request) {
	source := r.URL.Query().Get("source")
	externalID := r.URL.Query().Get("externalId")

	log.Printf("[METADATA DETAILS] GetMetadataDetails called")
	log.Printf("[METADATA DETAILS]   Source: '%s'", source)
	log.Printf("[METADATA DETAILS]   ExternalID: '%s'", externalID)

	if source == "" || externalID == "" {
		log.Printf("[METADATA DETAILS] Error: source and externalId are required")
		SendError(w, http.StatusBadRequest, "source and externalId are required")
		return
	}

	log.Printf("[METADATA DETAILS] Creating aggregator and fetching details...")
	aggregator := metadata.NewAggregator()
	details, err := aggregator.GetReleaseDetails(r.Context(), source, externalID)
	if err != nil {
		log.Printf("[METADATA DETAILS] Error fetching details: %v", err)
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if details == nil {
		log.Printf("[METADATA DETAILS] Error: Release not found")
		SendError(w, http.StatusNotFound, "Release not found")
		return
	}

	log.Printf("[METADATA DETAILS] Successfully fetched details for '%s - %s' (%s)", details.Artist, details.Album, details.Year)
	log.Printf("[METADATA DETAILS]   Tracks: %d", len(details.Tracks))
	SendJSON(w, models.APIResponse{Success: true, Data: details})
}

func ApplyMetadata(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AlbumPath   string                   `json:"albumPath"`
		Metadata    models.MetadataCandidate `json:"metadata"`
		CoverArtURL string                   `json:"coverArtUrl,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

	if req.AlbumPath == "" {
		SendError(w, http.StatusBadRequest, "albumPath is required")
		return
	}

	tagWriter := metadata.NewTagWriter()

	// Apply metadata tags
	result, err := tagWriter.ApplyMetadata(req.AlbumPath, req.Metadata)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Apply cover art if provided
	if req.CoverArtURL != "" {
		coverResult, err := tagWriter.ApplyCoverArt(req.AlbumPath, req.CoverArtURL)
		if err != nil {
			// Log but don't fail
			result.Errors = append(result.Errors, "Cover art: "+err.Error())
		} else {
			log.Printf("Cover art applied: %s (format: %s, size: %d bytes)", coverResult.DestPath, coverResult.Format, coverResult.ContentLength)
		}
	}

	SendJSON(w, models.APIResponse{Success: true, Data: result})
}

func GetCoverArtCandidates(w http.ResponseWriter, r *http.Request) {
	artist := r.URL.Query().Get("artist")
	album := r.URL.Query().Get("album")

	manager := coverart.NewManager()
	candidates, err := manager.FetchCandidates(artist, album)
	if err != nil {
		SendJSON(w, models.APIResponse{Success: true, Data: []models.CoverArtCandidate{}})
		return
	}
	SendJSON(w, models.APIResponse{Success: true, Data: candidates})
}

func ApplyCoverArt(w http.ResponseWriter, r *http.Request) {
	log.Printf("[API] ApplyCoverArt called")
	var req struct {
		AlbumPath string `json:"albumPath"`
		ImageURL  string `json:"imageUrl"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[API] ApplyCoverArt: Failed to decode JSON: %v", err)
		SendError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	log.Printf("[API] ApplyCoverArt: albumPath=%s, imageUrl=%s", req.AlbumPath, req.ImageURL)

	manager := coverart.NewManager()
	if err := manager.ApplyCover(req.AlbumPath, req.ImageURL); err != nil {
		log.Printf("[API] ApplyCoverArt: Failed to apply cover: %v", err)
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}
	log.Printf("[API] ApplyCoverArt: Success")
	SendJSON(w, models.APIResponse{Success: true})
}

func UploadCoverArt(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10MB max
		SendError(w, http.StatusBadRequest, "Failed to parse form: "+err.Error())
		return
	}

	albumPath := r.FormValue("albumPath")
	if albumPath == "" {
		SendError(w, http.StatusBadRequest, "albumPath is required")
		return
	}

	file, header, err := r.FormFile("cover")
	if err != nil {
		SendError(w, http.StatusBadRequest, "cover file is required")
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	manager := coverart.NewManager()
	if err := manager.SaveUploadedCover(albumPath, file, contentType); err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	SendJSON(w, models.APIResponse{Success: true})
}

// GetCoverArt serves cover art images from the cover art root directory
func GetCoverArt(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	path, _ := url.PathUnescape(vars["path"])

	manager := coverart.NewManager()
	filePath, err := manager.FindImage(path)
	if err != nil {
		if os.IsNotExist(err) {
			http.NotFound(w, r)
		} else {
			SendError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	// Get file info for ETag and Last-Modified
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Generate ETag from file content hash (SHA-256 of file path + modification time)
	etagContent := fmt.Sprintf("%x-%d", sha256.Sum256([]byte(filePath+fileInfo.ModTime().String())), fileInfo.Size())
	etag := fmt.Sprintf(`"%s"`, etagContent)

	// Set Last-Modified header
	lastModified := fileInfo.ModTime().UTC().Format(http.TimeFormat)
	w.Header().Set("Last-Modified", lastModified)

	// Check If-None-Match header
	ifNoneMatch := r.Header.Get("If-None-Match")
	if ifNoneMatch != "" {
		// Check if ETag matches
		if strings.Contains(ifNoneMatch, etag) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	// Check If-Modified-Since header
	ifModifiedSince := r.Header.Get("If-Modified-Since")
	if ifModifiedSince != "" && ifNoneMatch == "" {
		ifModifiedTime, err := time.Parse(http.TimeFormat, ifModifiedSince)
		if err == nil && !fileInfo.ModTime().After(ifModifiedTime) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
	}

	// Set ETag and cache control
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "public, max-age=300, immutable")

	// Determine content type based on file extension
	contentType := "image/jpeg"
	if strings.HasSuffix(strings.ToLower(filePath), ".png") {
		contentType = "image/png"
	} else if strings.HasSuffix(strings.ToLower(filePath), ".gif") {
		contentType = "image/gif"
	} else if strings.HasSuffix(strings.ToLower(filePath), ".webp") {
		contentType = "image/webp"
	}
	w.Header().Set("Content-Type", contentType)

	// Set Content-Length for better caching
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

	http.ServeFile(w, r, filePath)
}

// Playback Control Handlers
func Play(w http.ResponseWriter, r *http.Request) {
	_, err := mpd.GetClient().SendCommand("play")
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}
	SendJSON(w, models.APIResponse{Success: true})
}

func Pause(w http.ResponseWriter, r *http.Request) {
	_, err := mpd.GetClient().SendCommand("pause")
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}
	SendJSON(w, models.APIResponse{Success: true})
}

func Stop(w http.ResponseWriter, r *http.Request) {
	_, err := mpd.GetClient().SendCommand("stop")
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}
	SendJSON(w, models.APIResponse{Success: true})
}

func Next(w http.ResponseWriter, r *http.Request) {
	_, err := mpd.GetClient().SendCommand("next")
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}
	SendJSON(w, models.APIResponse{Success: true})
}

func Previous(w http.ResponseWriter, r *http.Request) {
	_, err := mpd.GetClient().SendCommand("previous")
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}
	SendJSON(w, models.APIResponse{Success: true})
}

func SetVolume(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	volumeStr, _ := url.PathUnescape(vars["volume"])
	volume := parseInt(volumeStr)

	if volume < 0 || volume > 100 {
		SendError(w, http.StatusBadRequest, "Volume must be between 0 and 100")
		return
	}

	_, err := mpd.GetClient().SendCommand(fmt.Sprintf("setvol %d", volume))
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}
	SendJSON(w, models.APIResponse{Success: true})
}

func PlayPos(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	pos, _ := url.PathUnescape(vars["pos"])

	_, err := mpd.GetClient().SendCommand(fmt.Sprintf("play %s", pos))
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}
	SendJSON(w, models.APIResponse{Success: true})
}
func GetPlaylist(w http.ResponseWriter, r *http.Request) {
	log.Printf("[API] GetPlaylist called")
	start := time.Now()

	// Parse optional pagination
	page := parseInt(r.URL.Query().Get("page"))
	limit := parseInt(r.URL.Query().Get("limit"))

	var items []models.PlaylistItem
	var err error

	// If pagination is requested
	if page > 0 && limit > 0 {
		log.Printf("[API] GetPlaylist: Getting range %d-%d", (page-1)*limit, (page-1)*limit+limit)
		start := (page - 1) * limit
		end := start + limit
		// MPD range is start:end where end is exclusive (usually).
		// client.GetPlaylistRange uses playlistinfo start:end.
		items, err = mpd.GetClient().GetPlaylistRange(start, end)
	} else {
		log.Printf("[API] GetPlaylist: Getting full playlist")
		// Get full playlist
		items, err = mpd.GetClient().GetPlaylist()
	}

	if err != nil {
		log.Printf("[API] GetPlaylist: Failed to get items: %v (took %v)", err, time.Since(start))
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}
	log.Printf("[API] GetPlaylist: Got %d items (took %v)", len(items), time.Since(start))

	// Get current status for current position and playlist length
	// Use pooled client to avoid head-of-line blocking
	status, err := mpd.GetClient().GetStatus()
	if err != nil {
		log.Printf("[API] GetPlaylist: Failed to get status: %v (took %v)", err, time.Since(start))
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	playlistInfo := models.PlaylistInfo{
		Items:      items,
		Length:     status.PlaylistLength, // Use the authoritative length from status
		CurrentPos: status.PlaylistPos,
	}

	log.Printf("[API] GetPlaylist: Success (took %v total)", time.Since(start))
	SendJSON(w, models.APIResponse{Success: true, Data: playlistInfo})
}

func parseInt(s string) int {
	var i int
	fmt.Sscanf(s, "%d", &i)
	return i
}

// Helpers
// SendJSON sends JSON response
func SendJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		SendError(w, http.StatusInternalServerError, "Failed to encode JSON: "+err.Error())
	}
}

// GetGenreDateMatrix returns a matrix of genre vs date with album counts
func GetGenreDateMatrix(w http.ResponseWriter, r *http.Request) {
	// Get all albums with their genre and date
	resp, err := mpd.GetClient().SendCommand("list album group genre group date")
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	lines := strings.Split(strings.TrimSpace(resp), "\n")

	// Map to store counts: matrix[genre][date] = count
	matrix := make(map[string]map[string]int)
	allGenres := make(map[string]bool)
	allDates := make(map[string]bool)

	var currentGenre, currentDate string

	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) != 2 {
			continue
		}

		key, value := parts[0], parts[1]
		switch key {
		case "Genre":
			currentGenre = value
			if currentGenre == "" {
				currentGenre = "Unknown"
			}
			allGenres[currentGenre] = true
			if matrix[currentGenre] == nil {
				matrix[currentGenre] = make(map[string]int)
			}
		case "Date":
			currentDate = value
			if currentDate == "" {
				currentDate = "Unknown"
			}
			// Extract year from date (take first 4 characters if it's a full date)
			if len(currentDate) >= 4 {
				currentDate = currentDate[:4]
			}
			allDates[currentDate] = true
		case "Album":
			if currentGenre != "" && currentDate != "" {
				matrix[currentGenre][currentDate]++
			}
		}
	}

	// Convert to sorted slices
	genres := make([]string, 0, len(allGenres))
	for g := range allGenres {
		genres = append(genres, g)
	}
	sort.Strings(genres)

	dates := make([]string, 0, len(allDates))
	for d := range allDates {
		dates = append(dates, d)
	}
	// Sort dates in descending order (newest first)
	sort.Slice(dates, func(i, j int) bool {
		return dates[i] > dates[j]
	})

	// Build response structure
	type MatrixCell struct {
		Count int    `json:"count"`
		Genre string `json:"genre"`
		Date  string `json:"date"`
	}

	result := struct {
		Genres []string                  `json:"genres"`
		Dates  []string                  `json:"dates"`
		Matrix map[string]map[string]int `json:"matrix"`
	}{
		Genres: genres,
		Dates:  dates,
		Matrix: matrix,
	}

	SendJSON(w, models.APIResponse{Success: true, Data: result})
}

func SendError(w http.ResponseWriter, code int, message string) {
	if strings.Contains(strings.ToLower(message), "server busy") {
		code = http.StatusTooManyRequests
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(models.APIResponse{Success: false, Error: message})
}

// GetCircuitBreakerStats returns the current state of the circuit breaker
func GetCircuitBreakerStats(w http.ResponseWriter, r *http.Request) {
	stats := MPDCircuitBreaker.GetStats()
	SendJSON(w, models.APIResponse{Success: true, Data: stats})
}

// Artist Art handlers

func GetArtistArtCandidates(w http.ResponseWriter, r *http.Request) {
	log.Printf("[API] GetArtistArtCandidates called")
	artist := r.URL.Query().Get("artist")
	if artist == "" {
		SendError(w, http.StatusBadRequest, "artist parameter is required")
		return
	}
	log.Printf("[API] GetArtistArtCandidates for artist: %s", artist)

	manager := artistimage.NewManager()
	candidates, err := manager.FetchCandidates(artist)
	if err != nil {
		log.Printf("[API] GetArtistArtCandidates error: %v", err)
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}
	log.Printf("[API] GetArtistArtCandidates found %d candidates", len(candidates))

	SendJSON(w, models.APIResponse{Success: true, Data: candidates})
}

func GetArtistArtCandidatesByArtist(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	artist, err := url.PathUnescape(vars["artist"])
	if err != nil {
		artist = vars["artist"]
	}

	if artist == "" {
		SendError(w, http.StatusBadRequest, "artist parameter is required")
		return
	}

	manager := artistimage.NewManager()
	candidates, err := manager.FetchCandidates(artist)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	SendJSON(w, models.APIResponse{Success: true, Data: candidates})
}

func ApplyArtistArt(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Artist   string `json:"artist"`
		ImageURL string `json:"imageUrl"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.Artist == "" || req.ImageURL == "" {
		SendError(w, http.StatusBadRequest, "artist and imageUrl are required")
		return
	}

	manager := artistimage.NewManager()
	if err := manager.ApplyArtistImage(req.Artist, req.ImageURL); err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	SendJSON(w, models.APIResponse{Success: true})
}
