# MPD Performance Issue Analysis

## Problem Description
As soon as the backend starts, MPD process takes 100% of its CPU and the backend complains about I/O timeouts.

## Root Causes Identified

### 1. CRITICAL BUG: Connection Liveness Check Corrupts Protocol (client.go:132-149)

```go
if c.conn != nil {
    // Check if connection is still alive
    c.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
    one := make([]byte, 1)
    if _, err := c.conn.Read(one); err != nil {
        if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
            // Still alive
            c.conn.SetReadDeadline(time.Time{})
        } else {
            c.conn.Close()
            c.conn = nil
            c.isConnected = false
        }
    } else {
        // Should not happen if we are just checking
        c.conn.SetReadDeadline(time.Time{})
    }
}
```

**Problem:**
- This code "peeks" at the connection to check if it's alive
- If there's ANY data waiting (even just a newline from MPD), it consumes ONE BYTE
- This corrupts the MPD protocol - the next command will receive garbled data
- MPD server has to deal with malformed requests, causing high CPU and errors

**When This Triggers:**
- Every time `EnsureConnection()` is called
- Called before EVERY MPD command
- With connection pooling and frequent commands, this happens thousands of times

### 2. Excessive MPD Command Rate on Startup

**What happens on server startup:**

1. `main.go` starts album cache refresh (line 48)
2. `Refresh()` calls `GetAllAlbumKeys()` - one heavy "list album group albumartist group date group genre" command
   - Log: "Fetched 28386 album keys in 1.527662284s"
3. After refresh, `MaintainRandomBuffer()` runs in background
4. `MaintainRandomBuffer()` enriches 5 albums at a time (RefillBatchSize = 5)
   - For each batch of 5 albums:
     - Calls `GetAlbumStats()` - sends 5 `count album ...` commands
     - Calls `GetAlbumRepresentatives()` - sends 5 `find album ... window 0:1` commands
   - Total: 10 commands per batch
   - Each batch takes ~966ms (from logs)
   - This continues until buffer has 120 albums = 24 batches
   - Total: ~240 commands taking ~23 seconds

**Commands running continuously:**
- 10 commands every ~1 second (from buffer maintenance)
- WebSocket broadcaster checks status every 60 seconds
- Any user interactions trigger additional commands

### 3. WebSocket Broadcaster May Trigger Additional Refreshes

The broadcaster monitors for "database" changes via IDLE and triggers cache refresh:
```go
for _, subsystem := range changedSubsystems {
    if subsystem == "database" {
        databaseChanged = true
        log.Println("[Broadcaster] Database changed, triggering cache refresh...")
        go func() {
            if databaseChangeCallback != nil {
                databaseChangeCallback()  // Triggers albumcache.Refresh()
            }
        }()
    }
}
```

If IDLE reports database changes frequently, this creates multiple concurrent refresh operations.

### 4. Connection Pool Semaphores May Cause Head-of-Line Blocking

```go
func (c *Client) Execute(fn func(*Connection) error) error {
    // Acquire semaphore token with timeout
    select {
    case c.semaphore <- struct{}{}:
        defer func() { <-c.semaphore }()
    case <-time.After(500 * time.Millisecond):
        return fmt.Errorf("server busy: too many concurrent MPD commands")
    }
    ...
}
```

With only 8 semaphore slots and the connection liveness check bug, slow operations block other operations.

## What Was Changed in Recent Updates

From git history, the critical changes were in commit `e76f5d0`:
- "feat: optimize MPD connection architecture to eliminate head-of-line blocking and improve performance with circuit breaker implementation"

**Changes introduced:**
1. Connection pooling with semaphores
2. Dedicated IDLE client in WebSocket broadcaster
3. Database change monitoring via IDLE
4. Connection liveness check (the buggy code)

**The connection liveness check appears to be the culprit** - it was likely added to detect stale connections but has a fatal flaw.

## Evidence from Logs

1. Commands taking ~966ms each (slower than expected)
2. "Error enriching all albums: failed to get stats: read tcp ... i/o timeout"
3. "failed to connect to MPD: dial tcp ... i/o timeout"
4. MPD eventually becomes unresponsive (100% CPU)

## Why MPD CPU Goes to 100%

1. The buggy connection liveness check corrupts the protocol
2. MPD receives malformed commands or missing data
3. MPD has to parse and error-handle these malformed requests
4. With ~10 commands per second, MPD spends all CPU time handling errors
5. Backend starts timing out because MPD is too busy to respond
6. This creates a death spiral: more retries → more corruption → more CPU

## Recommended Fixes

### Fix 1: Remove the Buggy Connection Liveness Check

The simplest fix is to remove the connection liveness check entirely and rely on errors from actual commands to detect connection failures.

### Fix 2: Implement a Better Liveness Check

If a liveness check is needed, use a proper approach:
- Send a simple NOOP command like `ping` (if MPD supports it)
- Or just rely on read/write timeouts on actual commands
- Don't peek at the connection buffer

### Fix 3: Rate Limit MPD Commands on Startup

Add backoff/delay between buffer enrichment batches:
- Currently: no delay between batches
- Should add: `time.Sleep(100 * time.Millisecond)` between batches

### Fix 4: Prevent Concurrent Cache Refreshes

Add a mutex to ensure only one `Refresh()` runs at a time.

## Files to Modify

1. `backend/internal/mpd/client.go` - Fix the liveness check
2. `backend/internal/albumcache/cache.go` - Add rate limiting and prevent concurrent refreshes
