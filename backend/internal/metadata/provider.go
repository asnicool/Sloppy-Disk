package metadata

import (
	"mpd-client-modern/internal/models"
)

// Provider defines the interface for music metadata providers
type Provider interface {
	// Name returns the provider identifier
	Name() string

	// Search searches for albums matching the given artist and album name
	Search(artist, album string) ([]models.MetadataCandidate, error)

	// GetReleaseDetails fetches detailed metadata for a specific release
	GetReleaseDetails(externalID string) (*models.MetadataCandidate, error)

	// GetCoverArt fetches cover art candidates for an album
	GetCoverArt(artist, album string) ([]models.CoverArtCandidate, error)
}

// ProviderError represents an error from a specific provider
type ProviderError struct {
	Provider string
	Err      error
}

func (e *ProviderError) Error() string {
	return e.Provider + ": " + e.Err.Error()
}

func (e *ProviderError) Unwrap() error {
	return e.Err
}

// NewProviderError creates a new ProviderError
func NewProviderError(provider string, err error) error {
	return &ProviderError{Provider: provider, Err: err}
}
