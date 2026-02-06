package mpd

import (
	"mpd-client-modern/internal/models"
	"strings"
	"testing"
)

func TestParseGetAllAlbumKeys(t *testing.T) {
	// Simulated MPD response for: list album group albumartist group date group genre
	resp := `AlbumArtist: !!!
Date: 2004
Genre: Rock
Album: Louden Up Now
Date: 2010
Album: Strange Weather , Isn't It
AlbumArtist: Air
Date: 2001
Genre: Electronic
Album: 10,000 Hz Legend
`
	lines := strings.Split(strings.TrimSpace(resp), "\n")
	var keys []models.AlbumKey

	var currentAlbumArtist, currentDate, currentGenre string

	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) != 2 {
			continue
		}

		key, value := parts[0], parts[1]

		switch key {
		case "AlbumArtist", "Artist":
			currentAlbumArtist = value
			currentDate = ""
			currentGenre = ""
		case "Date":
			currentDate = value
			currentGenre = ""
		case "Genre":
			currentGenre = value
		case "Album":
			if value != "" {
				keys = append(keys, models.AlbumKey{
					Album:       value,
					AlbumArtist: currentAlbumArtist,
					Date:        currentDate,
					Genre:       currentGenre,
				})
			}
		}
	}

	expected := []models.AlbumKey{
		{Album: "Louden Up Now", AlbumArtist: "!!!", Date: "2004", Genre: "Rock"},
		{Album: "Strange Weather , Isn't It", AlbumArtist: "!!!", Date: "2010", Genre: ""}, // Genre should be reset because Date changed
		{Album: "10,000 Hz Legend", AlbumArtist: "Air", Date: "2001", Genre: "Electronic"},
	}

	if len(keys) != len(expected) {
		t.Fatalf("Expected %d keys, got %d", len(expected), len(keys))
	}

	for i, k := range keys {
		if k.Album != expected[i].Album || k.AlbumArtist != expected[i].AlbumArtist || k.Date != expected[i].Date || k.Genre != expected[i].Genre {
			t.Errorf("Mismatch at index %d:\nExpected: %+v\nGot:      %+v", i, expected[i], k)
		}
	}
}
