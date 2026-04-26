package coverart

import (
	"context"
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

type candidateKey struct {
	artist string
	album  string
}

type Manager struct {
	client     *http.Client
	cacheMu    sync.Mutex
	candidates map[candidateKey][]models.CoverArtCandidate
	cacheOrder []candidateKey
}

func NewManager() *Manager {
	return &Manager{
		client:     &http.Client{Timeout: 60 * time.Second},
		candidates: make(map[candidateKey][]models.CoverArtCandidate),
	}
}

func (m *Manager) FetchCandidates(artist, album string) ([]models.CoverArtCandidate, error) {
	key := candidateKey{artist: strings.ToLower(artist), album: strings.ToLower(album)}

	m.cacheMu.Lock()
	if cands, ok := m.candidates[key]; ok {
		// Move to end for LRU
		m.removeFromOrder(key)
		m.cacheOrder = append(m.cacheOrder, key)
		m.cacheMu.Unlock()
		return cands, nil
	}
	m.cacheMu.Unlock()

	// Use aggregator to fetch candidates
	aggregator := metadata.NewAggregator()
	cands, err := aggregator.SearchCoverArt(context.Background(), artist, album)
	if err != nil {
		return nil, err
	}

	m.cacheMu.Lock()
	// Maintain max 20 entries
	if len(m.cacheOrder) >= 20 {
		oldest := m.cacheOrder[0]
		delete(m.candidates, oldest)
		m.cacheOrder = m.cacheOrder[1:]
	}
	m.candidates[key] = cands
	m.cacheOrder = append(m.cacheOrder, key)
	m.cacheMu.Unlock()

	return cands, nil
}

func (m *Manager) removeFromOrder(key candidateKey) {
	for i, k := range m.cacheOrder {
		if k == key {
			m.cacheOrder = append(m.cacheOrder[:i], m.cacheOrder[i+1:]...)
			return
		}
	}
}

func (m *Manager) ApplyCover(albumPath string, imageURL string) error {
	log.Printf("[CoverArt Manager] ApplyCover called: albumPath=%s, imageURL=%s", albumPath, imageURL)

	// Validate inputs
	if albumPath == "" {
		err := fmt.Errorf("albumPath cannot be empty")
		log.Printf("[CoverArt Manager] Validation error: %v", err)
		return err
	}
	if imageURL == "" {
		err := fmt.Errorf("imageURL cannot be empty")
		log.Printf("[CoverArt Manager] Validation error: %v", err)
		return err
	}

	cfg := config.Get()
	log.Printf("[CoverArt Manager] Config: MusicRoot=%s, CoverArtRoot=%s", cfg.MusicRoot, cfg.CoverArtRoot)

	// Validate configuration
	if cfg.MusicRoot == "" && cfg.CoverArtRoot == "" {
		err := fmt.Errorf("both MusicRoot and CoverArtRoot are not configured")
		log.Printf("[CoverArt Manager] Configuration error: %v", err)
		return err
	}

	// 1. Download image
	log.Printf("[CoverArt Manager] Downloading image from: %s", imageURL)
	resp, err := m.client.Get(imageURL)
	if err != nil {
		log.Printf("[CoverArt Manager] Failed to download image: %v", err)
		return fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("failed to download image: %s", resp.Status)
		log.Printf("[CoverArt Manager] Bad response status: %v", err)
		return err
	}
	log.Printf("[CoverArt Manager] Image downloaded successfully, Content-Type: %s", resp.Header.Get("Content-Type"))

	// 2. Determine destination path
	var destDir string
	if cfg.CoverArtRoot != "" {
		destDir = filepath.Join(cfg.CoverArtRoot, albumPath)
		log.Printf("[CoverArt Manager] Using CoverArtRoot: %s", destDir)
	} else {
		destDir = filepath.Join(cfg.MusicRoot, albumPath)
		log.Printf("[CoverArt Manager] Using MusicRoot: %s", destDir)
	}

	log.Printf("[CoverArt Manager] Creating directory: %s", destDir)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		log.Printf("[CoverArt Manager] Failed to create directory: %v", err)
		return fmt.Errorf("failed to create directory %s: %w", destDir, err)
	}

	// 3. Detect extension from Content-Type
	ext := ".jpg"
	contentType := resp.Header.Get("Content-Type")
	switch contentType {
	case "image/png":
		ext = ".png"
	case "image/webp":
		ext = ".webp"
	case "image/gif":
		ext = ".gif"
	}

	// Use capital "Folder" as the standard filename
	destPath := filepath.Join(destDir, "Folder"+ext)
	log.Printf("[CoverArt Manager] Destination path: %s", destPath)

	// Remove any existing folder/Folder cover art files (case-insensitive cleanup)
	entries, err := os.ReadDir(destDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			nameLower := strings.ToLower(entry.Name())
			// Check if this is a folder.jpg/Folder.jpg variant with any supported extension
			if strings.HasPrefix(nameLower, "folder.") {
				oldExt := strings.ToLower(filepath.Ext(entry.Name()))
				if oldExt == ".jpg" || oldExt == ".jpeg" || oldExt == ".png" || oldExt == ".webp" || oldExt == ".gif" {
					oldPath := filepath.Join(destDir, entry.Name())
					log.Printf("[CoverArt Manager] Removing old cover art: %s", oldPath)
					if err := os.Remove(oldPath); err != nil {
						log.Printf("[CoverArt Manager] Warning: failed to remove %s: %v", oldPath, err)
					}
				}
			}
		}
	}

	f, err := os.Create(destPath)
	if err != nil {
		log.Printf("[CoverArt Manager] Failed to create file: %v", err)
		return fmt.Errorf("failed to create file %s: %w", destPath, err)
	}
	defer f.Close()

	written, err := io.Copy(f, resp.Body)
	if err != nil {
		log.Printf("[CoverArt Manager] Failed to write file: %v", err)
		return fmt.Errorf("failed to write file: %w", err)
	}
	log.Printf("[CoverArt Manager] File written successfully: %d bytes", written)

	log.Printf("[CoverArt Manager] Cover art applied successfully")

	return nil
}

func (m *Manager) SaveUploadedCover(albumPath string, reader io.Reader, contentType string) error {
	log.Printf("[CoverArt Manager] SaveUploadedCover called: albumPath=%s, contentType=%s", albumPath, contentType)

	// Validate inputs
	if albumPath == "" {
		err := fmt.Errorf("albumPath cannot be empty")
		log.Printf("[CoverArt Manager] Validation error: %v", err)
		return err
	}
	if reader == nil {
		err := fmt.Errorf("reader cannot be nil")
		log.Printf("[CoverArt Manager] Validation error: %v", err)
		return err
	}

	cfg := config.Get()
	log.Printf("[CoverArt Manager] Config: MusicRoot=%s, CoverArtRoot=%s", cfg.MusicRoot, cfg.CoverArtRoot)

	// Validate configuration
	if cfg.MusicRoot == "" && cfg.CoverArtRoot == "" {
		err := fmt.Errorf("both MusicRoot and CoverArtRoot are not configured")
		log.Printf("[CoverArt Manager] Configuration error: %v", err)
		return err
	}

	// 1. Determine destination path
	var destDir string
	if cfg.CoverArtRoot != "" {
		destDir = filepath.Join(cfg.CoverArtRoot, albumPath)
		log.Printf("[CoverArt Manager] Using CoverArtRoot: %s", destDir)
	} else {
		destDir = filepath.Join(cfg.MusicRoot, albumPath)
		log.Printf("[CoverArt Manager] Using MusicRoot: %s", destDir)
	}

	log.Printf("[CoverArt Manager] Creating directory: %s", destDir)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		log.Printf("[CoverArt Manager] Failed to create directory: %v", err)
		return fmt.Errorf("failed to create directory %s: %w", destDir, err)
	}

	// 2. Detect extension from Content-Type
	ext := ".jpg"
	switch contentType {
	case "image/png":
		ext = ".png"
	case "image/webp":
		ext = ".webp"
	case "image/gif":
		ext = ".gif"
	}

	// Use capital "Folder" as the standard filename
	destPath := filepath.Join(destDir, "Folder"+ext)
	log.Printf("[CoverArt Manager] Destination path: %s", destPath)

	// Remove any existing folder/Folder cover art files (case-insensitive cleanup)
	entries, err := os.ReadDir(destDir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			nameLower := strings.ToLower(entry.Name())
			// Check if this is a folder.jpg/Folder.jpg variant with any supported extension
			if strings.HasPrefix(nameLower, "folder.") {
				oldExt := strings.ToLower(filepath.Ext(entry.Name()))
				if oldExt == ".jpg" || oldExt == ".jpeg" || oldExt == ".png" || oldExt == ".webp" || oldExt == ".gif" {
					oldPath := filepath.Join(destDir, entry.Name())
					log.Printf("[CoverArt Manager] Removing old cover art: %s", oldPath)
					if err := os.Remove(oldPath); err != nil {
						log.Printf("[CoverArt Manager] Warning: failed to remove %s: %v", oldPath, err)
					}
				}
			}
		}
	}

	f, err := os.Create(destPath)
	if err != nil {
		log.Printf("[CoverArt Manager] Failed to create file: %v", err)
		return fmt.Errorf("failed to create file %s: %w", destPath, err)
	}
	defer f.Close()

	written, err := io.Copy(f, reader)
	if err != nil {
		log.Printf("[CoverArt Manager] Failed to write file: %v", err)
		return fmt.Errorf("failed to write file: %w", err)
	}
	log.Printf("[CoverArt Manager] File written successfully: %d bytes", written)

	log.Printf("[CoverArt Manager] Uploaded cover art saved successfully")

	return nil
}

// FindImage searches for a cover art image for the given album path.
// It checks CoverArtRoot and MusicRoot for Folder.ext, cover.ext, and fallbacks.
func (m *Manager) FindImage(albumPath string) (string, error) {
	cfg := config.Get()

	// 1. Check in CoverArtRoot
	coverPath := filepath.Join(cfg.CoverArtRoot, albumPath)
	if path, err := m.searchInDir(coverPath); err == nil && path != "" {
		return path, nil
	}

	// 2. Check in MusicRoot
	musicPath := filepath.Join(cfg.MusicRoot, albumPath)
	if path, err := m.searchInDir(musicPath); err == nil && path != "" {
		return path, nil
	}

	return "", os.ErrNotExist
}

func (m *Manager) searchInDir(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	extensions := []string{".jpg", ".jpeg", ".png", ".webp", ".gif"}

	// Priority 1: Folder.ext (case insensitive)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.ToLower(entry.Name())
		for _, ext := range extensions {
			if name == "folder"+ext {
				return filepath.Join(dir, entry.Name()), nil
			}
		}
	}

	// Priority 2: cover.ext (case insensitive)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.ToLower(entry.Name())
		for _, ext := range extensions {
			if name == "cover"+ext {
				return filepath.Join(dir, entry.Name()), nil
			}
		}
	}

	// Priority 3: First image file
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		for _, supportedExt := range extensions {
			if ext == supportedExt {
				return filepath.Join(dir, entry.Name()), nil
			}
		}
	}

	return "", nil
}
