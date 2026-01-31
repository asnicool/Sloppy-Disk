package metadata

import (
	"context"
	"testing"

	"mpd-client-modern/internal/models"
)

// MockError is a simple error type for testing
type MockError struct {
	Message string
}

func (e *MockError) Error() string {
	return e.Message
}

// mockProvider is a simple mock implementation of Provider for testing
type mockProvider struct {
	name     string
	results  []models.MetadataCandidate
	details  *models.MetadataCandidate
	err      error
}

func (m *mockProvider) Name() string {
	return m.name
}

func (m *mockProvider) Search(artist, album string) ([]models.MetadataCandidate, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.results, nil
}

func (m *mockProvider) GetReleaseDetails(externalID string) (*models.MetadataCandidate, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.details, nil
}

func (m *mockProvider) GetCoverArt(artist, album string) ([]models.CoverArtCandidate, error) {
	if m.err != nil {
		return nil, m.err
	}
	return []models.CoverArtCandidate{}, nil
}

func TestNewAggregator(t *testing.T) {
	aggregator := NewAggregator()
	if aggregator == nil {
		t.Fatal("Expected aggregator to be created, got nil")
	}
}

func TestAggregator_Search(t *testing.T) {
	// Create mock providers with different results
	provider1 := &mockProvider{
		name: "Provider1",
		results: []models.MetadataCandidate{
			{
				ExternalID: "id1",
				Artist:     "Test Artist",
				Album:      "Test Album",
				Source:     "Provider1",
			},
		},
	}

	provider2 := &mockProvider{
		name: "Provider2",
		results: []models.MetadataCandidate{
			{
				ExternalID: "id2",
				Artist:     "Test Artist",
				Album:      "Test Album",
				Source:     "Provider2",
			},
		},
	}

	aggregator := &Aggregator{}
	aggregator.AddProvider(provider1)
	aggregator.AddProvider(provider2)

	results, err := aggregator.Search(context.Background(), "Test Artist", "Test Album", []string{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should have 2 results (1 from each provider)
	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}
}

func TestAggregator_Search_Deduplication(t *testing.T) {
	// Create mock providers with duplicate results (same external ID and source)
	provider1 := &mockProvider{
		name: "Provider1",
		results: []models.MetadataCandidate{
			{
				ExternalID: "same-id",
				Artist:     "Test Artist",
				Album:      "Test Album",
				Source:     "Provider1",
			},
		},
	}

	provider2 := &mockProvider{
		name: "Provider2",
		results: []models.MetadataCandidate{
			{
				ExternalID: "same-id",
				Artist:     "Test Artist",
				Album:      "Test Album",
				Source:     "Provider2",
			},
		},
	}

	aggregator := &Aggregator{}
	aggregator.AddProvider(provider1)
	aggregator.AddProvider(provider2)

	results, err := aggregator.Search(context.Background(), "Test Artist", "Test Album", []string{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should have 2 results (different sources)
	if len(results) != 2 {
		t.Errorf("Expected 2 results (different sources), got %d", len(results))
	}
}

func TestAggregator_Search_NoResults(t *testing.T) {
	// Create mock providers with no results
	provider := &mockProvider{
		name:    "EmptyProvider",
		results: []models.MetadataCandidate{},
	}

	aggregator := &Aggregator{}
	aggregator.AddProvider(provider)

	results, err := aggregator.Search(context.Background(), "Test Artist", "Test Album", []string{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should have 0 results
	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}

func TestAggregator_Search_ProviderError(t *testing.T) {
	// Create one provider that errors and one that works
	errorProvider := &mockProvider{
		name: "ErrorProvider",
		err:  &MockError{Message: "provider unavailable"},
	}

	workingProvider := &mockProvider{
		name: "WorkingProvider",
		results: []models.MetadataCandidate{
			{
				ExternalID: "id1",
				Artist:     "Test Artist",
				Album:      "Test Album",
				Source:     "WorkingProvider",
			},
		},
	}

	aggregator := &Aggregator{}
	aggregator.AddProvider(errorProvider)
	aggregator.AddProvider(workingProvider)

	results, err := aggregator.Search(context.Background(), "Test Artist", "Test Album", []string{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should have results from the working provider despite the error from the other
	if len(results) == 0 {
		t.Error("Expected results from working provider despite error from errorProvider")
	}

	if len(results) > 0 && results[0].Source != "WorkingProvider" {
		t.Errorf("Expected source 'WorkingProvider', got '%s'", results[0].Source)
	}
}

func TestAggregator_Search_NoProviders(t *testing.T) {
	aggregator := &Aggregator{}

	results, err := aggregator.Search(context.Background(), "Test Artist", "Test Album", []string{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results with no providers, got %d", len(results))
	}
}

func TestAggregator_GetReleaseDetails(t *testing.T) {
	// Create mock provider with details
	details := &models.MetadataCandidate{
		ExternalID: "id1",
		Artist:     "Test Artist",
		Album:      "Test Album",
		Source:     "TestProvider",
	}

	provider := &mockProvider{
		name:    "TestProvider",
		details: details,
	}

	aggregator := &Aggregator{}
	aggregator.AddProvider(provider)

	result, err := aggregator.GetReleaseDetails(context.Background(), "TestProvider", "id1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("Expected details, got nil")
	}

	// Check that details are from the provider
	if result.Source != "TestProvider" {
		t.Errorf("Expected source 'TestProvider', got '%s'", result.Source)
	}
}

func TestAggregator_GetReleaseDetails_NotFound(t *testing.T) {
	aggregator := &Aggregator{}

	result, err := aggregator.GetReleaseDetails(context.Background(), "NonExistent", "id1")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if result != nil {
		t.Error("Expected nil result for non-existent provider")
	}
}

func TestAggregator_SearchCoverArt(t *testing.T) {
	provider1 := &mockProvider{
		name: "Provider1",
	}

	provider2 := &mockProvider{
		name: "Provider2",
	}

	aggregator := &Aggregator{}
	aggregator.AddProvider(provider1)
	aggregator.AddProvider(provider2)

	results, err := aggregator.SearchCoverArt(context.Background(), "Test Artist", "Test Album")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should have no results from mock providers
	if len(results) != 0 {
		t.Errorf("Expected 0 cover art results from mock, got %d", len(results))
	}
}

func TestNormalizeString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Álbum", "album"},
		{"The Beatles", "the beatles"},
		{"Björk", "bjork"},
		{"Köln", "koln"},
		{"Café", "cafe"},
	}

	for _, tc := range tests {
		result := normalizeString(tc.input)
		if result != tc.expected {
			t.Errorf("normalizeString(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}

func TestStringSimilarity(t *testing.T) {
	tests := []struct {
		s1       string
		s2       string
		minScore float64
	}{
		{"test", "test", 0.9},
		{"test album", "test album", 0.9},
		{"test album", "test", 0.4},
		{"completely different", "nothing alike", 0.0},
	}

	for _, tc := range tests {
		score := stringSimilarity(tc.s1, tc.s2)
		if score < tc.minScore {
			t.Errorf("stringSimilarity(%q, %q) = %f, expected at least %f", tc.s1, tc.s2, score, tc.minScore)
		}
	}
}

func TestRemoveDiacritics(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"áéíóú", "aeiou"},
		{"ñç", "nc"},
		{"björk", "bjork"},
		{"café", "cafe"},
	}

	for _, tc := range tests {
		result := removeDiacritics(tc.input)
		if result != tc.expected {
			t.Errorf("removeDiacritics(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}

func TestDeduplicate(t *testing.T) {
	candidates := []models.MetadataCandidate{
		{ExternalID: "id1", Source: "Provider1"},
		{ExternalID: "id1", Source: "Provider1"}, // Duplicate
		{ExternalID: "id2", Source: "Provider1"},
		{ExternalID: "id1", Source: "Provider2"}, // Different source, not duplicate
	}

	result := deduplicate(candidates)

	// Should deduplicate only within same source
	if len(result) != 3 {
		t.Errorf("Expected 3 deduplicated candidates, got %d", len(result))
	}
}

func TestDeduplicateCoverArt(t *testing.T) {
	candidates := []models.CoverArtCandidate{
		{URL: "http://example.com/art1.jpg"},
		{URL: "http://example.com/art1.jpg"}, // Duplicate
		{URL: "http://example.com/art2.jpg"},
	}

	result := deduplicateCoverArt(candidates)

	// Should have 2 unique cover art candidates
	if len(result) != 2 {
		t.Errorf("Expected 2 unique cover art candidates, got %d", len(result))
	}
}
