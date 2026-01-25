package sync

import (
	"testing"
)

func TestSyncStatus(t *testing.T) {
	m := GetManager()
	
	status := m.GetStatus()
	if status.IsRunning {
		t.Error("Expected sync not to be running initially")
	}

	m.mu.Lock()
	m.status.IsRunning = true
	m.status.Progress = 50.0
	m.mu.Unlock()

	status = m.GetStatus()
	if !status.IsRunning {
		t.Error("Expected sync to be running")
	}
	if status.Progress != 50.0 {
		t.Errorf("Expected progress 50.0, got %f", status.Progress)
	}

	m.finish(true, "")
	status = m.GetStatus()
	if status.IsRunning {
		t.Error("Expected sync to be finished")
	}
	if !status.LastSuccess {
		t.Error("Expected last success to be true")
	}
	if status.Progress != 100.0 {
		t.Errorf("Expected progress 100.0, got %f", status.Progress)
	}
}
