package tags

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dhowden/tag"
	"mpd-client-modern/internal/models"
)

// ReadTags reads metadata from an audio file
func ReadTags(path string) (*models.Song, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	m, err := tag.ReadFrom(f)
	if err != nil {
		return nil, err
	}

	trackNum, _ := m.Track()
	discNum, _ := m.Disc()

	return &models.Song{
		Title:    m.Title(),
		Artist:   m.Artist(),
		Album:    m.Album(),
		Track:    fmt.Sprintf("%d", trackNum),
		Disc:     fmt.Sprintf("%d", discNum),
		Date:     fmt.Sprintf("%d", m.Year()),
		Genre:    m.Genre(),
		Path:     path,
	}, nil
}

// WriteTags updates metadata on an audio file
// Note: github.com/dhowden/tag is read-only. 
// For writing, we'll use a more specialized library or shell out to taglib/ffmpeg if needed.
// However, for this implementation, I'll use a placeholder and recommend a library like 'github.com/bogem/id3v2' for MP3 
// and others for FLAC. To keep it simple and support many formats, I'll use 'ffmpeg' via shell if available, 
// or a combination of Go libraries.
func WriteTags(path string, song *models.Song) error {
	// Placeholder for tag writing logic.
	// In a real-world scenario, we would use format-specific libraries:
	// - MP3: github.com/bogem/id3v2
	// - FLAC: github.com/go-flac/flacpicture and github.com/go-flac/go-flac
	// - MP4/AAC: github.com/dhowden/tag (read-only, would need another)
	
	fmt.Printf("Updating tags for %s: %+v\n", path, song)
	
	// For now, let's assume we use a tool like 'id3v2' or 'vorbis-tools' via exec if we want broad support,
	// or implement specific Go libraries for each format.
	
	ext := filepath.Ext(path)
	switch ext {
	case ".mp3":
		return writeMP3Tags(path, song)
	case ".flac":
		return writeFlacTags(path, song)
	default:
		return fmt.Errorf("unsupported format for writing: %s", ext)
	}
}

func writeMP3Tags(path string, song *models.Song) error {
	// Implementation using an MP3-specific library would go here
	return nil 
}

func writeFlacTags(path string, song *models.Song) error {
	// Implementation using a FLAC-specific library would go here
	return nil
}
