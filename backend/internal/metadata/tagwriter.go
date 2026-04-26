package metadata

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.senan.xyz/taglib"

	"sloppy-disk/internal/config"
	"sloppy-disk/internal/models"
	"sloppy-disk/internal/mpd"
)

// AudioExtensions contains all supported audio file extensions
var AudioExtensions = map[string]bool{
	".flac": true,
	".mp3":  true,
	".m4a":  true,
	".aac":  true,
	".ogg":  true,
	".opus": true,
	".wav":  true,
	".wma":  true,
	".mpc":  true,
	".ape":  true,
	".aiff": true,
	".aif":  true,
}

// TagWriter handles writing metadata tags to audio files
type TagWriter struct {
	client *http.Client
}

// TrackMatcher handles matching files to tracks using multiple strategies
type TrackMatcher struct {
	tracks []models.Song
}

// NewTrackMatcher creates a new track matcher
func NewTrackMatcher(tracks []models.Song) *TrackMatcher {
	return &TrackMatcher{tracks: tracks}
}

// MatchResult represents the result of matching a file to a track
type MatchResult struct {
	Title     string
	TrackNum  string
	Matched   bool
	Strategy  string
}

// MatchFile attempts to match a filename to a track using multiple strategies
func (tm *TrackMatcher) MatchFile(filename string) MatchResult {
	if len(tm.tracks) == 0 {
		return MatchResult{Matched: false, Strategy: "no_tracks"}
	}

	// Strategy 1: Track number prefix matching (e.g., "01", "1-", "1. ")
	if result := tm.matchByTrackNumber(filename); result.Matched {
		return result
	}

	// Strategy 2: Normalized title matching
	if result := tm.matchByTitle(filename); result.Matched {
		return result
	}

	// Strategy 3: Position-based fallback (if file count matches track count)
	// This would require knowing the file position in the sorted list
	// For now, we return unmatched

	return MatchResult{Matched: false, Strategy: "none"}
}

// matchByTrackNumber tries to match by track number prefix
func (tm *TrackMatcher) matchByTrackNumber(filename string) MatchResult {
	normFile := normalizeString(filename)

	for _, t := range tm.tracks {
		if t.Track == "" {
			continue
		}

		// Try various track number prefixes
		trackNum := t.Track
		prefixes := []string{
			trackNum + " ",
			trackNum + "-",
			trackNum + ".",
			strings.TrimLeft(trackNum, "0") + " ",  // "1 " matches "01 "
		}

		for _, prefix := range prefixes {
			if strings.HasPrefix(normFile, prefix) {
				return MatchResult{
					Title:    t.Title,
					TrackNum: t.Track,
					Matched:  true,
					Strategy: "track_number",
				}
			}
		}
	}

	return MatchResult{Matched: false, Strategy: "track_number_failed"}
}

// matchByTitle tries to match by normalized title
func (tm *TrackMatcher) matchByTitle(filename string) MatchResult {
	normFile := normalizeString(filename)

	for _, t := range tm.tracks {
		if t.Title == "" {
			continue
		}

		normTitle := normalizeString(t.Title)

		// Check for title containment with word boundary awareness
		if strings.Contains(normFile, normTitle) {
			return MatchResult{
				Title:    t.Title,
				TrackNum: t.Track,
				Matched:  true,
				Strategy: "title",
			}
		}
	}

	return MatchResult{Matched: false, Strategy: "title_failed"}
}

// NewTagWriter creates a new tag writer
func NewTagWriter() *TagWriter {
	return &TagWriter{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// ApplyMetadata applies metadata to all audio files in an album directory
func (w *TagWriter) ApplyMetadata(albumPath string, metadata models.MetadataCandidate) (*ApplyResult, error) {
	// Input validation
	if albumPath == "" {
		return nil, fmt.Errorf("albumPath cannot be empty")
	}
	if metadata.Artist == "" || metadata.Album == "" {
		return nil, fmt.Errorf("metadata must include both Artist and Album")
	}

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

	// Build common tags
	commonTags := w.buildTagMap(metadata)

	// Create track matcher for better file-to-track matching
	matcher := NewTrackMatcher(metadata.Tracks)

	// Process each file
	for _, file := range files {
		tags := make(map[string][]string)
		for k, v := range commonTags {
			tags[k] = v
		}

		// Match file to track using robust matcher
		filename := filepath.Base(file)
		matchResult := matcher.MatchFile(filename)

		// Handle title tag
		if matchResult.Matched && matchResult.Title != "" {
			tags[taglib.Title] = []string{matchResult.Title}
			log.Printf("Matched %s to track %q (strategy: %s)", filename, matchResult.Title, matchResult.Strategy)
		} else {
			// No match found - try to preserve existing title tag to prevent malformed files
			if existingTags, err := taglib.ReadTags(file); err == nil {
				if title, ok := existingTags[taglib.Title]; ok && len(title) > 0 {
					tags[taglib.Title] = title
					log.Printf("No match for %s, preserved existing title: %s", filename, title[0])
				}
			} else {
				log.Printf("WARNING: Could not match file %s to any track in metadata and failed to read existing tags", filename)
			}
		}

		// Handle track number
		if matchResult.Matched && matchResult.TrackNum != "" {
			tags[taglib.TrackNumber] = []string{matchResult.TrackNum}
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
		if _, err := mpd.GetClient().SendCommand(fmt.Sprintf("update %q", albumPath)); err != nil {
			log.Printf("MPD update failed: %v", err)
			result.Errors = append(result.Errors, "MPD update failed: "+err.Error())
		}
	}

	return result, nil
}

// ApplyCoverArt downloads and saves cover art to the album directory
func (w *TagWriter) ApplyCoverArt(albumPath string, imageURL string) (*CoverArtResult, error) {
	cfg := config.Get()

	// Download image
	resp, err := w.client.Get(imageURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download cover art: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download cover art: HTTP %d", resp.StatusCode)
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
		return nil, fmt.Errorf("failed to create cover art directory: %w", err)
	}

	destPath := filepath.Join(destDir, "Folder"+ext)
	f, err := os.Create(destPath)
	if err != nil {
		return nil, fmt.Errorf("failed to save cover art: %w", err)
	}
	defer f.Close()

	contentLength, err := io.Copy(f, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to save cover art: %w", err)
	}

	// Trigger MPD update
	if _, err := mpd.GetClient().SendCommand(fmt.Sprintf("update %q", albumPath)); err != nil {
		log.Printf("MPD update failed: %v", err)
	}

	return &CoverArtResult{
		SourceURL:     imageURL,
		DestPath:      destPath,
		Format:        ext,
		ContentLength: contentLength,
	}, nil
}

// buildTagMap creates a tag map from metadata
func (w *TagWriter) buildTagMap(metadata models.MetadataCandidate) map[string][]string {
	tags := make(map[string][]string)

	// Basic tags
	if metadata.Album != "" {
		tags[taglib.Album] = []string{metadata.Album}
	}
	if metadata.Artist != "" {
		tags[taglib.Artist] = []string{metadata.Artist}
	}

	// AlbumArtist - try metadata first, fall back to Artist
	// Important for multi-artist compilations
	if albumArtist, ok := metadata.Metadata["albumArtist"].(string); ok && albumArtist != "" {
		tags["ALBUMARTIST"] = []string{albumArtist}
	} else if metadata.Artist != "" {
		tags["ALBUMARTIST"] = []string{metadata.Artist}
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

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if AudioExtensions[ext] {
			files = append(files, filepath.Join(dir, entry.Name()))
		}
	}

	return files, nil
}

// ApplyResult represents the result of applying metadata
type ApplyResult struct {
	TotalFiles   int      `json:"totalFiles"`
	UpdatedFiles int      `json:"updatedFiles"`
	SkippedFiles int      `json:"skippedFiles"`
	Errors       []string `json:"errors"`
}

// CoverArtResult represents the result of applying cover art
type CoverArtResult struct {
	SourceURL      string `json:"sourceUrl"`
	DestPath       string `json:"destPath"`
	Format         string `json:"format"`
	ContentLength  int64  `json:"contentLength"`
}

// ApplyMetadataRequest represents a request to apply metadata
type ApplyMetadataRequest struct {
	AlbumPath   string                   `json:"albumPath"`
	Metadata    models.MetadataCandidate `json:"metadata"`
	CoverArtURL string                   `json:"coverArtUrl,omitempty"`
}
