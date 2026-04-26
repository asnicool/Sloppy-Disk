package artistimage

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"sloppy-disk/internal/config"
	"sloppy-disk/internal/metadata"
	"sloppy-disk/internal/models"
)

type Manager struct {
	client *http.Client
	mu     sync.Mutex
	cache  map[string]string // artist name -> cached image path
}

var (
	instance *Manager
	once     sync.Once
)

func NewManager() *Manager {
	once.Do(func() {
		instance = &Manager{
			client: &http.Client{Timeout: 60 * time.Second},
			cache:  make(map[string]string),
		}
	})
	return instance
}

// GetOrFetchArtistImage returns the local path for an artist's image.
// If not cached, it auto-fetches from web sources and saves to disk.
func (m *Manager) GetOrFetchArtistImage(artistName string) (string, error) {
	if artistName == "" {
		return "", fmt.Errorf("artist name cannot be empty")
	}

	// Check cache
	m.mu.Lock()
	if path, ok := m.cache[artistName]; ok {
		// Verify file still exists
		if _, err := os.Stat(path); err == nil {
			m.mu.Unlock()
			return path, nil
		}
		// File missing, remove from cache
		delete(m.cache, artistName)
	}
	m.mu.Unlock()

	// Fetch from web sources
	imagePath, err := m.fetchAndCache(artistName)
	if err != nil {
		return "", err
	}

	// Update cache
	m.mu.Lock()
	m.cache[artistName] = imagePath
	m.mu.Unlock()

	return imagePath, nil
}

// GetArtistImageURL returns the URL path for accessing artist image via API
func (m *Manager) GetArtistImageURL(artistName string) string {
	// URL encoded artist name for API access
	encoded := strings.ReplaceAll(artistName, "/", "%2F")
	return fmt.Sprintf("/api/artistart/%s/Artist.jpg", encoded)
}

// HasArtistImage checks if an artist image exists locally
func (m *Manager) HasArtistImage(artistName string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if path, ok := m.cache[artistName]; ok {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}

	// Check filesystem
	path := m.getArtistImagePath(artistName)
	if _, err := os.Stat(path); err == nil {
		// Update cache
		m.cache[artistName] = path
		return true
	}

	return false
}

// fetchAndCache fetches artist image from web sources and saves to disk
func (m *Manager) fetchAndCache(artistName string) (string, error) {
	log.Printf("[ArtistImage] Fetching image for artist: %s", artistName)

	// Get candidates from Discogs (primary source for artist images)
	discogs := metadata.NewDiscogsProvider()

	candidates, err := discogs.GetArtistImage(artistName)
	if err != nil {
		log.Printf("[ArtistImage] Error fetching from Discogs: %v", err)
		candidates = []models.ArtistImageCandidate{}
	}

	// Also try MusicBrainz
	mbProvider := metadata.NewMusicBrainzProvider()
	mbCandidates, err := mbProvider.GetArtistImage(artistName)
	if err != nil {
		log.Printf("[ArtistImage] Error fetching from MusicBrainz: %v", err)
	}
	candidates = append(candidates, mbCandidates...)

	if len(candidates) == 0 {
		return "", fmt.Errorf("no artist images found for: %s", artistName)
	}

	// Use the first candidate (highest priority)
	firstCandidate := candidates[0]
	log.Printf("[ArtistImage] Using image from %s: %s", firstCandidate.Source, firstCandidate.URL)

	// Download image
	resp, err := m.client.Get(firstCandidate.URL)
	if err != nil {
		return "", fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download image: %s", resp.Status)
	}

	// Determine destination path
	destPath, err := m.saveImage(artistName, resp.Header.Get("Content-Type"), resp.Body)
	if err != nil {
		return "", err
	}

	log.Printf("[ArtistImage] Saved image to: %s", destPath)
	return destPath, nil
}

// saveImage saves the downloaded image to disk
func (m *Manager) saveImage(artistName, contentType string, body io.Reader) (string, error) {
	cfg := config.Get()

	// Determine root directory
	var rootDir string
	if cfg.CoverArtRoot != "" {
		rootDir = cfg.CoverArtRoot
	} else if cfg.MusicRoot != "" {
		rootDir = cfg.MusicRoot
	} else {
		return "", fmt.Errorf("neither CoverArtRoot nor MusicRoot is configured")
	}

	// Create artist directory
	artistDir := filepath.Join(rootDir, artistName)
	if err := os.MkdirAll(artistDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create artist directory: %w", err)
	}

	// Determine extension
	ext := ".jpg"
	switch contentType {
	case "image/png":
		ext = ".png"
	case "image/webp":
		ext = ".webp"
	case "image/gif":
		ext = ".gif"
	}

	destPath := filepath.Join(artistDir, "Artist"+ext)

	// Clean up old variants
	entries, err := os.ReadDir(artistDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			nameLower := strings.ToLower(entry.Name())
			if strings.HasPrefix(nameLower, "artist.") {
				oldPath := filepath.Join(artistDir, entry.Name())
				if oldPath != destPath {
					os.Remove(oldPath)
				}
			}
		}
	}

	// Write file
	f, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	written, err := io.Copy(f, body)
	if err != nil {
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	log.Printf("[ArtistImage] Written %d bytes to %s", written, destPath)
	return destPath, nil
}

// getArtistImagePath returns the filesystem path for an artist's image
func (m *Manager) getArtistImagePath(artistName string) string {
	cfg := config.Get()

	var rootDir string
	if cfg.CoverArtRoot != "" {
		rootDir = cfg.CoverArtRoot
	} else if cfg.MusicRoot != "" {
		rootDir = cfg.MusicRoot
	} else {
		return ""
	}

	artistDir := filepath.Join(rootDir, artistName)

	// Check for Artist.jpg, Artist.png, etc.
	extensions := []string{".jpg", ".jpeg", ".png", ".webp", ".gif"}
	for _, ext := range extensions {
		path := filepath.Join(artistDir, "Artist"+ext)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// FetchCandidates returns available artist image candidates for management UI
func (m *Manager) FetchCandidates(artistName string) ([]models.ArtistImageCandidate, error) {
	log.Printf("[ArtistImage Manager] FetchCandidates for: %s", artistName)

	if artistName == "" {
		return nil, fmt.Errorf("artist name cannot be empty")
	}

	discogs := metadata.NewDiscogsProvider()
	candidates, err := discogs.GetArtistImage(artistName)
	if err != nil {
		log.Printf("[ArtistImage Manager] Discogs error: %v", err)
		// Don't return error, just return empty
		candidates = []models.ArtistImageCandidate{}
	}
	log.Printf("[ArtistImage Manager] Discogs returned %d candidates", len(candidates))

	mbProvider := metadata.NewMusicBrainzProvider()
	mbCandidates, _ := mbProvider.GetArtistImage(artistName)
	log.Printf("[ArtistImage Manager] MusicBrainz returned %d candidates", len(mbCandidates))
	candidates = append(candidates, mbCandidates...)

	return candidates, nil
}

// ApplyArtistImage saves a selected artist image URL to disk
func (m *Manager) ApplyArtistImage(artistName, imageURL string) error {
	if artistName == "" || imageURL == "" {
		return fmt.Errorf("artist name and image URL are required")
	}

	log.Printf("[ArtistImage] Applying image for %s: %s", artistName, imageURL)

	resp, err := m.client.Get(imageURL)
	if err != nil {
		return fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download image: %s", resp.Status)
	}

	path, err := m.saveImage(artistName, resp.Header.Get("Content-Type"), resp.Body)
	if err != nil {
		return err
	}

	// Update cache
	m.mu.Lock()
	m.cache[artistName] = path
	m.mu.Unlock()

	log.Printf("[ArtistImage] Applied image to: %s", path)
	return nil
}
