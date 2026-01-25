package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"

	"mpd-client-modern/internal/config"
	"mpd-client-modern/internal/coverart"
	"mpd-client-modern/internal/metadata"
	"mpd-client-modern/internal/models"
	"mpd-client-modern/internal/mpd"
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

	// Playback Controls
	api.HandleFunc("/play", Play).Methods("POST")
	api.HandleFunc("/play/{pos}", PlayPos).Methods("POST")
	api.HandleFunc("/pause", Pause).Methods("POST")
	api.HandleFunc("/stop", Stop).Methods("POST")
	api.HandleFunc("/next", Next).Methods("POST")
	api.HandleFunc("/previous", Previous).Methods("POST")
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
	api.HandleFunc("/albums/random", HandleRandomAlbums).Methods("GET")
	api.HandleFunc("/albums/search", HandleAlbumSearch).Methods("GET")
	api.HandleFunc("/albums/enrich", HandleAlbumEnrich).Methods("POST")
	api.HandleFunc("/playlist/album", HandlePlaylistAlbum).Methods("POST")
	api.HandleFunc("/artists", ListArtists).Methods("GET")
	api.HandleFunc("/dates", ListDates).Methods("GET")
	api.HandleFunc("/genres", ListGenres).Methods("GET")
	api.HandleFunc("/album/{artist}/{album}", HandleAlbumDetails).Methods("GET")
	api.HandleFunc("/album/{artist}/{album}/tags", GetAlbumTags).Methods("GET")
	api.HandleFunc("/album/{artist}/{album}/tags", UpdateAlbumTags).Methods("POST")

	// Search
	api.HandleFunc("/search", Search).Methods("GET")

	// Metadata
	api.HandleFunc("/metadata/search", SearchMetadata).Methods("GET")

	// Cover Art
	api.HandleFunc("/coverart/{path:.*}", GetCoverArt).Methods("GET")
	api.HandleFunc("/coverart/candidates", GetCoverArtCandidates).Methods("GET")
	api.HandleFunc("/coverart/apply", ApplyCoverArt).Methods("POST")

	// WebSockets
	r.HandleFunc("/ws", WebsocketHandler)
	r.HandleFunc("/ws/logs", LogWebsocketHandler)

	// Status refresh (for explicit user requests)
	api.HandleFunc("/status/refresh", RefreshStatus).Methods("POST")
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

	var cfg config.Config
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}
	if err := cfg.Validate(); err != nil {
		SendError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := config.Save(&cfg); err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Reset the MPD client to use the new configuration in a goroutine to avoid blocking
	go func() {
		mpd.ResetClient()
	}()

	SendJSON(w, models.APIResponse{Success: true, Data: cfg})
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
	status, err := mpd.GetStatusClient().GetStatus()
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}
	SendJSON(w, models.APIResponse{Success: true, Data: status})
}

func RefreshStatus(w http.ResponseWriter, r *http.Request) {
	// Get current status
	status, err := mpd.GetStatusClient().GetStatus()
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
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

	provider := metadata.NewDiscogsProvider()
	candidates, err := provider.Search(artist, album)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}
	SendJSON(w, models.APIResponse{Success: true, Data: candidates})
}

func GetCoverArtCandidates(w http.ResponseWriter, r *http.Request) {
	artist := r.URL.Query().Get("artist")
	album := r.URL.Query().Get("album")

	manager := coverart.NewManager()
	candidates, err := manager.FetchCandidates(artist, album)
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}
	SendJSON(w, models.APIResponse{Success: true, Data: candidates})
}

func ApplyCoverArt(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AlbumPath string `json:"albumPath"`
		ImageURL  string `json:"imageUrl"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		SendError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	manager := coverart.NewManager()
	if err := manager.ApplyCover(req.AlbumPath, req.ImageURL); err != nil {
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

	// Add cache control
	w.Header().Set("Cache-Control", "public, max-age=300")

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
	volume, _ := url.PathUnescape(vars["volume"])

	_, err := mpd.GetClient().SendCommand(fmt.Sprintf("setvol %s", volume))
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
	// Parse optional pagination
	page := parseInt(r.URL.Query().Get("page"))
	limit := parseInt(r.URL.Query().Get("limit"))

	var items []models.PlaylistItem
	var err error

	// If pagination is requested
	if page > 0 && limit > 0 {
		start := (page - 1) * limit
		end := start + limit
		// MPD range is start:end where end is exclusive (usually).
		// client.GetPlaylistRange uses playlistinfo start:end.
		items, err = mpd.GetClient().GetPlaylistRange(start, end)
	} else {
		// Get full playlist
		items, err = mpd.GetClient().GetPlaylist()
	}

	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Get current status for current position and playlist length
	status, err := mpd.GetStatusClient().GetStatus()
	if err != nil {
		SendError(w, http.StatusInternalServerError, err.Error())
		return
	}

	playlistInfo := models.PlaylistInfo{
		Items:      items,
		Length:     status.PlaylistLength, // Use the authoritative length from status
		CurrentPos: status.PlaylistPos,
	}

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

func SendError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(models.APIResponse{Success: false, Error: message})
}
