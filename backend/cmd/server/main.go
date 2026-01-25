package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"mpd-client-modern/internal/albumcache"
	"mpd-client-modern/internal/api"
	"mpd-client-modern/internal/config"
	"mpd-client-modern/internal/models"
	"mpd-client-modern/internal/mpd"

	"github.com/gorilla/mux"
)

func main() {
	// 1. Load Configuration
	if err := config.Load(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Set up the MPD connection status callback to broadcast to WebSocket clients
	mpd.SetConnectionStatusCallback(func(status *models.MPDStatus) {
		if api.GlobalBroadcaster != nil {
			api.GlobalBroadcaster.Broadcast(status)
		}
	})

	// Initialize Album Cache
	go func() {
		log.Println("Initializing album cache...")
		if err := albumcache.GetCache().Refresh(); err != nil {
			log.Printf("Failed to refresh album cache: %v", err)
		}
	}()

	// Start MPD Idle Listener for Database changes
	go func() {
		// Wait for config load and initial connection
		time.Sleep(2 * time.Second)

		idleClient := mpd.NewIdleClient()
		for {
			if err := idleClient.EnsureConnection(); err != nil {
				log.Printf("Idle client connection failed: %v, retrying in 5s...", err)
				time.Sleep(5 * time.Second)
				continue
			}

			// Wait for database changes
			changed, err := idleClient.Idle("database")
			if err != nil {
				log.Printf("Idle error: %v, retrying...", err)
				time.Sleep(2 * time.Second) // prevent tight loop on error
				continue
			}

			// Check if database changed
			for _, subsystem := range changed {
				if subsystem == "database" {
					log.Println("MPD database changed, refreshing cache...")
					if err := albumcache.GetCache().Refresh(); err != nil {
						log.Printf("Cache refresh failed: %v", err)
					}
				}
			}
		}
	}()

	// 2. Setup Router
	r := mux.NewRouter()
	r.UseEncodedPath()

	// CORS middleware
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	// 3. Register Routes
	api.RegisterRoutes(r)

	// Static file serving for frontend
	// Use relative paths to stay portable after project moves
	baseFrontendDir := "../frontend"
	if _, err := os.Stat(baseFrontendDir); os.IsNotExist(err) {
		// Fallback for when running from project root
		baseFrontendDir = "frontend"
	}

	// Try to use Vue production build (dist) if available
	frontendDir := filepath.Join(baseFrontendDir, "dist")
	if _, err := os.Stat(frontendDir); os.IsNotExist(err) {
		frontendDir = baseFrontendDir
	}
	log.Printf("Serving frontend from: %s", frontendDir)

	// Determine the main entry point (prefer dist/index.html, fallback to simple.html)
	indexFile := filepath.Join(frontendDir, "index.html")
	if _, err := os.Stat(indexFile); os.IsNotExist(err) {
		indexFile = filepath.Join(baseFrontendDir, "simple.html")
	}

	// SPA routing: Serve index.html for all non-API routes that don't match a file
	// Note: API routes are already registered by api.RegisterRoutes(r)

	// For all other routes, serve index.html (SPA routing)
	r.PathPrefix("/").Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if the request is for a file that exists
		filePath := filepath.Join(frontendDir, r.URL.Path)
		if _, err := os.Stat(filePath); err == nil {
			// File exists and is not a directory, serve it
			if !strings.HasSuffix(r.URL.Path, "/") {
				http.ServeFile(w, r, filePath)
				return
			}
		}
		// Otherwise serve index.html for SPA routing
		serveFrontendFile(indexFile)(w, r)
	}))

	// 4. Start Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "7070"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		fmt.Printf("Backend starting on port %s\n", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// 5. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}
}

func serveFrontendFile(path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Try to find the file relative to the workspace root
		absPath, err := filepath.Abs(path)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		file, err := os.Open(absPath)
		if err != nil {
			// Fallback to simple.html if index.html is missing
			if path == "frontend/index.html" {
				serveFrontendFile("frontend/simple.html")(w, r)
				return
			}
			http.Error(w, fmt.Sprintf("Frontend file not found: %v", err), http.StatusNotFound)
			return
		}
		defer file.Close()

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.Copy(w, file)
	}
}
