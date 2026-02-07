package coverart

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"mpd-client-modern/internal/config"
	"mpd-client-modern/internal/metadata"
	"mpd-client-modern/internal/models"
	"mpd-client-modern/internal/mpd"
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
		client:     &http.Client{Timeout: 30 * time.Second},
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
	cfg := config.Get()

	// 1. Download image
	resp, err := m.client.Get(imageURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download image: %s", resp.Status)
	}

	// 2. Determine destination path
	var destDir string
	if cfg.CoverArtRoot != "" {
		destDir = filepath.Join(cfg.CoverArtRoot, albumPath)
	} else {
		destDir = filepath.Join(cfg.MusicRoot, albumPath)
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
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

	destPath := filepath.Join(destDir, "folder"+ext)

	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return err
	}

	// 4. Trigger MPD update for this path
	client := mpd.GetClient()
	_, err = client.SendCommand(fmt.Sprintf("update %q", albumPath))
	return err
}

func (m *Manager) SaveUploadedCover(albumPath string, reader io.Reader, contentType string) error {
	cfg := config.Get()

	// 1. Determine destination path
	var destDir string
	if cfg.CoverArtRoot != "" {
		destDir = filepath.Join(cfg.CoverArtRoot, albumPath)
	} else {
		destDir = filepath.Join(cfg.MusicRoot, albumPath)
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
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

	destPath := filepath.Join(destDir, "folder"+ext)

	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, reader)
	if err != nil {
		return err
	}

	// 3. Trigger MPD update for this path
	client := mpd.GetClient()
	_, err = client.SendCommand(fmt.Sprintf("update %q", albumPath))
	return err
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
