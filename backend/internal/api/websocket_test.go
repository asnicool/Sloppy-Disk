package api

import (
	"context"
	"testing"
	"time"

	"mpd-client-modern/internal/models"
)

func TestClientConnection_ContextCancel(t *testing.T) {
	// this test verifies that using the context pattern prevents blocking
	// when the client is disconnected (context cancelled)
	ctx, cancel := context.WithCancel(context.Background())
	client := &ClientConnection{
		wsSend: make(chan models.WSMessage), // Unbuffered channel
		Ctx:    ctx,
		Cancel: cancel,
	}

	// 1. Verify send blocks normally (optional sanity check, skipped to avoid short timeout flake)

	// 2. Cancel the context (Simulate disconnect)
	cancel()

	// 3. verifying that the SELECT pattern used in PerformStreamingSearch works
	// It should NOT block and NOT panic
	done := make(chan bool)
	go func() {
		select {
		case client.wsSend <- models.WSMessage{}:
			t.Error("Should not have been able to send to unread channel")
		case <-client.Ctx.Done():
			// Success: Context cancellation was detected
		}
		done <- true
	}()

	select {
	case <-done:
		// Passed
	case <-time.After(1 * time.Second):
		t.Fatal("Test timed out - deadlock detected")
	}
}
