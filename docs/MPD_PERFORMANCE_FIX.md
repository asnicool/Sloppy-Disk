# MPD Performance Fix Summary

## Issue Description
As soon as the backend started, the MPD process would take 100% CPU and the backend would experience I/O timeout errors.

## Root Cause - INOTIFY WATCH LIMIT

The **primary cause** is the Linux kernel inotify watch limit being exceeded. MPD logs show:
```
inotify: Failed to register /media/music/algeria: inotify_add_watch() has failed: No space left on device
```

This is NOT a disk space issue - it means the kernel parameter `fs.inotify.max_user_watches` (default: 8192) is too low for the large music library (~28,000 albums). When MPD tries to register inotify watches for all directories and fails, it keeps retrying, consuming 100% CPU.

### How Backend Triggers This

1. Backend starts and triggers album cache refresh
2. Cache refresh sends MPD commands that may trigger database updates
3. MPD tries to rescan and register inotify watches
4. Kernel limit is exceeded
5. MPD keeps retrying, consuming 100% CPU
6. Backend times out waiting for MPD

### Secondary Issues Addressed

#### BUG: Connection Liveness Check Corrupts MPD Protocol
In [`backend/internal/mpd/client.go:128-149`](backend/internal/mpd/client.go:128-149), the `EnsureConnection()` function had a flawed liveness check:

```go
// Check if connection is still alive
c.conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
one := make([]byte, 1)
if _, err := c.conn.Read(one); err != nil {
    ...
}
```

**Problem:** This code "peeks" at the connection by reading one byte, which **consumes protocol data** meant for the next command, corrupting the MPD protocol state.

### Contributing Factors

1. **Excessive Command Rate on Startup:** The album cache maintenance (`MaintainRandomBuffer()`) sends ~10 MPD commands per second during startup, running 24 batches of enrichment commands without adequate delays.

2. **Potential Concurrent Refreshes:** Multiple `Refresh()` calls could run simultaneously if database changes were detected, multiplying the load on MPD.

3. **No Rate Limiting:** Commands were sent back-to-back without giving MPD time to process them efficiently.

## Changes Made

### 1. Fixed MPD Connection Liveness Check ([`backend/internal/mpd/client.go`](backend/internal/mpd/client.go))

**Removed the buggy connection liveness check** that was consuming protocol data. Now the code relies on actual command failures (timeouts, I/O errors) to detect broken connections, which is the correct approach.

```go
// Don't perform liveness check - rely on actual command failures to detect broken connections.
// Previous liveness check used Read() which consumed protocol data, causing corruption.
// If the connection is broken, actual commands will fail with timeout or I/O errors.

if c.conn == nil {
    // Create new connection
}
```

### 2. Added Concurrent Refresh Protection ([`backend/internal/albumcache/cache.go`](backend/internal/albumcache/cache.go))

Added a mutex to prevent multiple `Refresh()` operations from running simultaneously:

```go
// Prevent concurrent refreshes
refreshMutex sync.Mutex
refreshing  bool
```

The `Refresh()` function now checks if a refresh is already in progress and skips if so.

### 3. Reduced MPD Command Rate ([`backend/internal/albumcache/cache.go`](backend/internal/albumcache/cache.go))

Increased the delay between buffer enrichment batches from 50ms to 200ms to reduce the load on MPD:

```go
// Yield/Sleep to allow other MPD commands to interleave
// Increased from 50ms to 200ms to reduce MPD load during startup
time.Sleep(200 * time.Millisecond)
```

**Impact:** This reduces the command rate from ~10 per second to ~2-3 per second during buffer maintenance, giving MPD more time to process each command efficiently.

## Expected Results

### With Inotify Limit Fix (System Configuration)
1. **MPD CPU usage stays normal** - No more 100% CPU spikes
2. **No more inotify errors** - All directories can be watched successfully
3. **Faster database updates** - MPD doesn't waste CPU retrying watch registrations

### With Backend Fixes
1. **Reduced command rate** - Fewer potential database updates triggering MPD rescans
2. **No concurrent refreshes** - Only one cache refresh at a time
3. **Better protocol integrity** - Connection doesn't consume data meant for other commands

## REQUIRED: Fix Inotify Watch Limits

### Immediate Fix (Temporary)
```bash
sudo sysctl fs.inotify.max_user_watches=524288
sudo sysctl fs.inotify.max_user_instances=256
```

### Permanent Fix (Persistent)
Add to `/etc/sysctl.conf`:
```
fs.inotify.max_user_watches = 524288
fs.inotify.max_user_instances = 256
```

Then apply:
```bash
sudo sysctl -p
```

### Optional: Reduce MPD Inotify Usage

In MPD config (`~/.config/mpd/mpd.conf` or `/etc/mpd.conf`):
```conf
follow_outside_symlinks "no"
follow_inside_symlinks "no"
auto_update "no"
```

Restart MPD after changes:
```bash
sudo systemctl restart mpd
# or
pkill mpd && mpd
```

## Testing Recommendations

### 1. Check Current Inotify Limits
```bash
cat /proc/sys/fs/inotify/max_user_watches
cat /proc/sys/fs/inotify/max_user_instances
```

### 2. Increase Inotify Limits
```bash
sudo sysctl fs.inotify.max_user_watches=524288
sudo sysctl fs.inotify.max_user_instances=256
```

### 3. Restart MPD and Monitor
```bash
# Check current MPD process
ps aux | grep mpd

# Restart MPD
sudo systemctl restart mpd
# or
pkill mpd && mpd

# Monitor MPD CPU usage
top -p $(pgrep mpd)

# Check MPD logs for inotify errors
journalctl -u mpd -f
# or
tail -f /var/log/mpd/mpd.log
```

### 4. Test Backend
1. Start the backend server
2. Check backend logs: `tail -f backend/server.log`
3. Monitor for "Random buffer appended X items" messages
4. Verify no I/O timeout errors
5. Test normal playback and browsing functions

## Files Modified

- [`backend/internal/mpd/client.go`](backend/internal/mpd/client.go) - Removed buggy connection liveness check
- [`backend/internal/albumcache/cache.go`](backend/internal/albumcache/cache.go) - Added concurrent refresh protection and reduced command rate

## Next Steps

Monitor the system after these fixes. If issues persist, consider:
1. Further increasing the delay between batches
2. Implementing proper command queuing with rate limiting
3. Adding circuit breaker for excessive MPD failures
