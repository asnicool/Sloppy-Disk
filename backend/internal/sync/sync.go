package sync

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	"mpd-client-modern/internal/config"
	"mpd-client-modern/internal/models"
)

type Manager struct {
	mu     sync.RWMutex
	status models.SyncStatus
}

var (
	defaultManager *Manager
	once           sync.Once
)

func GetManager() *Manager {
	once.Do(func() {
		defaultManager = &Manager{
			status: models.SyncStatus{
				LastRun: time.Time{},
			},
		}
	})
	return defaultManager
}

func (m *Manager) GetStatus() models.SyncStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status
}

func (m *Manager) StartSync(ctx context.Context) error {
	m.mu.Lock()
	if m.status.IsRunning {
		m.mu.Unlock()
		return fmt.Errorf("sync already in progress")
	}
	m.status.IsRunning = true
	m.status.Progress = 0
	m.mu.Unlock()

	go m.runRsync(ctx)
	return nil
}

func (m *Manager) runRsync(ctx context.Context) {
	cfg := config.Get()
	
	// rsync -avz --progress /local/path/ user@host:/remote/path/
	args := []string{"-avz", "--progress"}
	if cfg.RsyncOptions != "" {
		args = append(args, strings.Fields(cfg.RsyncOptions)...)
	}
	args = append(args, cfg.MusicRoot+"/", cfg.RsyncRemoteTarget)

	cmd := exec.CommandContext(ctx, "rsync", args...)
	
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		m.finish(false, err.Error())
		return
	}

	if err := cmd.Start(); err != nil {
		m.finish(false, err.Error())
		return
	}

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		// Simple progress parsing (rsync output format varies)
		if strings.Contains(line, "%") {
			parts := strings.Fields(line)
			for _, p := range parts {
				if strings.HasSuffix(p, "%") {
					var progress float64
					fmt.Sscanf(p, "%f%%", &progress)
					m.mu.Lock()
					m.status.Progress = progress
					m.mu.Unlock()
				}
			}
		}
	}

	if err := cmd.Wait(); err != nil {
		m.finish(false, err.Error())
	} else {
		m.finish(true, "")
	}
}

func (m *Manager) finish(success bool, errMsg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status.IsRunning = false
	m.status.LastRun = time.Now()
	m.status.LastSuccess = success
	m.status.LastError = errMsg
	if success {
		m.status.Progress = 100
	}
}
