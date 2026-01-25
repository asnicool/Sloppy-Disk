package config

import (
	"os"
	"testing"
)

func TestConfigLoadSave(t *testing.T) {
	configPath = "test_config.json"
	defer os.Remove(configPath)

	cfg := &Config{
		MPDHost: "1.2.3.4",
		MPDPort: 1234,
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
