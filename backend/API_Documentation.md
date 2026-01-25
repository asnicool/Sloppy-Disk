# MPD Client Modern - Backend API Documentation

This document lists all the API endpoints implemented in the Go backend for the MPD Client Modern application.

## API Endpoints

### Status and Playback

1. **GET /api/status**
   - Purpose: Returns current MPD playback status
   - Response: MPDStatus object with state, current song, elapsed time, volume, etc.

2. **POST /api/play**
   - Purpose: Sends play command to MPD
   - Response: Success confirmation

3. **POST /api/pause**
   - Purpose: Sends pause command to MPD
   - Response: Success confirmation

4. **POST /api/next**
   - Purpose: Sends next track command to MPD
   - Response: Success confirmation

5. **POST /api/previous**
   - Purpose: Sends previous track command to MPD
   - Response: Success confirmation

6. **POST /api/volume/{volume}**
   - Purpose: Sets MPD volume (0-100)
   - Parameters: volume (integer 0-100)
   - Response: Success confirmation

### Library Management

7. **GET /api/albums**
    - Purpose: Returns paginated list of albums
    - Query Parameters:
      - page (int): Page number (default 1)
      - limit (int): Items per page (default 50)
      - sort (string): Sort by "date" or "name" (default "name")
      - full (bool): If true, includes additional album details (path, track count, duration) - may take longer to process
    - Response: Paginated list of Album objects

8. **GET /api/artists**
   - Purpose: Returns paginated list of artists
   - Query Parameters:
     - page (int): Page number (default 1)
     - limit (int): Items per page (default 50)
     - search (string): Search term for artist names
   - Response: Paginated list of artist strings

9. **GET /api/album/{artist}/{album}**
   - Purpose: Returns songs for a specific album
   - Path Parameters:
     - artist: URL-encoded artist name
     - album: URL-encoded album name
   - Query Parameters:
     - page (int): Page number (default 1)
     - limit (int): Items per page (default 50)
   - Response: Paginated list of Song objects

10. **GET /api/search**
    - Purpose: Unified search across albums, artists, and songs
    - Query Parameters:
      - q (string): Search query (required)
      - type (string): Search type - "album", "artist", or "song"
      - page (int): Page number (default 1)
      - limit (int): Items per page (default 50)
    - Response: Paginated list of search results

### Playlist Management

11. **POST /api/playlist/add/{uri}**
    - Purpose: Adds a song to the playlist
    - Path Parameters:
      - uri: Song URI to add
    - Response: Success confirmation
    - Note: Currently implemented as mock

12. **POST /api/playlist/remove/{pos}**
    - Purpose: Removes a song from playlist by position
    - Path Parameters:
      - pos (int): Position in playlist (0-based)
    - Response: Success confirmation
    - Note: Currently implemented as mock

### Configuration

13. **GET /api/config**
    - Purpose: Gets current MPD server configuration
    - Response: MPDConfig object with host and port

14. **POST /api/config**
    - Purpose: Updates MPD server configuration
    - Body: JSON with host (string) and port (int)
    - Response: Updated configuration

### Logging

15. **GET /api/logs**
    - Purpose: Gets recent application logs
    - Query Parameters:
      - limit (int): Number of logs to return (default 100, max 1000)
    - Response: Array of LogEntry objects

### Real-time Updates

16. **WebSocket /ws**
    - Purpose: Real-time MPD status updates
    - Messages: JSON status updates every second

17. **WebSocket /ws/logs**
    - Purpose: Real-time log streaming
    - Messages: JSON log entries every 2 seconds

### Static Content

18. **GET /simple.html**
    - Purpose: Serves the simple HTML interface
    - Response: HTML content

19. **GET /**
    - Purpose: Serves the simple HTML interface (root path)
    - Response: HTML content

## Response Format

All API responses follow this structure:

```json
{
  "success": true|false,
  "data": <response_data>,
  "error": "<error_message>", // only present if success is false
  "meta": {
    "page": <current_page>,
    "limit": <items_per_page>,
    "total": <total_items>,
    "hasMore": true|false,
    "nextPage": <next_page_number>, // optional
    "prevPage": <prev_page_number>  // optional
  }
}
```

## Data Structures

### MPDStatus
```go
type MPDStatus struct {
    State       string  // "play", "pause", "stop"
    CurrentSong Song
    Elapsed     float64 // seconds
    Duration    float64 // seconds
    Volume      int     // 0-100
    Random      bool
    Repeat      bool
    Single      bool
    Consume     bool
    Playlist    int     // playlist length
    PlaylistPos int     // current position
}
```

### Song
```go
type Song struct {
    Title    string
    Artist   string
    Album    string
    Track    string // optional
    Date     string // optional
    Genre    string // optional
    Duration int    // seconds
    Path     string
    Pos      int    // position in playlist, optional
}
```

### Album
```go
type Album struct {
    Album       string `json:"album"`
    Artist      string `json:"artist"`  // added when using "list album group artist"
    Title       string `json:"title"`
    Date        string `json:"date,omitempty"`  // optional, only present when available
    Genre       string `json:"genre,omitempty"` // optional, only present when available
    TrackCount  int    `json:"trackCount"`      // only present when full=true
    Duration    int    `json:"duration"`        // total seconds, only present when full=true
    Path        string `json:"path"`            // unique album path, only present when full=true
    CoverURL    string `json:"coverUrl,omitempty"` // optional, only present when available
}
```

### Additional Notes for /api/albums
- When `full=true` parameter is provided, the response will include additional fields: `path`, `trackCount`, and `duration`
- When `full=true`, the API call takes longer as it needs to collect all the data and join it
- The `artist` field is now properly populated using MPD's "list album group artist" command
- Empty/omitted fields are not sent in the JSON response to reduce payload size
```

### MPDConfig
```go
type MPDConfig struct {
    Host string
    Port int
}
```

### LogEntry
```go
type LogEntry struct {
    Timestamp time.Time
    Level     string // "INFO", "ERROR", "DEBUG"
    Message   string
    Category  string // "SYSTEM", "MPD", "PLAYBACK", etc.
}
```

## Error Handling

- Invalid requests return HTTP 400 with error message
- Internal server errors return HTTP 500 with error message
- All errors include a JSON response with `success: false` and an `error` field

## CORS Support

All endpoints support CORS with:
- Allow-Origin: *
- Allow-Methods: GET, POST, PUT, DELETE, OPTIONS
- Allow-Headers: Content-Type, Authorization