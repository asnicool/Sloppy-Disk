package api

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"mpd-client-modern/internal/albumcache"
	"mpd-client-modern/internal/models"
	"mpd-client-modern/internal/mpd"
)

// PerformStreamingSearch executes multiple searches in parallel and sends results via WS
func PerformStreamingSearch(client *ClientConnection, query string, isExact bool, category string) {
	if len(query) < 3 {
		return
	}

	cmdType := "search"
	if isExact {
		cmdType = "find"
	}

	var wg sync.WaitGroup

	// Categorize searches. If category is specified, only that goroutine will run.
	runAlbums := category == "" || category == "albums" || category == "album"
	runArtists := category == "" || category == "artists" || category == "artist"
	runGenres := category == "" || category == "genres" || category == "genre"
	runDates := category == "" || category == "dates" || category == "date"
	runSongs := category == "" || category == "songs" || category == "song" || category == "title"

	if runAlbums {
		wg.Add(1)
		// 1. Search Albums (using existing fuzzy cache)
		go func() {
			defer wg.Done()
			cache := albumcache.GetCache()
			var albums []models.Album
			if isExact {
				// Direct MPD search for exact matches to ensure accuracy
				client := mpd.GetClient()

				// Search by artist
				artistAlbums, err := client.FindAlbumsByFilter("artist", query)
				if err != nil {
					log.Printf("Error searching albums by artist: %v", err)
				}

				// Search by album name
				nameAlbums, err := client.FindAlbumsByFilter("album", query)
				if err != nil {
					log.Printf("Error searching albums by name: %v", err)
				}

				// Merge and Deduplicate
				seen := make(map[string]bool)
				results := make([]models.Album, 0)

				for _, a := range artistAlbums {
					if !seen[a.ID] {
						results = append(results, a)
						seen[a.ID] = true
					}
				}

				for _, a := range nameAlbums {
					if !seen[a.ID] {
						results = append(results, a)
						seen[a.ID] = true
					}
				}

				// Enrich results
				// We skip server-side enrichment to allow instant return of search results.
				// The frontend will lazy-load details for each album in parallel.
				albums = results
			} else {
				albums, _ = cache.SearchAlbums(query, 0, 100)
			}

			if len(albums) > 0 {
				select {
				case client.wsSend <- models.WSMessage{
					Type: "search_results",
					Data: map[string]interface{}{
						"category": "albums",
						"items":    albums,
					},
				}:
				case <-client.Ctx.Done():
					return
				}
			}
		}()
	}

	// 2. Search Artists
	if runArtists {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := mpd.GetClient().SendCommand(fmt.Sprintf("%s artist \"%s\"", cmdType, query))
			if err == nil {
				artists := parseUniqueTags(resp, "Artist")
				if len(artists) > 0 {
					select {
					case client.wsSend <- models.WSMessage{
						Type: "search_results",
						Data: map[string]interface{}{
							"category": "artists",
							"items":    artists,
						},
					}:
					case <-client.Ctx.Done():
						return
					}
				}
			}
		}()
	}

	// 3. Search Genres
	if runGenres {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := mpd.GetClient().SendCommand(fmt.Sprintf("%s genre \"%s\"", cmdType, query))
			if err == nil {
				genres := parseUniqueTags(resp, "Genre")
				if len(genres) > 0 {
					select {
					case client.wsSend <- models.WSMessage{
						Type: "search_results",
						Data: map[string]interface{}{
							"category": "genres",
							"items":    genres,
						},
					}:
					case <-client.Ctx.Done():
						return
					}
				}
			}
		}()
	}

	// 4. Search Dates
	if runDates {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := mpd.GetClient().SendCommand(fmt.Sprintf("%s date \"%s\"", cmdType, query))
			if err == nil {
				dates := parseUniqueTags(resp, "Date")
				if len(dates) > 0 {
					select {
					case client.wsSend <- models.WSMessage{
						Type: "search_results",
						Data: map[string]interface{}{
							"category": "dates",
							"items":    dates,
						},
					}:
					case <-client.Ctx.Done():
						return
					}
				}
			}
		}()
	}

	// 5. Search Songs (by title)
	if runSongs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := mpd.GetClient().SendCommand(fmt.Sprintf("%s title \"%s\"", cmdType, query))
			if err == nil {
				songs := parseSongs(resp)
				if len(songs) > 0 {
					select {
					case client.wsSend <- models.WSMessage{
						Type: "search_results",
						Data: map[string]interface{}{
							"category": "songs",
							"items":    songs,
						},
					}:
					case <-client.Ctx.Done():
						return
					}
				}
			}
		}()
	}

	wg.Wait()
}

func parseUniqueTags(resp string, key string) []string {
	lines := strings.Split(strings.TrimSpace(resp), "\n")
	seen := make(map[string]bool)
	results := make([]string, 0)
	prefix := key + ": "
	for _, line := range lines {
		if strings.HasPrefix(line, prefix) {
			val := strings.TrimPrefix(line, prefix)
			if val != "" && !seen[val] {
				seen[val] = true
				results = append(results, val)
			}
		}
	}
	return results
}

func parseSongs(resp string) []models.Song {
	lines := strings.Split(strings.TrimSpace(resp), "\n")
	songs := make([]models.Song, 0)
	var currentSong *models.Song

	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ": ", 2)
		if len(parts) != 2 {
			continue
		}
		key, value := parts[0], parts[1]

		if key == "file" {
			if currentSong != nil {
				songs = append(songs, *currentSong)
			}
			currentSong = &models.Song{Path: value}
		} else if currentSong != nil {
			switch key {
			case "Title":
				currentSong.Title = value
			case "Artist":
				currentSong.Artist = value
			case "Album":
				currentSong.Album = value
			case "Date":
				currentSong.Date = value
			case "Genre":
				currentSong.Genre = value
			case "Track":
				currentSong.Track = value
			case "Disc":
				currentSong.Disc = value
			case "duration", "Time":
				// duration is often float in some MPD versions, Time is integer seconds
				var d float64
				fmt.Sscanf(value, "%f", &d)
				currentSong.Duration = int(d)
			}
		}
	}
	if currentSong != nil {
		songs = append(songs, *currentSong)
	}
	return songs
}
