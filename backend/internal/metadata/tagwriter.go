package metadata

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"go.senan.xyz/taglib"

	"mpd-client-modern/internal/config"
	"mpd-client-modern/internal/models"
	"mpd-client-modern/internal/mpd"
)

// TagWriter handles writing metadata tags to audio files
type TagWriter struct {
	client *http.Client
}

// NewTagWriter creates a new tag writer
func NewTagWriter() *TagWriter {
	return &TagWriter{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// ApplyMetadata applies metadata to all audio files in an album directory
func (w *TagWriter) ApplyMetadata(albumPath string, metadata models.MetadataCandidate) (*ApplyResult, error) {
	cfg := config.Get()
	fullPath := filepath.Join(cfg.MusicRoot, albumPath)

	// Find all audio files in the directory
	files, err := w.findAudioFiles(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to find audio files: %w", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no audio files found in %s", albumPath)
	}

	result := &ApplyResult{
		TotalFiles:   len(files),
		UpdatedFiles: 0,
		Errors:       make([]string, 0),
	}

	// Build tag map
	tags := w.buildTagMap(metadata)

	// Process each file
	for _, file := range files {
		// Determine track number from filename if not in metadata
		trackNum := w.extractTrackNumber(file, metadata.Tracks)
		if trackNum != "" {
			tags[taglib.TrackNumber] = []string{trackNum}
		}

		// Write tags
		err := taglib.WriteTags(file, tags, taglib.Clear)
		if err != nil {
			result.Errors = append(result.Errors, filepath.Base(file)+": "+err.Error())
			continue
		}

		result.UpdatedFiles++
	}

	// Trigger MPD update
	if result.UpdatedFiles > 0 {
		if _, err := mpd.GetClient().SendCommand("update " + albumPath); err != nil {
			// Log but don't fail
			log.Printf("MPD update failed: %v", err)
			result.Errors = append(result.Errors, "MPD update failed: "+err.Error())
		}
	}

	return result, nil
}

// ApplyCoverArt downloads and saves cover art to the album directory
func (w *TagWriter) ApplyCoverArt(albumPath string, imageURL string) error {
	cfg := config.Get()

	// Download image
	resp, err := w.client.Get(imageURL)
	if err != nil {
		return fmt.Errorf("failed to download cover art: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download cover art: HTTP %d", resp.StatusCode)
	}

	// Determine file extension from content type
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

	// Save to CoverArtRoot directory
	destDir := filepath.Join(cfg.CoverArtRoot, albumPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create cover art directory: %w", err)
	}

	destPath := filepath.Join(destDir, "Folder"+ext)
	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to save cover art: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return fmt.Errorf("failed to save cover art: %w", err)
	}

	// Trigger MPD update
	if _, err := mpd.GetClient().SendCommand("update " + albumPath); err != nil {
		log.Printf("MPD update failed: %v", err)
	}

	return nil
}

// buildTagMap creates a tag map from metadata
func (w *TagWriter) buildTagMap(metadata models.MetadataCandidate) map[string][]string {
	tags := make(map[string][]string)

	// Basic tags - MetadataCandidate uses Album and Artist, not Title and AlbumArtist
	if metadata.Album != "" {
		tags[taglib.Album] = []string{metadata.Album}
	}
	if metadata.Artist != "" {
		tags[taglib.Artist] = []string{metadata.Artist}
	}
	if metadata.Year != "" {
		// Ensure year is in YYYY format
		if len(metadata.Year) > 4 {
			tags[taglib.Date] = []string{metadata.Year[:4]}
		} else {
			tags[taglib.Date] = []string{metadata.Year}
		}
	}
	if metadata.Genre != "" {
		tags[taglib.Genre] = []string{metadata.Genre}
	}

	// Extract MusicBrainz IDs from metadata
	if mbID, ok := metadata.Metadata["musicbrainzAlbumID"].(string); ok && mbID != "" {
		tags[taglib.MusicBrainzAlbumID] = []string{mbID}
	}
	if mbID, ok := metadata.Metadata["musicbrainzArtistID"].(string); ok && mbID != "" {
		tags[taglib.MusicBrainzArtistID] = []string{mbID}
	}

	return tags
}

// findAudioFiles finds all audio files in a directory
func (w *TagWriter) findAudioFiles(dir string) ([]string, error) {
	var files []string

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	audioExtensions := map[string]bool{
		".flac": true, ".mp3": true, ".m4a": true, ".aac": true,
		".ogg": true, ".opus": true, ".wav": true, ".wma": true,
		".mpc": true, ".ape": true, ".aiff": true, ".aif": true,
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if audioExtensions[ext] {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}

	return files, nil
}

// extractTrackNumber determines track number from filename or metadata
func (w *TagWriter) extractTrackNumber(file string, tracks []models.Song) string {
	filename := filepath.Base(file)
	ext := filepath.Ext(filename)
	baseName := strings.TrimSuffix(filename, ext)

	// Try to parse track number from filename (e.g., "01 - Song Title.flac")
	parts := strings.SplitN(baseName, " - ", 2)
	if len(parts) > 0 {
		trackNum := strings.TrimPrefix(parts[0], "0")
		if _, err := strconv.Atoi(trackNum); err == nil {
			return parts[0]
		}
	}

	return ""
}

// ApplyResult represents the result of applying metadata
type ApplyResult struct {
	TotalFiles   int      `json:"totalFiles"`
	UpdatedFiles int      `json:"updatedFiles"`
	SkippedFiles int      `json:"skippedFiles"`
	Errors       []string `json:"errors"`
}

// ApplyMetadataRequest represents a request to apply metadata
type ApplyMetadataRequest struct {
	AlbumPath   string                 `json:"albumPath"`
	Metadata    models.MetadataCandidate `json:"metadata"`
	CoverArtURL string                 `json:"coverArtUrl,omitempty"`
}
