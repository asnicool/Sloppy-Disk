package models

import "time"

// Album represents an album in MPD database
type Album struct {
	ID         string `json:"id,omitempty"`
	Album      string `json:"album,omitempty"`
	Artist     string `json:"artist,omitempty"`
	Title      string `json:"title,omitempty"`
	Date       string `json:"date,omitempty"`
	Genre      string `json:"genre,omitempty"`
	TrackCount int    `json:"trackCount,omitempty"`
	Duration   int    `json:"duration,omitempty"` // in seconds
	Path       string `json:"path,omitempty"`
	CoverURL   string `json:"coverUrl,omitempty"`
}

// Song represents a song with metadata
type Song struct {
	Title    string `json:"title"`
	Artist   string `json:"artist"`
	Album    string `json:"album"`
	Track    string `json:"track,omitempty"`
	Disc     string `json:"disc,omitempty"`
	Date     string `json:"date,omitempty"`
	Genre    string `json:"genre,omitempty"`
	Duration int    `json:"duration"`
	Path     string `json:"path"`
	Pos      int    `json:"pos,omitempty"` // Position in playlist
}

// PlaylistItem represents a song in the playlist
type PlaylistItem struct {
	Pos      int    `json:"pos"`
	Title    string `json:"title,omitempty"`
	Artist   string `json:"artist,omitempty"`
	Album    string `json:"album,omitempty"`
	Track    string `json:"track,omitempty"`
	Date     string `json:"date,omitempty"`
	Genre    string `json:"genre,omitempty"`
	Duration int    `json:"duration"`
	Path     string `json:"path"`
	CoverURL string `json:"coverUrl,omitempty"`
}

// PlaylistInfo represents the full playlist state for the frontend
type PlaylistInfo struct {
	Items      []PlaylistItem `json:"items"`
	Length     int            `json:"length"`
	CurrentPos int            `json:"currentPos"`
}

// MPDStatus represents current MPD playback status
type MPDStatus struct {
	State           string  `json:"state"` // play, pause, stop
	CurrentSong     Song    `json:"currentSong"`
	Elapsed         float64 `json:"elapsed"`  // seconds
	Duration        float64 `json:"duration"` // seconds
	Volume          int     `json:"volume"`   // 0-100
	Random          bool    `json:"random"`
	Repeat          bool    `json:"repeat"`
	Single          bool    `json:"single"`
	Consume         bool    `json:"consume"`
	Playlist        int     `json:"playlist"`        // playlist length (backward compat)
	PlaylistLength  int     `json:"playlistLength"`  // playlist length
	PlaylistVersion int     `json:"playlistVersion"` // playlist version
	PlaylistPos     int     `json:"playlistPos"`     // current position
}

// MetadataCandidate represents a metadata record from external sources
type MetadataCandidate struct {
	Source     string                 `json:"source"`
	Artist     string                 `json:"artist"`
	Album      string                 `json:"album"`
	Year       string                 `json:"year,omitempty"`
	Genre      string                 `json:"genre,omitempty"`
	Tracks     []Song                 `json:"tracks,omitempty"`
	ExternalID string                 `json:"externalId,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	Confidence float64                `json:"confidence,omitempty"`
}

// CoverArtCandidate represents a cover art image from external sources
type CoverArtCandidate struct {
	Source    string `json:"source"`
	URL       string `json:"url"`
	Thumbnail string `json:"thumbnail,omitempty"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`
	Size      string `json:"size,omitempty"`
}

// SyncStatus represents the status of rsync synchronization
type SyncStatus struct {
	IsRunning   bool      `json:"isRunning"`
	LastRun     time.Time `json:"lastRun"`
	LastSuccess bool      `json:"lastSuccess"`
	LastError   string    `json:"lastError,omitempty"`
	Progress    float64   `json:"progress"`
}

// APIResponse is the standard API response structure
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
}

// PaginationMeta defines pagination limits for MPD queries
type PaginationMeta struct {
	Page     int  `json:"page"`
	Limit    int  `json:"limit"`
	Total    int  `json:"total"`
	HasMore  bool `json:"hasMore"`
	NextPage *int `json:"nextPage,omitempty"`
	PrevPage *int `json:"prevPage,omitempty"`
}

// AlbumKey uniquely identifies an album by name and artist
type AlbumKey struct {
	Album       string
	AlbumArtist string
	Date        string
	Genre       string
	Path        string
}

// AlbumStats contains aggregate statistics for an album
type AlbumStats struct {
	TrackCount    int
	TotalDuration int // in seconds
}

// WSMessage represents a generic message sent over WebSocket
type WSMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// ArtistGroup represents an artist and their albums
type ArtistGroup struct {
	Artist string   `json:"artist"`
	Albums []string `json:"albums"`
}

// GroupedResult represents a generic grouping (Genre, Date, etc.)
type GroupedResult struct {
	Key    string   `json:"key"`
	Albums []string `json:"albums"`
}

// MetadataField represents a field that can be provided by metadata sources
type MetadataField string

const (
	FieldArtistID MetadataField = "artist_id"
	FieldAlbumID  MetadataField = "album_id"
	FieldMBID     MetadataField = "musicbrainz_id"
	FieldDiscogs  MetadataField = "discogs_id"
	FieldLastFM   MetadataField = "lastfm_id"
)

// AlbumMetadata represents enriched album metadata from external sources
type AlbumMetadata struct {
	Artist     string                 `json:"artist"`
	Album      string                 `json:"album"`
	Year       int                    `json:"year,omitempty"`
	Genre      string                 `json:"genre,omitempty"`
	Tracks     []Song                 `json:"tracks,omitempty"`
	Provider   string                 `json:"provider"`
	Confidence float64                `json:"confidence,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// ArtistImageCandidate represents an artist image from external sources
type ArtistImageCandidate struct {
	Source    string `json:"source"`
	URL       string `json:"url"`
	Thumbnail string `json:"thumbnail,omitempty"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`
	License   string `json:"license,omitempty"`
}
