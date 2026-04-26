package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
)

type Config struct {
	MPDHost               string `json:"mpdHost"`
	MPDPort               int    `json:"mpdPort"`
	MPDPassword           string `json:"mpdPassword,omitempty"`
	MusicRoot             string `json:"musicRoot"`
	CoverArtRoot          string `json:"coverArtRoot"`
	CoverArtBaseUrl       string `json:"coverArtBaseUrl"`
	DiscogsToken          string `json:"discogsToken,omitempty"`
	DiscogsKey            string `json:"discogsKey,omitempty"`
	DiscogsSecret         string `json:"discogsSecret,omitempty"`
	AlbumArtAPIKey        string `json:"albumArtApiKey,omitempty"`
	RsyncRemoteTarget     string `json:"rsyncRemoteTarget"`
	RsyncOptions          string `json:"rsyncOptions"`
	RandomAlbumCount      int    `json:"randomAlbumCount"`
	EnableActivityRefresh bool   `json:"enableActivityRefresh"`
	// Metadata provider settings
	MusicBrainzEnabled bool `json:"musicBrainzEnabled"`
	DiscogsEnabled     bool `json:"discogsEnabled"`
	FreeDBEnabled      bool `json:"freeDbEnabled"` // Deprecated: Use GNUDbEnabled
	GNUDbEnabled       bool `json:"gnuDbEnabled"`
	AlbumArtEnabled    bool `json:"albumArtEnabled"`
	// N50 HIFI Component settings
	N50Enabled       bool   `json:"n50Enabled"`
	N50Host          string `json:"n50Host"`
	N50Port          int    `json:"n50Port"`
	N50Input         string `json:"n50Input"`         // Input source (e.g., "DigitalIn1", "DigitalIn2", "MusicServer")
	N50AutoControl   bool   `json:"n50AutoControl"`   // Auto check/change input before playback
	N50IgnoreOnStart bool   `json:"n50IgnoreOnStart"` // Option to ignore N50 for starting playback
}

// ConfigDTO is used for partial updates and handling optional fields
type ConfigDTO struct {
	MPDHost               *string `json:"mpdHost"`
	MPDPort               *int    `json:"mpdPort"`
	Host                  *string `json:"host"` // Support old naming
	Port                  *int    `json:"port"` // Support old naming
	MPDPassword           *string `json:"mpdPassword"`
	MusicRoot             *string `json:"musicRoot"`
	CoverArtRoot          *string `json:"coverArtRoot"`
	CoverArtBaseUrl       *string `json:"coverArtBaseUrl"`
	DiscogsToken          *string `json:"discogsToken"`
	DiscogsKey            *string `json:"discogsKey"`
	DiscogsSecret         *string `json:"discogsSecret"`
	AlbumArtAPIKey        *string `json:"albumArtApiKey"`
	RsyncRemoteTarget     *string `json:"rsyncRemoteTarget"`
	RsyncOptions          *string `json:"rsyncOptions"`
	RandomAlbumCount      *int    `json:"randomAlbumCount"`
	EnableActivityRefresh *bool   `json:"enableActivityRefresh"`
	// Metadata provider settings
	MusicBrainzEnabled *bool `json:"musicBrainzEnabled"`
	DiscogsEnabled     *bool `json:"discogsEnabled"`
	FreeDBEnabled      *bool `json:"freeDbEnabled"`
	GNUDbEnabled       *bool `json:"gnuDbEnabled"`
	AlbumArtEnabled    *bool `json:"albumArtEnabled"`
	// N50 HIFI Component settings
	N50Enabled       *bool   `json:"n50Enabled"`
	N50Host          *string `json:"n50Host"`
	N50Port          *int    `json:"n50Port"`
	N50Input         *string `json:"n50Input"`
	N50AutoControl   *bool   `json:"n50AutoControl"`
	N50IgnoreOnStart *bool   `json:"n50IgnoreOnStart"`
}

var (
	instance   *Config
	mu         sync.RWMutex
	configPath = "config.json"
)

func Get() *Config {
	mu.RLock()
	defer mu.RUnlock()
	if instance == nil {
		// Fallback to defaults if not loaded
		return &Config{
			MPDHost: "127.0.0.1",
			MPDPort: 6600,
		}
	}
	return instance
}

func Load() error {
	mu.Lock()
	defer mu.Unlock()

	log.Printf("Loading configuration from: %s", configPath)
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create default config if it doesn't exist
			instance = &Config{
				MPDHost:               "192.168.1.90",
				MPDPort:               6600,
				RandomAlbumCount:      30,
				EnableActivityRefresh: true,
				MusicBrainzEnabled:    true,
				DiscogsEnabled:        true,
				GNUDbEnabled:          true,
				AlbumArtEnabled:       true,
				N50Enabled:            false,
				N50Port:               8102,
				N50Input:              "DigitalIn1",
				N50AutoControl:        true,
				N50IgnoreOnStart:      false,
			}
			return saveLocked()
		}
		return err
	}

	// Unmarshal into DTO to handle legacy fields and partial updates
	var dto ConfigDTO
	if err := json.Unmarshal(data, &dto); err != nil {
		return err
	}

	// Initialize with defaults
	instance = &Config{
		MPDHost:               "192.168.1.90",
		MPDPort:               6600,
		RandomAlbumCount:      30,
		EnableActivityRefresh: true,
		MusicBrainzEnabled:    true,
		DiscogsEnabled:        true,
		GNUDbEnabled:          true,
		AlbumArtEnabled:       true,
		N50Enabled:            false,
		N50Port:               8102,
		N50Input:              "DigitalIn1",
		N50AutoControl:        true,
		N50IgnoreOnStart:      false,
	}

	// Apply DTO values
	instance.ApplyDTO(&dto)

	return nil
}

func Save(c *Config) error {
	mu.Lock()
	defer mu.Unlock()
	instance = c
	return saveLocked()
}

func saveLocked() error {
	data, err := json.MarshalIndent(instance, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0600)
}

func (c *Config) Validate() error {
	if c.MPDHost == "" {
		return fmt.Errorf("MPD host is required")
	}
	if c.MPDPort <= 0 || c.MPDPort > 65535 {
		return fmt.Errorf("invalid MPD port")
	}
	return nil
}

// ApplyDTO updates the configuration with values from the provided DTO
// Only non-nil fields in the DTO are applied
func (c *Config) ApplyDTO(dto *ConfigDTO) {
	if dto.MPDHost != nil {
		c.MPDHost = *dto.MPDHost
	} else if dto.Host != nil {
		c.MPDHost = *dto.Host
	}

	if dto.MPDPort != nil {
		c.MPDPort = *dto.MPDPort
	} else if dto.Port != nil {
		c.MPDPort = *dto.Port
	}

	if dto.MPDPassword != nil {
		c.MPDPassword = *dto.MPDPassword
	}
	if dto.MusicRoot != nil {
		c.MusicRoot = *dto.MusicRoot
	}
	if dto.CoverArtRoot != nil {
		c.CoverArtRoot = *dto.CoverArtRoot
	}
	if dto.CoverArtBaseUrl != nil {
		c.CoverArtBaseUrl = *dto.CoverArtBaseUrl
	}
	if dto.DiscogsToken != nil {
		c.DiscogsToken = *dto.DiscogsToken
	}
	if dto.DiscogsKey != nil {
		c.DiscogsKey = *dto.DiscogsKey
	}
	if dto.DiscogsSecret != nil {
		c.DiscogsSecret = *dto.DiscogsSecret
	}
	if dto.AlbumArtAPIKey != nil {
		c.AlbumArtAPIKey = *dto.AlbumArtAPIKey
	}
	if dto.RsyncRemoteTarget != nil {
		c.RsyncRemoteTarget = *dto.RsyncRemoteTarget
	}
	if dto.RsyncOptions != nil {
		c.RsyncOptions = *dto.RsyncOptions
	}
	if dto.RandomAlbumCount != nil {
		c.RandomAlbumCount = *dto.RandomAlbumCount
	}
	if dto.EnableActivityRefresh != nil {
		c.EnableActivityRefresh = *dto.EnableActivityRefresh
	}

	// Metadata settings
	if dto.MusicBrainzEnabled != nil {
		c.MusicBrainzEnabled = *dto.MusicBrainzEnabled
	}
	if dto.DiscogsEnabled != nil {
		c.DiscogsEnabled = *dto.DiscogsEnabled
	}

	// Handle GNUDb/FreeDB backward compatibility
	if dto.GNUDbEnabled != nil {
		c.GNUDbEnabled = *dto.GNUDbEnabled
	} else if dto.FreeDBEnabled != nil {
		c.GNUDbEnabled = *dto.FreeDBEnabled
	}
	// Always keep FreeDBEnabled synced
	c.FreeDBEnabled = c.GNUDbEnabled

	if dto.AlbumArtEnabled != nil {
		c.AlbumArtEnabled = *dto.AlbumArtEnabled
	}

	// N50 settings
	if dto.N50Enabled != nil {
		c.N50Enabled = *dto.N50Enabled
	}
	if dto.N50Host != nil {
		c.N50Host = *dto.N50Host
	}
	if dto.N50Port != nil {
		c.N50Port = *dto.N50Port
	}
	if dto.N50Input != nil {
		c.N50Input = *dto.N50Input
	}
	if dto.N50AutoControl != nil {
		c.N50AutoControl = *dto.N50AutoControl
	}
	if dto.N50IgnoreOnStart != nil {
		c.N50IgnoreOnStart = *dto.N50IgnoreOnStart
	}
}
