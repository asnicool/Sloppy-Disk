xpackage api

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"mpd-client-modern/internal/models"
	"mpd-client-modern/internal/mpd"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 300 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
)

type ClientConnection struct {
	conn   *websocket.Conn
	wsSend chan models.WSMessage
	Ctx    context.Context    // Context to signal disconnection
	Cancel context.CancelFunc // Function to cancel the context
}

type Broadcaster struct {
	clients             map[*ClientConnection]bool
	register            chan *ClientConnection
	unregister          chan *ClientConnection
	mpdClient           *mpd.Client
	idleMpdClient       *mpd.Connection // Single dedicated client for IDLE operations
	idleClientConnected bool            // Track if idle client is connected
	ctx                 context.Context
	cancel              context.CancelFunc
	mu                  sync.RWMutex
}

func NewBroadcaster() *Broadcaster {
	ctx, cancel := context.WithCancel(context.Background())

	// Create a separate MPD client instance for IDLE operations
	idleClient := mpd.NewIdleClient()

	return &Broadcaster{
		clients:             make(map[*ClientConnection]bool),
		register:            make(chan *ClientConnection),
		unregister:          make(chan *ClientConnection),
		mpdClient:           mpd.GetClient(),
		idleMpdClient:       idleClient,
		idleClientConnected: false,
		ctx:                 ctx,
		cancel:              cancel,
	}
}

func (b *Broadcaster) Run() {
	go b.listenForMPDChanges()

	for {
		select {
		case client := <-b.register:
			b.mu.Lock()
			b.clients[client] = true
			b.mu.Unlock()

			// Send current status to the newly connected client
			// Use pooled client to avoid head-of-line blocking
			if status, err := b.mpdClient.GetStatus(); err == nil {
				select {
				case client.wsSend <- models.WSMessage{Type: "status", Data: status}:
				case <-client.Ctx.Done(): // Check if client disconnected
					b.mu.Lock()
					delete(b.clients, client)
					b.mu.Unlock()
				default:
					// If buffer is full, we can drop or disconnect.
					// For now, let's just drop the status update to avoid blocking.
					// We DO NOT close the channel here to avoid panics in other goroutines.
				}
			}

		case client := <-b.unregister:
			b.mu.Lock()
			// DO NOT close the channel. The client ctx is cancelled by the handler.
			// close(client.wsSend)
			delete(b.clients, client)
			b.mu.Unlock()

		case <-b.ctx.Done():
			return
		}
	}
}

func (b *Broadcaster) listenForMPDChanges() {
	ticker := time.NewTicker(60 * time.Second) // 60-second polling as backup
	defer ticker.Stop()

	// Initialize idle client connection
	b.ensureIdleClientConnection()

	for {
		select {
		case <-b.ctx.Done():
			return
		case <-ticker.C:
			// Periodic status check as backup when idle fails
			// Use pooled client to avoid head-of-line blocking
			if status, err := b.mpdClient.GetStatus(); err == nil {
				b.Broadcast(status)
			}
			b.ensureIdleClientConnection()
		default:
			// Use MPD idle command to wait for changes - use the dedicated idle client
			// First check if idle client is connected
			if !b.idleMpdClient.IsConnected() {
				b.ensureIdleClientConnection()
				if !b.idleMpdClient.IsConnected() {
					// If still not connected, use polling
					time.Sleep(time.Second)
					continue
				}
			}

			changedSubsystems, err := b.idleMpdClient.Idle()
			if err != nil {
				log.Printf("Error in MPD idle: %v", err)
				// Mark idle client as disconnected
				b.idleClientConnected = false
				// Attempt to reconnect with exponential backoff
				b.reconnectIdleClientWithBackoff()
				// Use pooled client to get status if idle failed (to avoid head-of-line blocking)
				if status, err := b.mpdClient.GetStatus(); err == nil {
					b.Broadcast(status)
				}
				continue
			}

			// Exit idle mode to get current status - use the dedicated idle client
			// Idle() returns when the idle state is finished (server sends OK), so we are already back in command mode.
			// No need to call NoIdle() here.

			// Handle database changes by triggering cache refresh
			databaseChanged := false
			for _, subsystem := range changedSubsystems {
				if subsystem == "database" {
					databaseChanged = true
					log.Println("[Broadcaster] Database changed, triggering cache refresh...")
					go func() {
						// Import albumcache package to trigger refresh
						// This avoids circular import by using a callback approach
						if databaseChangeCallback != nil {
							databaseChangeCallback()
						}
					}()
				}
			}

			// Broadcast database update to all clients
			if databaseChanged {
				b.BroadcastWS(models.WSMessage{
					Type: "database_update",
					Data: map[string]interface{}{
						"timestamp": time.Now().Unix(),
						"message":   "Music library has been updated",
					},
				})
			}

			// Only fetch status if relevant subsystems changed
			if b.shouldUpdateStatus(changedSubsystems) {
				log.Printf("[Broadcaster] Subsystems changed: %v", changedSubsystems)
				// Use pooled client to avoid head-of-line blocking
				status, err := b.mpdClient.GetStatus()
				if err != nil {
					log.Printf("Error getting MPD status: %v", err)
				} else {
					log.Printf("[Broadcaster] Broadcasting status update (playlist version: %d)", status.PlaylistVersion)
					b.Broadcast(status)
				}
			}

			// Reset ticker to start from fresh after an event
			ticker.Stop()
			ticker = time.NewTicker(60 * time.Second)
		}
	}
}

func (b *Broadcaster) shouldUpdateStatus(subsystems []string) bool {
	// Update status if any of these subsystems changed
	relevantSubsystems := map[string]bool{
		"player":          true, // playback state, elapsed time, etc.
		"mixer":           true, // volume changes
		"playlist":        true, // playlist changes
		"options":         true, // random, repeat, etc.
		"database":        true, // database updates
		"update":          true, // database updates in progress
		"stored_playlist": true, // stored playlist changes
		"output":          true, // audio output changes
		"partition":       true, // partition changes
		"sticker":         true, // sticker changes
		"subscription":    true, // subscription changes
		"message":         true, // message bus
	}

	for _, subsystem := range subsystems {
		if relevantSubsystems[subsystem] {
			return true
		}
	}
	return false
}

func (b *Broadcaster) Register(client *ClientConnection) {
	b.register <- client
}

func (b *Broadcaster) Unregister(client *ClientConnection) {
	b.unregister <- client
}

func (b *Broadcaster) Broadcast(status *models.MPDStatus) {
	b.BroadcastWS(models.WSMessage{
		Type: "status",
		Data: status,
	})
}

func (b *Broadcaster) BroadcastWS(msg models.WSMessage) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for client := range b.clients {
		// Use a non-blocking send or a goroutine per client to avoid blocking the broadcaster
		go func(c *ClientConnection, m models.WSMessage) {
			select {
			case c.wsSend <- m:
			case <-c.Ctx.Done(): // Stop if client disconnected
				return
			case <-time.After(100 * time.Millisecond):
				// Slow consumer, dropping message is better than blocking
			}
		}(client, msg)
	}
}

func (b *Broadcaster) Stop() {
	b.cancel()
}

// GetClientsCount returns the number of currently connected clients
func (b *Broadcaster) GetClientsCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.clients)
}

// SetDatabaseChangeCallback allows setting a callback function to handle database changes
// This helps avoid circular import between api and albumcache packages
var databaseChangeCallback func()

func SetDatabaseChangeCallback(callback func()) {
	databaseChangeCallback = callback
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Global broadcaster instance
var GlobalBroadcaster *Broadcaster

// Initialize the broadcaster when the package is loaded
func init() {
	GlobalBroadcaster = NewBroadcaster()
	go GlobalBroadcaster.Run()
}

// SearchWebSocketHandler handles dedicated search connections
func SearchWebSocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Search WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	// Create a context for this connection
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := &ClientConnection{
		conn:   conn,
		wsSend: make(chan models.WSMessage, 256),
		Ctx:    ctx,
		Cancel: cancel,
	}

	// Start writer goroutine
	go func() {
		ticker := time.NewTicker(pingPeriod)
		defer ticker.Stop()

		for {
			select {
			case msg := <-client.wsSend:
				conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := conn.WriteJSON(msg); err != nil {
					return
				}
			case <-ticker.C:
				conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Set up ping handler
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	// Process incoming search requests
	for {
		var msg struct {
			Type     string `json:"type"`
			Query    string `json:"query"`
			Exact    bool   `json:"exact"`
			Category string `json:"category"`
		}
		if err := conn.ReadJSON(&msg); err != nil {
			break
		}

		if msg.Type == "search" && msg.Query != "" {
			go PerformStreamingSearch(client, msg.Query, msg.Exact, msg.Category)
		}
	}
}

func WebsocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	// Create a context for this connection
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Ensure context is cancelled when handler exits (disconnection)

	client := &ClientConnection{
		conn:   conn,
		wsSend: make(chan models.WSMessage, 256), // Buffered channel to handle bursts
		Ctx:    ctx,
		Cancel: cancel,
	}

	// Register the client with the broadcaster
	GlobalBroadcaster.Register(client)

	// Start goroutine to send messages to the client
	go func() {
		ticker := time.NewTicker(pingPeriod)
		defer func() {
			ticker.Stop()
			GlobalBroadcaster.Unregister(client)
			conn.Close()
		}()

		for {
			select {
			case msg := <-client.wsSend:
				conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := conn.WriteJSON(msg); err != nil {
					log.Printf("WebSocket write error: %v", err)
					return
				}
				// Add a small delay to prevent overwhelming the client
				time.Sleep(10 * time.Millisecond)

			case <-ticker.C:
				conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					log.Printf("WebSocket ping error: %v", err)
					return
				}

			case <-ctx.Done():
				return
			}
		}
	}()

	// Keep the connection alive by handling ping/pong
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		var msg struct {
			Type string `json:"type"`
		}
		if err := conn.ReadJSON(&msg); err != nil {
			log.Printf("WebSocket read error: %v", err)
			// Context cancel (defer) will handle cleanup
			GlobalBroadcaster.Unregister(client)
			break
		}

		if msg.Type == "get_status" {
			// Send current status to this client
			// Use pooled client to avoid head-of-line blocking
			if status, err := GlobalBroadcaster.mpdClient.GetStatus(); err == nil {
				select {
				case client.wsSend <- models.WSMessage{Type: "status", Data: status}:
				case <-ctx.Done():
				default:
					// Client buffer full, skip
				}
			}
		}
	}
}

func LogWebsocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// Placeholder for log streaming
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if err := conn.WriteJSON(map[string]interface{}{
			"type": "logs",
			"data": []interface{}{},
		}); err != nil {
			break
		}
	}
}

// ensureIdleClientConnection ensures the idle client is connected
func (b *Broadcaster) ensureIdleClientConnection() {
	if !b.idleMpdClient.IsConnected() {
		if err := b.idleMpdClient.EnsureConnection(); err != nil {
			log.Printf("Failed to connect idle client: %v", err)
			b.idleClientConnected = false
		} else {
			b.idleClientConnected = true
		}
	}
}

// reconnectIdleClientWithBackoff attempts to reconnect the idle client with exponential backoff
func (b *Broadcaster) reconnectIdleClientWithBackoff() {
	// Exponential backoff: 3s, 10s, 30s, then 60s
	retryIntervals := []time.Duration{3 * time.Second, 10 * time.Second, 30 * time.Second, 60 * time.Second}

	for i, interval := range retryIntervals {
		select {
		case <-b.ctx.Done():
			return
		default:
			log.Printf("Attempting to reconnect idle client (attempt %d)", i+1)

			// Close the current connection if it exists
			b.idleMpdClient.ResetConnection()

			// Try to establish a new connection
			if err := b.idleMpdClient.EnsureConnection(); err != nil {
				log.Printf("Failed to reconnect idle client: %v, retrying in %v", err, interval)
				time.Sleep(interval)
			} else {
				log.Printf("Successfully reconnected idle client")
				b.idleClientConnected = true
				return // Successfully reconnected
			}
		}
	}

	// If all retries failed, keep trying every minute
	log.Printf("All reconnection attempts failed, continuing with periodic retries")
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-b.ctx.Done():
			return
		case <-ticker.C:
			log.Printf("Attempting periodic idle client reconnection")

			// Close the current connection if it exists
			b.idleMpdClient.ResetConnection()

			if err := b.idleMpdClient.EnsureConnection(); err != nil {
				log.Printf("Periodic reconnection failed: %v", err)
			} else {
				log.Printf("Successfully reconnected idle client after periodic attempt")
				b.idleClientConnected = true
				return
			}
		}
	}
}
