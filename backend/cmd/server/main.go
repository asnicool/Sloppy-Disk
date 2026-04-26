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

	"sloppy-disk/internal/albumcache"
	"sloppy-disk/internal/api"
	"sloppy-disk/internal/config"
	"sloppy-disk/internal/models"
	"sloppy-disk/internal/mpd"

	"github.com/gorilla/mux"
)

func main() {
	// 1. Load Configuration
	configPath := "config.json"
	if err := config.Load(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Set up the MPD connection status callback to broadcast to WebSocket clients
	mpd.SetConnectionStatusCallback(func(status *models.MPDStatus) {
		if api.GlobalBroadcaster != nil {
			api.GlobalBroadcaster.Broadcast(status)
		}
	})

	// Set up the database change callback to trigger cache refresh
	api.SetDatabaseChangeCallback(func() {
		log.Println("Database change detected, refreshing album cache...")
		if err := albumcache.GetCache().Refresh(); err != nil {
			log.Printf("Failed to refresh album cache on database change: %v", err)
		}
	})

	// Initialize Album Cache
	go func() {
		log.Println("Initializing album cache...")
		if err := albumcache.GetCache().Refresh(); err != nil {
			log.Printf("Failed to refresh album cache: %v", err)
		}
	}()

	// Note: Database change detection is now handled by the WebSocket broadcaster's
	// idle listener, which subscribes to all subsystems including "database".
	// The broadcaster will trigger cache refreshes when database changes occur.

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
	api.RegisterN50Routes(r)

	// Static file serving for music folder (for cover art)
	// This serves files from the musicRoot directory at /folder path
	cfg := config.Get()

	// Determine root directories to serve
	var roots []string
	if cfg.MusicRoot != "" {
		musicRoot := cfg.MusicRoot
		// Ensure musicRoot is an absolute path
		if !filepath.IsAbs(musicRoot) {
			// Try to make it absolute relative to the config file location
			if absConfigPath, err := filepath.Abs(configPath); err == nil {
				musicRoot = filepath.Join(filepath.Dir(absConfigPath), musicRoot)
			}
		}
		roots = append(roots, musicRoot)
		log.Printf("Serving music files from: %s", musicRoot)
	}

	// Also serve CoverArtRoot if configured (for artist images and album covers)
	if cfg.CoverArtRoot != "" {
		coverArtRoot := cfg.CoverArtRoot
		if !filepath.IsAbs(coverArtRoot) {
			if absConfigPath, err := filepath.Abs(configPath); err == nil {
				coverArtRoot = filepath.Join(filepath.Dir(absConfigPath), coverArtRoot)
			}
		}
		// Avoid duplicate if CoverArtRoot is same as MusicRoot
		isDuplicate := false
		for _, r := range roots {
			if r == coverArtRoot {
				isDuplicate = true
				break
			}
		}
		if !isDuplicate {
			roots = append(roots, coverArtRoot)
			log.Printf("Serving cover art root from: %s", coverArtRoot)
		}
	}

	// Serve multiple roots using a combined handler
	if len(roots) > 0 {
		r.PathPrefix("/folder/").Handler(http.StripPrefix("/folder/", http.FileServer(http.Dir(roots[0]))))
	}

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
