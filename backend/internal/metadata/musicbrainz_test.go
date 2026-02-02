package metadata

import (
	"log"
	"testing"
)

func TestMusicBrainzProvider_Lib(t *testing.T) {
	provider := NewMusicBrainzProvider()

	log.Println("Searching for Ultravox - Lament...")
	candidates, err := provider.Search("Ultravox", "Lament")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if len(candidates) == 0 {
		t.Fatal("No candidates found")
	}

	for _, c := range candidates {
		log.Printf("Candidate: %s - %s (ID: %s)", c.Artist, c.Album, c.ExternalID)
	}

	// Pick first candidate and get details
	first := candidates[0]
	log.Printf("Fetching details for: %s", first.ExternalID)

	details, err := provider.GetReleaseDetails(first.ExternalID)
	if err != nil {
		t.Fatalf("GetReleaseDetails failed: %v", err)
	}

	log.Printf("Details: %s - %s (%d tracks)", details.Artist, details.Album, len(details.Tracks))
	if len(details.Tracks) == 0 {
		t.Error("No tracks found in details")
	}
}
