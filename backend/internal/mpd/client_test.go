package mpd

import (
	"fmt"
	"testing"
	"time"

	"mpd-client-modern/internal/config"
)

func TestPathMapping(t *testing.T) {
	cfg := &config.Config{
		MusicRoot: "/mnt/music",
	}
	config.Save(cfg)

	client := GetClient()

	rel := "Pink Floyd/Dark Side of the Moon/01. Speak to Me.mp3"
	abs := client.ToAbsolutePath(rel)
	expectedAbs := "/mnt/music/Pink Floyd/Dark Side of the Moon/01. Speak to Me.mp3"

	if abs != expectedAbs {
		t.Errorf("Expected absolute path %s, got %s", expectedAbs, abs)
	}

	relBack, err := client.ToRelativePath(abs)
	if err != nil {
		t.Errorf("Failed to map back to relative path: %v", err)
	}
	if relBack != rel {
		t.Errorf("Expected relative path %s, got %s", rel, relBack)
	}
}

func TestParseResponse(t *testing.T) {
	resp := "Artist: Pink Floyd\nAlbum: Dark Side of the Moon\nTitle: Speak to Me\nOK\n"
	attrs := ParseResponse(resp)

	if attrs["Artist"] != "Pink Floyd" {
		t.Errorf("Expected Artist Pink Floyd, got %s", attrs["Artist"])
	}
	if attrs["Album"] != "Dark Side of the Moon" {
		t.Errorf("Expected Album Dark Side of the Moon, got %s", attrs["Album"])
	}
}

func TestPool(t *testing.T) {
	client := GetClient()
	if client.pool == nil {
		t.Fatal("Client pool not initialized")
	}

	// Draining the pool
	c1 := client.acquire()
	c2 := client.acquire()

	if c1 == nil || c2 == nil {
		t.Fatal("Failed to acquire connections")
	}

	if c1 == c2 {
		t.Error("Acquired the same connection instance twice")
	}

	// Mock connection
	c1.isConnected = true
	c2.isConnected = true

	// Releasing
	client.release(c1)
	client.release(c2)

	// Should be back in pool
	c3 := client.acquire()
	if c3 != c1 && c3 != c2 {
		t.Error("Did not reuse connection from pool")
	}
}
func TestExecute(t *testing.T) {
	client := GetPool()

	err := client.Execute(func(conn *Connection) error {
		if conn == nil {
			return fmt.Errorf("connection is nil")
		}
		// Mock connection state
		conn.isConnected = true
		return nil
	})

	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}
}

func TestConcurrency(t *testing.T) {
	client := GetPool()
	const concurrentRequests = 10
	errChan := make(chan error, concurrentRequests)

	for i := 0; i < concurrentRequests; i++ {
		go func() {
			err := client.Execute(func(conn *Connection) error {
				// Simulate some work
				time.Sleep(100 * time.Millisecond)
				conn.isConnected = true
				return nil
			})
			errChan <- err
		}()
	}

	for i := 0; i < concurrentRequests; i++ {
		err := <-errChan
		if err != nil {
			t.Errorf("Concurrent request %d failed: %v", i, err)
		}
	}
}

func TestExecuteRetry(t *testing.T) {
	client := GetPool()

	attempts := 0
	err := client.Execute(func(conn *Connection) error {
		attempts++
		if attempts == 1 {
			return fmt.Errorf("EOF") // Simulate connection error
		}
		conn.isConnected = true
		return nil
	})

	if err != nil {
		t.Errorf("Execute failed: %v", err)
	}
	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
}
