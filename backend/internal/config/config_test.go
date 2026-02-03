package config

import (
	"os"
	"testing"
)

func TestConfigLoadSave(t *testing.T) {
	configPath = "test_config.json"
	defer os.Remove(configPath)

	cfg := &Config{
		MPDHost:   "1.2.3.4",
		MPDPort:   1234,
		MusicRoot: "/tmp/music",
	}

	err := Save(cfg)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Reset instance to force reload
	instance = nil

	err = Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	loaded := Get()
	if loaded.MPDHost != "1.2.3.4" {
		t.Errorf("Expected MPDHost 1.2.3.4, got %s", loaded.MPDHost)
	}
	if loaded.MPDPort != 1234 {
		t.Errorf("Expected MPDPort 1234, got %d", loaded.MPDPort)
	}
}

func TestConfigValidate(t *testing.T) {
	cfg := &Config{
		MPDHost: "",
		MPDPort: 6600,
	}
	if err := cfg.Validate(); err == nil {
		t.Error("Expected error for empty MPDHost, got nil")
	}

	cfg.MPDHost = "localhost"
	cfg.MPDPort = 0
	if err := cfg.Validate(); err == nil {
		t.Error("Expected error for invalid MPDPort, got nil")
	}
}

func TestConfigMigration_FreeDBToGNUDb(t *testing.T) {
	// Identify a temporary config file
	configPath = "test_config_migration.json"
	defer os.Remove(configPath)

	// Create a JSON with old field names
	oldJSON := []byte(`{
		"mpdHost": "localhost",
		"mpdPort": 6600,
		"musicRoot": "/tmp",
		"coverArtRoot": "/tmp",
		"coverArtBaseUrl": "http://localhost",
		"rsyncRemoteTarget": "",
		"rsyncOptions": "",
		"randomAlbumCount": 10,
		"enableActivityRefresh": true,
		"musicBrainzEnabled": true,
		"discogsEnabled": true,
		"freeDbEnabled": true,
		"albumArtEnabled": true
	}`)

	err := os.WriteFile(configPath, oldJSON, 0600)
	if err != nil {
		t.Fatalf("Failed to write mock config: %v", err)
	}

	// Reset instance
	instance = nil

	// Load config
	err = Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	loaded := Get()

	// Verify GNUDbEnabled is true (migrated from freeDbEnabled)
	if !loaded.GNUDbEnabled {
		t.Error("Expected GNUDbEnabled to be true (migrated from freeDbEnabled), got false")
	}

	// Verify FreeDBEnabled is also true (synced)
	if !loaded.FreeDBEnabled {
		t.Error("Expected FreeDBEnabled to be true (synced), got false")
	}

	// Test case where freeDbEnabled is false
	oldJSONFalse := []byte(`{
		"mpdHost": "localhost",
		"mpdPort": 6600,
		"freeDbEnabled": false
	}`)
	os.WriteFile(configPath, oldJSONFalse, 0600)
	instance = nil
	Load()
	loaded = Get()

	if loaded.GNUDbEnabled {
		t.Error("Expected GNUDbEnabled to be false (migrated from freeDbEnabled=false), got true")
	}
}

func TestConfigMigration_GNUDbExplicit(t *testing.T) {
	// Identify a temporary config file
	configPath = "test_config_explicit.json"
	defer os.Remove(configPath)

	// Create a JSON with new field name
	newJSON := []byte(`{
		"mpdHost": "localhost",
		"mpdPort": 6600,
		"gnuDbEnabled": true,
		"freeDbEnabled": false
	}`)
	// In the logic, if gnuDbEnabled is present, it takes precedence.

	err := os.WriteFile(configPath, newJSON, 0600)
	if err != nil {
		t.Fatalf("Failed to write mock config: %v", err)
	}

	// Reset instance
	instance = nil

	// Load config
	err = Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	loaded := Get()

	// Verify GNUDbEnabled is true (explicitly set)
	if !loaded.GNUDbEnabled {
		t.Error("Expected GNUDbEnabled to be true (explicit), got false")
	}
}
