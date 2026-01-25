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
	RsyncRemoteTarget     string `json:"rsyncRemoteTarget"`
	RsyncOptions          string `json:"rsyncOptions"`
	RandomAlbumCount      int    `json:"randomAlbumCount"`
	EnableActivityRefresh bool   `json:"enableActivityRefresh"`
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
			}
			return saveLocked()
		}
		return err
	}

	instance = &Config{}
	return json.Unmarshal(data, instance)
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

// Custom UnmarshalJSON to handle both old and new field names
func (c *Config) UnmarshalJSON(data []byte) error {
	// Create a temporary struct with both naming conventions
	var temp struct {
		MPDHost               *string `json:"mpdHost"`
		MPDPort               *int    `json:"mpdPort"`
		Host                  *string `json:"host"` // Support old naming
		Port                  *int    `json:"port"` // Support old naming
		MPDPassword           *string `json:"mpdPassword"`
		MusicRoot             *string `json:"musicRoot"`
		CoverArtRoot          *string `json:"coverArtRoot"`
		CoverArtBaseUrl       *string `json:"coverArtBaseUrl"`
		DiscogsToken          *string `json:"discogsToken"`
		RsyncRemoteTarget     *string `json:"rsyncRemoteTarget"`
		RsyncOptions          *string `json:"rsyncOptions"`
		RandomAlbumCount      *int    `json:"randomAlbumCount"`
		EnableActivityRefresh *bool   `json:"enableActivityRefresh"`
	}

	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	// Only set values if they were provided in the JSON (not nil)
	if temp.MPDHost != nil {
		c.MPDHost = *temp.MPDHost
	} else if temp.Host != nil {
		c.MPDHost = *temp.Host
	}

	if temp.MPDPort != nil {
		c.MPDPort = *temp.MPDPort
	} else if temp.Port != nil {
		c.MPDPort = *temp.Port
	}

	if temp.MPDPassword != nil {
		c.MPDPassword = *temp.MPDPassword
	}
	if temp.MusicRoot != nil {
		c.MusicRoot = *temp.MusicRoot
	}
	if temp.CoverArtRoot != nil {
		c.CoverArtRoot = *temp.CoverArtRoot
	}
	if temp.CoverArtBaseUrl != nil {
		c.CoverArtBaseUrl = *temp.CoverArtBaseUrl
	}
	if temp.DiscogsToken != nil {
		c.DiscogsToken = *temp.DiscogsToken
	}
	if temp.RsyncRemoteTarget != nil {
		c.RsyncRemoteTarget = *temp.RsyncRemoteTarget
	}
	if temp.RsyncOptions != nil {
		c.RsyncOptions = *temp.RsyncOptions
	}
	if temp.RandomAlbumCount != nil {
		c.RandomAlbumCount = *temp.RandomAlbumCount
	}
	if temp.EnableActivityRefresh != nil {
		c.EnableActivityRefresh = *temp.EnableActivityRefresh
	}

	return nil
}
