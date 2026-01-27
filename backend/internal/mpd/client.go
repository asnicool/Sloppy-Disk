package mpd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"mpd-client-modern/internal/config"
	"mpd-client-modern/internal/models"
)

type Connection struct {
	mu               sync.Mutex
	conn             net.Conn
	reader           *bufio.Reader
	writer           *bufio.Writer
	lastUsed         time.Time
	isConnected      bool
	configVersion    int64         // Track config changes to detect when to reconnect
	isIdleClient     bool          // Flag to indicate if this client is used for IDLE operations
	idleTimeout      time.Duration // Timeout for idle connections, if needed
	DisableBroadcast bool          // If true, connection changes won't be broadcasted
}

type connectionPool struct {
	conns chan *Connection
	max   int
}

type Client struct {
	pool *connectionPool
}

var (
	defaultPool  *Client
	idleClient   *Connection // Separate client for IDLE operations
	statusClient *Connection // Separate client for Status operations (non-blocking)
	poolOnce     sync.Once
	idleOnce     sync.Once
	statusOnce   sync.Once
)

// Create a new connection instance
func NewConnection() *Connection {
	return &Connection{
		DisableBroadcast: true,
	}
}

// NewStandaloneConnection returns a new connection instance
func NewStandaloneConnection() *Connection {
	return NewConnection()
}

// NewIdleClient creates a new connection specifically for IDLE operations
func NewIdleClient() *Connection {
	return &Connection{
		isIdleClient:     true,
		idleTimeout:      24 * time.Hour, // 24-hour timeout for idle connections
		DisableBroadcast: true,           // Idle client shouldn't broadcast global status on reconnects
	}
}

// NewStatusClient creates a new connection specifically for Status operations
func NewStatusClient() *Connection {
	return &Connection{}
}

func GetPool() *Client {
	poolOnce.Do(func() {
		defaultPool = &Client{
			pool: &connectionPool{
				conns: make(chan *Connection, 10),
				max:   10,
			},
		}
	})
	return defaultPool
}

func GetClient() *Client {
	return GetPool()
}

func GetStatusClient() *Connection {
	statusOnce.Do(func() {
		statusClient = &Connection{}
	})
	return statusClient
}

func (c *Client) acquire() *Connection {
	select {
	case conn := <-c.pool.conns:
		return conn
	default:
		return NewConnection()
	}
}

func (c *Client) release(conn *Connection) {
	if conn == nil {
		return
	}
	// Only put back if connected and not an error-prone connection
	if conn.isConnected {
		select {
		case c.pool.conns <- conn:
			return
		default:
			// Pool full
		}
	}
	conn.Close()
}

func (c *Connection) EnsureConnection() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		// Check if connection is still alive
		c.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		one := make([]byte, 1)
		if _, err := c.conn.Read(one); err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// Still alive
				c.conn.SetReadDeadline(time.Time{})
			} else {
				c.conn.Close()
				c.conn = nil
				c.isConnected = false
			}
		} else {
			// Should not happen if we are just checking
			c.conn.SetReadDeadline(time.Time{})
		}
	}

	if c.conn == nil {
		cfg := config.Get()
		addr := net.JoinHostPort(cfg.MPDHost, fmt.Sprintf("%d", cfg.MPDPort))
		conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
		if err != nil {
			// Only broadcast connection failure if we were previously connected
			if c.isConnected {
				c.isConnected = false
				if !c.DisableBroadcast {
					c.broadcastConnectionStatus(false)
				}
			}
			return fmt.Errorf("failed to connect to MPD: %w", err)
		}

		c.conn = conn
		c.reader = bufio.NewReaderSize(conn, 128*1024) // 128KB buffer to handle large responses
		c.writer = bufio.NewWriterSize(conn, 32*1024)  // 32KB write buffer
		c.lastUsed = time.Now()
		c.isConnected = true

		// Set timeout for reading greeting
		c.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		line, err := c.reader.ReadString('\n')
		if err != nil {
			c.conn.Close()
			c.conn = nil
			c.isConnected = false
			return fmt.Errorf("failed to read MPD greeting: %w", err)
		}
		// Reset deadline after successful greeting
		c.conn.SetReadDeadline(time.Time{})
		if !strings.HasPrefix(line, "OK MPD") {
			c.conn.Close()
			c.conn = nil
			c.isConnected = false
			return fmt.Errorf("unexpected MPD greeting: %s", line)
		}

		// Authenticate if password is set
		if cfg.MPDPassword != "" {
			if _, err := c.sendCommandLocked(fmt.Sprintf("password %s", cfg.MPDPassword)); err != nil {
				c.conn.Close()
				c.conn = nil
				c.isConnected = false
				return fmt.Errorf("MPD authentication failed: %w", err)
			}
		}

		// Broadcast successful connection
		if !c.DisableBroadcast {
			c.broadcastConnectionStatus(true)
		}
	}

	return nil
}

// Close closes the underlying connection
func (c *Connection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		c.isConnected = false
		return err
	}
	return nil
}

// SetConnectionStatusCallback allows setting a callback function to broadcast connection status
// This helps avoid circular import between mpd and api packages
var connectionStatusCallback func(*models.MPDStatus)

func SetConnectionStatusCallback(callback func(*models.MPDStatus)) {
	connectionStatusCallback = callback
}

// broadcastConnectionStatus sends the connection status to all WebSocket clients via callback
func (c *Connection) broadcastConnectionStatus(connected bool) {
	// Create a minimal status with connection info to avoid hanging on full status retrieval
	status := &models.MPDStatus{
		State: "disconnected",
	}

	if connected {
		// Only set state to connected if we know we're connected
		status.State = "connected"
	}

	if connectionStatusCallback != nil {
		connectionStatusCallback(status)
	}
}

func (c *Connection) SendCommand(command string) (string, error) {
	if err := c.EnsureConnection(); err != nil {
		return "", err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	return c.sendCommandLocked(command)
}

func (c *Connection) sendCommandLocked(command string) (string, error) {
	// Ensure connection is still valid before sending command
	if c.conn == nil {
		return "", fmt.Errorf("no connection available")
	}

	// Set write deadline to prevent hanging
	c.conn.SetWriteDeadline(time.Now().Add(30 * time.Second))
	if _, err := c.writer.WriteString(command + "\n"); err != nil {
		c.conn.Close()
		c.conn = nil
		return "", fmt.Errorf("write error: %w", err)
	}
	if err := c.writer.Flush(); err != nil {
		c.conn.Close()
		c.conn = nil
		return "", fmt.Errorf("flush error: %w", err)
	}

	var response strings.Builder
	cmdStart := time.Now()
	for {
		// Set read timeout to prevent hanging
		c.conn.SetReadDeadline(time.Now().Add(30 * time.Second))
		line, err := c.reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				// Connection was closed by MPD server
				c.conn.Close()
				c.conn = nil
				return response.String(), fmt.Errorf("connection closed by server: %w", err)
			}
			// Check if it's a timeout error
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				c.conn.Close()
				c.conn = nil
				return response.String(), fmt.Errorf("read timeout: %w", err)
			}
			c.conn.Close()
			c.conn = nil
			return response.String(), fmt.Errorf("read error: %w", err)
		}

		if line == "OK\n" {
			break
		}
		if strings.HasPrefix(line, "ACK") {
			c.conn.Close()
			c.conn = nil
			return response.String(), fmt.Errorf("MPD error: %s", strings.TrimSpace(line))
		}
		response.WriteString(line)
	}

	duration := time.Since(cmdStart)
	if duration > 100*time.Millisecond {
		log.Printf("[MPD] Command '%s' took %v", command, duration)
	}

	c.lastUsed = time.Now()
	return response.String(), nil
}

// Path Mapping
func (c *Connection) ToAbsolutePath(relativePath string) string {
	cfg := config.Get()
	return filepath.Join(cfg.MusicRoot, relativePath)
}

func (c *Connection) ToRelativePath(absolutePath string) (string, error) {
	cfg := config.Get()
	rel, err := filepath.Rel(cfg.MusicRoot, absolutePath)
	if err != nil {
		return "", err
	}
	return rel, nil
}

// GetStatus retrieves current MPD status using a command list for efficiency
// This follows the MPD best practice: send status and currentsong in one batch
func (c *Connection) GetStatus() (*models.MPDStatus, error) {
	// Use a context with timeout to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	done := make(chan struct{})
	var status *models.MPDStatus
	var err error

	go func() {
		defer close(done)
		// Use command list to get status and currentsong in a single round trip
		commands := []string{"status", "currentsong"}
		responses, cmdErr := c.SendCommandList(commands)
		if cmdErr != nil {
			err = cmdErr
			return
		}

		if len(responses) != 2 {
			err = fmt.Errorf("expected 2 responses, got %d", len(responses))
			return
		}

		// Parse status response
		attrs := ParseResponse(responses[0])
		
		// Parse currentsong response
		songAttrs := ParseResponse(responses[1])

		status = &models.MPDStatus{
			State:           attrs["state"],
			Elapsed:         parseFloat(attrs["elapsed"]),
			Duration:        parseFloat(attrs["duration"]),
			Volume:          parseInt(attrs["volume"]),
			Random:          attrs["random"] == "1",
			Repeat:          attrs["repeat"] == "1",
			Single:          attrs["single"] == "1",
			Consume:         attrs["consume"] == "1",
			Playlist:        parseInt(attrs["playlistlength"]),
			PlaylistLength:  parseInt(attrs["playlistlength"]),
			PlaylistVersion: parseInt(attrs["playlist"]),
			PlaylistPos:     parseInt(attrs["song"]),
			CurrentSong: models.Song{
				Title:    songAttrs["Title"],
				Artist:   songAttrs["Artist"],
				Album:    songAttrs["Album"],
				Track:    songAttrs["Track"],
				Date:     songAttrs["Date"],
				Genre:    songAttrs["Genre"],
				Duration: parseInt(songAttrs["duration"]),
				Path:     songAttrs["file"],
			},
		}
	}()

	select {
	case <-done:
		return status, err
	case <-ctx.Done():
		return nil, fmt.Errorf("GetStatus timed out: %w", ctx.Err())
	}
}

// SendCommandList sends multiple commands as a single atomic operation
func (c *Connection) SendCommandList(commands []string) ([]string, error) {
	if err := c.EnsureConnection(); err != nil {
		return nil, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Set read/write timeouts to prevent hanging, especially for large batches
	c.conn.SetDeadline(time.Now().Add(30 * time.Second))

	// Send command list begin
	if _, err := c.writer.WriteString("command_list_ok_begin\n"); err != nil {
		return nil, err
	}

	// Send all commands
	for _, cmd := range commands {
		if _, err := c.writer.WriteString(cmd + "\n"); err != nil {
			return nil, err
		}
	}

	// Send command list end
	if _, err := c.writer.WriteString("command_list_end\n"); err != nil {
		return nil, err
	}

	if err := c.writer.Flush(); err != nil {
		return nil, err
	}

	// Read responses
	var responses []string
	var currentResponse strings.Builder
	cmdListStart := time.Now()
	for {
		line, err := c.reader.ReadString('\n')
		if err != nil {
			c.conn.Close()
			c.conn = nil
			return nil, err
		}

		if line == "OK\n" {
			// This is the end of the command list
			break
		} else if line == "list_OK\n" {
			// End of current command response
			responses = append(responses, currentResponse.String())
			currentResponse.Reset()
		} else if strings.HasPrefix(line, "ACK") {
			c.conn.Close()
			c.conn = nil
			return nil, fmt.Errorf("MPD error: %s", strings.TrimSpace(line))
		} else {
			// Add line to current response
			currentResponse.WriteString(line)
		}
	}

	duration := time.Since(cmdListStart)
	if duration > 100*time.Millisecond {
		log.Printf("[MPD] CommandList (%d cmds) took %v", len(commands), duration)
	}

	c.lastUsed = time.Now()
	return responses, nil
}

// GetAlbumsWithArtistAndDate gets albums with artist and date using efficient grouping
func (c *Connection) GetAlbumsWithArtistAndDate() ([]map[string]string, error) {
	resp, err := c.SendCommand("list album group artist group date")
	if err != nil {
		return nil, err
	}

	// Parse the response into album records
	lines := strings.Split(strings.TrimSpace(resp), "\n")
	var albums []map[string]string
	currentAlbum := make(map[string]string)

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ": ", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		switch key {
		case "Album":
			// Save previous album if exists
			if len(currentAlbum) > 0 {
				albums = append(albums, currentAlbum)
			}
			// Start new album
			currentAlbum = make(map[string]string)
			currentAlbum["Album"] = value
		case "Artist", "AlbumArtist":
			currentAlbum[key] = value
		case "Date":
			currentAlbum[key] = value
		}
	}

	// Add the last album
	if len(currentAlbum) > 0 {
		albums = append(albums, currentAlbum)
	}

	return albums, nil
}

// FindAlbumsByFilter finds albums based on a specific filter (e.g., "artist", "album")
// It executes: list album <filterTag> <filterValue> group artist group date
func (c *Connection) FindAlbumsByFilter(filterTag, filterValue string) ([]models.Album, error) {
	// sanitize and quote the value
	escapedValue := strings.ReplaceAll(filterValue, "\"", "\\\"")
	// The correct syntax is: list album [filter_tag] [filter_value] group artist group date
	// Example: list album artist "Al Green" group artist group date
	cmd := fmt.Sprintf("list album \"%s\" \"%s\" group artist group date", filterTag, escapedValue)

	resp, err := c.SendCommand(cmd)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(resp), "\n")
	var albums []models.Album
	// MPD list output with grouping maintains state (Artist/Date are printed once for a group)
	var currentArtist, currentDate string

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ": ", 2)
		if len(parts) != 2 {
			continue
		}

		key := parts[0]
		value := parts[1]

		switch key {
		case "Album":
			// Album is the item we are listing, so when we see it, we emit a new album
			// using the current context (Artist/Date)
			albums = append(albums, models.Album{
				Album:  value,
				Artist: currentArtist,
				Date:   currentDate,
			})
		case "Artist", "AlbumArtist":
			currentArtist = value
		case "Date":
			currentDate = value
		}
	}

	// Post-processing: Generate IDs
	for i := range albums {
		if albums[i].Artist == "" {
			albums[i].Artist = "Unknown Artist"
		}
		// Generate ID consistent with cache
		albums[i].ID = fmt.Sprintf("%x", fmt.Sprintf("%s|%s", albums[i].Artist, albums[i].Album))
	}

	return albums, nil
}

// GetDetailedAlbumInfo gets detailed album information for multiple paths efficiently
func (c *Connection) GetDetailedAlbumInfo(paths []string) (map[string][]map[string]string, error) {
	// Build command list to get info for all paths
	commands := make([]string, len(paths))
	for i, path := range paths {
		commands[i] = "listallinfo \"" + strings.ReplaceAll(path, "\"", "\\\"") + "\""
	}

	responses, err := c.SendCommandList(commands)
	if err != nil {
		return nil, err
	}

	// Parse all responses
	result := make(map[string][]map[string]string)
	for i, response := range responses {
		path := paths[i]
		lines := strings.Split(strings.TrimSpace(response), "\n")
		var songs []map[string]string
		var currentSong map[string]string

		for _, line := range lines {
			if line == "" {
				continue
			}

			parts := strings.SplitN(line, ": ", 2)
			if len(parts) != 2 {
				continue
			}

			key := parts[0]
			value := parts[1]

			if key == "file" {
				// New song starts
				if len(currentSong) > 0 {
					songs = append(songs, currentSong)
				}
				currentSong = make(map[string]string)
				currentSong[key] = value
			} else {
				if currentSong != nil {
					currentSong[key] = value
				}
			}
		}

		// Add the last song
		if len(currentSong) > 0 {
			songs = append(songs, currentSong)
		}

		result[path] = songs
	}

	return result, nil
}

func (c *Connection) UpdateDB(path string) error {
	cmd := "update"
	if path != "" {
		cmd = fmt.Sprintf("update \"%s\"", strings.ReplaceAll(path, "\"", "\\\""))
	}
	_, err := c.SendCommand(cmd)
	return err
}

// Utils
func ParseResponse(response string) map[string]string {
	attrs := make(map[string]string)
	lines := strings.Split(strings.TrimSpace(response), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) == 2 {
			attrs[parts[0]] = parts[1]
		}
	}
	return attrs
}

func parseInt(s string) int {
	var i int
	fmt.Sscanf(s, "%d", &i)
	return i
}

func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

// GetCoverArtURL returns the cover art URL for a given song path
func (c *Connection) GetCoverArtURL(songPath string) string {
	// Convert song path to relative path and then to cover art path
	relPath, err := c.ToRelativePath(songPath)
	if err != nil {
		// Fallback: try to find cover art in the same directory as the song
		relPath = songPath
	}

	// Get the directory containing the song
	dir := filepath.Dir(relPath)

	cfg := config.Get()
	coverPath := filepath.Join(cfg.CoverArtRoot, dir, "Folder.jpg")

	// Check if file exists
	if _, err := os.Stat(coverPath); err == nil {
		// Return URL-encoded path for the cover art endpoint
		return "/api/coverart/" + dir
	}

	// Also check for .png
	coverPathPng := filepath.Join(cfg.CoverArtRoot, dir, "Folder.png")
	if _, err := os.Stat(coverPathPng); err == nil {
		return "/api/coverart/" + dir
	}

	return ""
}

// GetPlaylist returns the current playlist with all songs and their positions
func (c *Connection) GetPlaylist() ([]models.PlaylistItem, error) {
	// Use command list to get playlist info efficiently
	// First get playlistinfo to get all songs
	resp, err := c.SendCommand("playlistinfo")
	if err != nil {
		return nil, err
	}

	return c.parsePlaylistResponse(resp)
}

// GetPlaylistRange returns a range of playlist items (for lazy loading)
func (c *Connection) GetPlaylistRange(start, end int) ([]models.PlaylistItem, error) {
	cmd := fmt.Sprintf("playlistinfo %d:%d", start, end)
	resp, err := c.SendCommand(cmd)
	if err != nil {
		return nil, err
	}

	return c.parsePlaylistResponse(resp)
}

// GetPlaylistLength returns the total number of songs in the playlist
func (c *Connection) GetPlaylistLength() (int, error) {
	resp, err := c.SendCommand("playlistlength")
	if err != nil {
		return 0, err
	}
	return parseInt(resp), nil
}

func (c *Connection) parsePlaylistResponse(response string) ([]models.PlaylistItem, error) {
	lines := strings.Split(strings.TrimSpace(response), "\n")
	var items []models.PlaylistItem
	var currentItem *models.PlaylistItem

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ": ", 2)
		if len(parts) != 2 {
			continue
		}

		key, value := parts[0], parts[1]

		if key == "file" {
			// Save previous item if exists
			if currentItem != nil {
				items = append(items, *currentItem)
			}
			// Start new item
			currentItem = &models.PlaylistItem{
				Path: value,
			}
		} else if currentItem != nil {
			switch key {
			case "Title":
				currentItem.Title = value
			case "Artist":
				currentItem.Artist = value
			case "Album":
				currentItem.Album = value
			case "Track":
				currentItem.Track = value
			case "Date":
				currentItem.Date = value
			case "Genre":
				currentItem.Genre = value
			case "Duration":
				currentItem.Duration = parseInt(value)
			case "Pos":
				currentItem.Pos = parseInt(value)
			}
		}
	}

	// Add the last item
	if currentItem != nil {
		items = append(items, *currentItem)
	}

	// Enrich each item with cover art URL
	for i := range items {
		items[i].CoverURL = c.GetCoverArtURL(items[i].Path)
	}

	return items, nil
}

// Move moves a song in the playlist from one position to another
func (c *Connection) Move(from, to int) error {
	cmd := fmt.Sprintf("move %d %d", from, to)
	_, err := c.SendCommand(cmd)
	return err
}

// MoveRange moves a range of songs [start, end) to a new position
func (c *Connection) MoveRange(start, end, to int) error {
	// MPD syntax for range is start:end
	cmd := fmt.Sprintf("move %d:%d %d", start, end, to)
	_, err := c.SendCommand(cmd)
	return err
}

// DeleteId deletes a song from the playlist by its position (id is confusing in MPD, usually it's delete <pos> or deleteid <id>)
// For drag and drop we usually use positions. MPD 'delete' command takes a position.
func (c *Connection) Delete(pos int) error {
	cmd := fmt.Sprintf("delete %d", pos)
	_, err := c.SendCommand(cmd)
	return err
}

// Idle waits for MPD to notify about changes
// Returns the list of changed subsystems or an error
func (c *Connection) Idle(subsystems ...string) ([]string, error) {
	if err := c.EnsureConnection(); err != nil {
		return nil, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Build the idle command
	cmd := "idle"
	if len(subsystems) > 0 {
		cmd += " " + strings.Join(subsystems, " ")
	}

	// Set write deadline for the idle command
	c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if _, err := c.writer.WriteString(cmd + "\n"); err != nil {
		return nil, err
	}
	if err := c.writer.Flush(); err != nil {
		return nil, err
	}
	// Reset write deadline
	c.conn.SetWriteDeadline(time.Time{})

	var changedSubsystems []string
	for {
		// For idle connections, we don't want a timeout since idle is meant to wait indefinitely
		// Only set a timeout if this is not an idle client or if an idle timeout is configured
		if !c.isIdleClient || c.idleTimeout > 0 {
			timeout := c.idleTimeout
			if timeout == 0 {
				timeout = 30 * time.Second // fallback timeout if not specified
			}
			c.conn.SetReadDeadline(time.Now().Add(timeout))
		} else {
			// No timeout for dedicated idle connections
			c.conn.SetReadDeadline(time.Time{})
		}
		line, err := c.reader.ReadString('\n')
		if err != nil {
			c.conn.Close()
			c.conn = nil
			c.isConnected = false
			return nil, err
		}

		line = strings.TrimSpace(line)

		// Check for changed subsystem notifications ("changed: <subsystem>")
		if strings.HasPrefix(line, "changed: ") {
			subsystem := strings.TrimSpace(strings.TrimPrefix(line, "changed: "))
			changedSubsystems = append(changedSubsystems, subsystem)
		} else if line == "OK" {
			// End of idle response
			break
		} else if strings.HasPrefix(line, "ACK") {
			return changedSubsystems, fmt.Errorf("MPD error: %s", strings.TrimSpace(line))
		}

	}

	return changedSubsystems, nil
}

// IsConnected returns whether the client is currently connected to the MPD server
func (c *Connection) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.isConnected
}

// NoIdle exits idle mode and returns to normal command mode
func (c *Connection) NoIdle() (string, error) {
	// Ensure connection is valid first
	if err := c.EnsureConnection(); err != nil {
		return "", fmt.Errorf("failed to establish connection for noidle: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Set write deadline before sending the command (important for idle connections)
	c.conn.SetWriteDeadline(time.Now().Add(5 * time.Second))

	// Send the noidle command
	if _, err := c.writer.WriteString("noidle\n"); err != nil {
		// If write fails, close the connection so it will be re-established on next use
		c.conn.Close()
		c.conn = nil
		c.isConnected = false
		return "", fmt.Errorf("write error during noidle: %w", err)
	}

	if err := c.writer.Flush(); err != nil {
		// If flush fails, close the connection so it will be re-established on next use
		c.conn.Close()
		c.conn = nil
		c.isConnected = false
		return "", fmt.Errorf("flush error during noidle: %w", err)
	}

	// Now read the response with a short timeout
	c.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	var response strings.Builder
	for {
		line, err := c.reader.ReadString('\n')
		if err != nil {
			// If read fails, close the connection so it will be re-established on next use
			c.conn.Close()
			c.conn = nil
			c.isConnected = false
			return "", fmt.Errorf("read error during noidle: %w", err)
		}

		if line == "OK\n" {
			break
		}
		if strings.HasPrefix(line, "ACK") {
			return response.String(), fmt.Errorf("MPD error during noidle: %s", strings.TrimSpace(line))
		}
		response.WriteString(line)
	}

	// Reset deadlines after successful noidle
	c.conn.SetReadDeadline(time.Time{})
	c.conn.SetWriteDeadline(time.Time{})

	return response.String(), nil
}

// ResetClient resets the MPD client connection to force reconnection with new settings
func ResetClient() {
	if defaultPool != nil {
		// Drain the pool
		for {
			select {
			case conn := <-defaultPool.pool.conns:
				conn.Close()
			default:
				return
			}
		}
	}
}

// Proxy methods for Client (Pool Manager)
func (c *Client) SendCommand(command string) (string, error) {
	var resp string
	err := c.Execute(func(conn *Connection) error {
		var execErr error
		resp, execErr = conn.SendCommand(command)
		return execErr
	})
	return resp, err
}

func (c *Client) Execute(fn func(*Connection) error) error {
	conn := c.acquire()
	defer c.release(conn)
	return fn(conn)
}

func (c *Client) SendCommandList(commands []string) ([]string, error) {
	var resps []string
	err := c.Execute(func(conn *Connection) error {
		var execErr error
		resps, execErr = conn.SendCommandList(commands)
		return execErr
	})
	return resps, err
}

func (c *Client) GetStatus() (*models.MPDStatus, error) {
	var status *models.MPDStatus
	err := c.Execute(func(conn *Connection) error {
		var execErr error
		status, execErr = conn.GetStatus()
		return execErr
	})
	return status, err
}

func (c *Client) GetAlbumsWithArtistAndDate() ([]map[string]string, error) {
	conn := c.acquire()
	defer c.release(conn)
	return conn.GetAlbumsWithArtistAndDate()
}

func (c *Client) GetDetailedAlbumInfo(paths []string) (map[string][]map[string]string, error) {
	conn := c.acquire()
	defer c.release(conn)
	return conn.GetDetailedAlbumInfo(paths)
}

func (c *Client) UpdateDB(path string) error {
	conn := c.acquire()
	defer c.release(conn)
	return conn.UpdateDB(path)
}

func (c *Client) GetPlaylist() ([]models.PlaylistItem, error) {
	conn := c.acquire()
	defer c.release(conn)
	return conn.GetPlaylist()
}

func (c *Client) GetPlaylistRange(start, end int) ([]models.PlaylistItem, error) {
	conn := c.acquire()
	defer c.release(conn)
	return conn.GetPlaylistRange(start, end)
}

func (c *Client) GetPlaylistLength() (int, error) {
	conn := c.acquire()
	defer c.release(conn)
	return conn.GetPlaylistLength()
}

func (c *Client) Move(from, to int) error {
	conn := c.acquire()
	defer c.release(conn)
	return conn.Move(from, to)
}

func (c *Client) MoveRange(start, end, to int) error {
	conn := c.acquire()
	defer c.release(conn)
	return conn.MoveRange(start, end, to)
}

func (c *Client) Delete(pos int) error {
	conn := c.acquire()
	defer c.release(conn)
	return conn.Delete(pos)
}

func (c *Client) Idle(subsystems ...string) ([]string, error) {
	conn := c.acquire()
	defer c.release(conn)
	return conn.Idle(subsystems...)
}

func (c *Client) IsConnected() bool {
	// For pooled client, we consider it connected if we can get a connection
	// but for simplicity, we just check if it's initialized
	return c.pool != nil
}

func (c *Client) NoIdle() (string, error) {
	conn := c.acquire()
	defer c.release(conn)
	return conn.NoIdle()
}

func (c *Client) GetAllAlbumKeys() ([]models.AlbumKey, error) {
	conn := c.acquire()
	defer c.release(conn)
	return conn.GetAllAlbumKeys()
}

func (c *Client) GetAlbumStats(albumKeys []models.AlbumKey) (map[models.AlbumKey]models.AlbumStats, error) {
	conn := c.acquire()
	defer c.release(conn)
	return conn.GetAlbumStats(albumKeys)
}

func (c *Client) GetAlbumRepresentative(key models.AlbumKey) (*models.Song, error) {
	conn := c.acquire()
	defer c.release(conn)
	return conn.GetAlbumRepresentative(key)
}

func (c *Client) GetAlbumRepresentatives(keys []models.AlbumKey) (map[models.AlbumKey]*models.Song, error) {
	conn := c.acquire()
	defer c.release(conn)
	return conn.GetAlbumRepresentatives(keys)
}

func (c *Client) ToAbsolutePath(relativePath string) string {
	return NewStandaloneConnection().ToAbsolutePath(relativePath)
}

func (c *Client) FindAlbumsByFilter(filterTag, filterValue string) ([]models.Album, error) {
	conn := c.acquire()
	defer c.release(conn)
	return conn.FindAlbumsByFilter(filterTag, filterValue)
}

func (c *Client) ToRelativePath(absolutePath string) (string, error) {
	return NewStandaloneConnection().ToRelativePath(absolutePath)
}

func (c *Client) GetCoverArtURL(songPath string) string {
	return NewStandaloneConnection().GetCoverArtURL(songPath)
}

// There's a duplicate method, so I'll remove this one

// ResetConnection closes the current connection to allow reconnection
func (c *Connection) ResetConnection() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
		c.isConnected = false
	}
}

// GetAllAlbumKeys returns all unique album keys (album + albumartist + date)
func (c *Connection) GetAllAlbumKeys() ([]models.AlbumKey, error) {
	if err := c.EnsureConnection(); err != nil {
		return nil, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Set a very long timeout for this heavy operation
	c.conn.SetDeadline(time.Now().Add(120 * time.Second))

	// list album group albumartist group date group genre
	cmd := "list album group albumartist group date group genre"

	if _, err := c.writer.WriteString(cmd + "\n"); err != nil {
		c.conn.Close()
		c.conn = nil
		return nil, fmt.Errorf("write error: %w", err)
	}
	if err := c.writer.Flush(); err != nil {
		c.conn.Close()
		c.conn = nil
		return nil, fmt.Errorf("flush error: %w", err)
	}

	var response strings.Builder
	for {
		line, err := c.reader.ReadString('\n')
		if err != nil {
			c.conn.Close()
			c.conn = nil
			return nil, fmt.Errorf("read error: %w", err)
		}

		if line == "OK\n" {
			break
		}
		if strings.HasPrefix(line, "ACK") {
			c.conn.Close()
			c.conn = nil
			return nil, fmt.Errorf("MPD error: %s", strings.TrimSpace(line))
		}
		response.WriteString(line)
	}

	c.lastUsed = time.Now()
	resp := response.String()

	var keys []models.AlbumKey
	lines := strings.Split(strings.TrimSpace(resp), "\n")

	var currentAlbum, currentArtist, currentDate, currentGenre string
	keys = make([]models.AlbumKey, 0)

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
		case "Album":
			currentAlbum = value
			keys = append(keys, models.AlbumKey{
				Album:       currentAlbum,
				AlbumArtist: currentArtist,
				Date:        currentDate,
				Genre:       currentGenre,
			})
		case "AlbumArtist":
			currentArtist = value
		case "Date":
			currentDate = value
		case "Genre":
			currentGenre = value
		}
	}

	return keys, nil
}

// GetAlbumStats fetches track count and total duration for a batch of albums
func (c *Connection) GetAlbumStats(albumKeys []models.AlbumKey) (map[models.AlbumKey]models.AlbumStats, error) {
	if len(albumKeys) == 0 {
		return make(map[models.AlbumKey]models.AlbumStats), nil
	}

	var commands []string
	for _, key := range albumKeys {

		// Use simple tag-value pairs which are more reliably supported
		// count album "Album" albumartist "Artist"

		albumEsc := strings.ReplaceAll(key.Album, "\"", "\\\"")
		artistEsc := strings.ReplaceAll(key.AlbumArtist, "\"", "\\\"")

		cmd := fmt.Sprintf("count album \"%s\"", albumEsc)
		if key.AlbumArtist != "" {
			cmd += fmt.Sprintf(" albumartist \"%s\"", artistEsc)
		}

		commands = append(commands, cmd)
	}

	responses, err := c.SendCommandList(commands)
	if err != nil {
		return nil, err
	}

	statsMap := make(map[models.AlbumKey]models.AlbumStats)

	for i, resp := range responses {
		key := albumKeys[i]
		stats := models.AlbumStats{}

		lines := strings.Split(strings.TrimSpace(resp), "\n")
		for _, line := range lines {
			parts := strings.SplitN(line, ": ", 2)
			if len(parts) == 2 {
				switch parts[0] {
				case "songs":
					stats.TrackCount = parseInt(parts[1])
				case "playtime":
					stats.TotalDuration = parseInt(parts[1])
				}
			}
		}
		statsMap[key] = stats
	}

	return statsMap, nil
}

// GetAlbumRepresentative gets a single song from the album to extract metadata like Path and Date
func (c *Connection) GetAlbumRepresentative(key models.AlbumKey) (*models.Song, error) {
	// find album "Album1" albumartist "Artist1" window 0:1
	albumEsc := strings.ReplaceAll(key.Album, "\"", "\\\"")
	artistEsc := strings.ReplaceAll(key.AlbumArtist, "\"", "\\\"")

	cmd := fmt.Sprintf("find album \"%s\"", albumEsc)
	if key.AlbumArtist != "" {
		cmd += fmt.Sprintf(" albumartist \"%s\"", artistEsc)
	}
	cmd += " window 0:1"

	resp, err := c.SendCommand(cmd)
	if err != nil {
		return nil, err
	}

	attrs := ParseResponse(resp)
	if len(attrs) == 0 {
		return nil, nil // No song found
	}

	return &models.Song{
		Title:  attrs["Title"],
		Artist: attrs["Artist"],
		Album:  attrs["Album"],
		Date:   attrs["Date"],
		Path:   attrs["file"],
	}, nil
}

// GetAlbumRepresentatives gets representative songs for a batch of albums efficiently
func (c *Connection) GetAlbumRepresentatives(keys []models.AlbumKey) (map[models.AlbumKey]*models.Song, error) {
	if len(keys) == 0 {
		return make(map[models.AlbumKey]*models.Song), nil
	}

	var commands []string
	for _, key := range keys {
		// find album "Album1" albumartist "Artist1" window 0:1
		albumEsc := strings.ReplaceAll(key.Album, "\"", "\\\"")
		artistEsc := strings.ReplaceAll(key.AlbumArtist, "\"", "\\\"")

		cmd := fmt.Sprintf("find album \"%s\"", albumEsc)
		if key.AlbumArtist != "" {
			cmd += fmt.Sprintf(" albumartist \"%s\"", artistEsc)
		}
		cmd += " window 0:1"
		commands = append(commands, cmd)
	}

	responses, err := c.SendCommandList(commands)
	if err != nil {
		return nil, err
	}

	result := make(map[models.AlbumKey]*models.Song)

	for i, resp := range responses {
		key := keys[i]
		attrs := ParseResponse(resp)

		if len(attrs) > 0 {
			result[key] = &models.Song{
				Title:  attrs["Title"],
				Artist: attrs["Artist"],
				Album:  attrs["Album"],
				Date:   attrs["Date"],
				Path:   attrs["file"],
			}
		}
	}

	return result, nil
}
