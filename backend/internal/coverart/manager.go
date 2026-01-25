package coverart

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"mpd-client-modern/internal/config"
	"mpd-client-modern/internal/models"
)

type Manager struct {
	client *http.Client
}

func NewManager() *Manager {
	return &Manager{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (m *Manager) FetchCandidates(artist, album string) ([]models.CoverArtCandidate, error) {
	// This would call multiple providers (Discogs, MusicBrainz, etc.)
	// For now, returning a placeholder
	return []models.CoverArtCandidate{
		{Source: "Placeholder", URL: "https://via.placeholder.com/500", Size: "500x500"},
	}, nil
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

	// 2. Determine destination path (mirroring music structure on SSD)
	destDir := filepath.Join(cfg.CoverArtRoot, albumPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	// 3. Save as Folder.jpg (or appropriate extension)
	ext := ".jpg" // Should detect from Content-Type
	destPath := filepath.Join(destDir, "Folder"+ext)

	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
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
