# MPD Performance Issue - INOTIFY WATCH LIMIT

## Real Root Cause Identified

The MPD logs show:
```
inotify: Failed to register /media/music/algeria: inotify_add_watch() has failed: No space left on device
```

**This is NOT a disk space issue.** The "No space left on device" error from `inotify_add_watch()` means the Linux kernel has run out of **inotify watch instances**, which is controlled by the kernel parameter `fs.inotify.max_user_watches`.

## How This Happens

1. **Backend starts** and triggers album cache refresh
2. **Cache refresh** calls `client.GetAllAlbumKeys()` which queries MPD
3. **MPD database updates** may be triggered by backend commands
4. **MPD tries to scan music directory** and register inotify watches for monitoring changes
5. **Kernel limit exceeded** - the default `fs.inotify.max_user_watches` is too low for large music libraries
6. **MPD keeps retrying** to register watches, consuming 100% CPU
7. **Backend times out** because MPD is too busy to respond

## Why Backend Triggers This

Looking at the code flow:

1. **Initial cache refresh** (`main.go:48`):
   ```go
   go func() {
       log.Println("Initializing album cache...")
       if err := albumcache.GetCache().Refresh(); err != nil {
           log.Printf("Failed to refresh album cache: %v", err)
       }
   }()
   ```

2. **Database change detection** (via WebSocket broadcaster):
   - When MPD IDLE reports "database" changes, it triggers `Refresh()`
   - This may cause MPD to update its database
   - Each database update triggers new inotify watch attempts

3. **Aggressive background enrichment**:
   - `MaintainRandomBuffer()` runs continuously
   - Each batch sends multiple MPD commands
   - If any command causes a database update, MPD tries to re-register watches

## Kernel Limits

Check current limits:
```bash
cat /proc/sys/fs/inotify/max_user_watches
cat /proc/sys/fs/inotify/max_user_instances
```

Typical defaults:
- `max_user_watches`: 8192 (way too low for large music libraries)
- `max_user_instances`: 128

With 28,386 albums and thousands of directories, this is insufficient.

## Solutions

### Solution 1: Increase Inotify Watch Limits (Recommended)

Add to `/etc/sysctl.conf`:
```
fs.inotify.max_user_watches = 524288
fs.inotify.max_user_instances = 256
```

Apply immediately:
```bash
sudo sysctl fs.inotify.max_user_watches=524288
sudo sysctl fs.inotify.max_user_instances=256
```

### Solution 2: Reduce Backend-Induced Database Updates

The fixes already applied help:
1. **Prevent concurrent refreshes** - Only one refresh runs at a time
2. **Reduce command rate** - 200ms delay between batches
3. **Remove buggy liveness check** - Fewer failed commands that might trigger retries

### Solution 3: Configure MPD to Use Fewer Inotify Watches

In MPD config (`~/.config/mpd/mpd.conf` or `/etc/mpd.conf`):

```conf
# Reduce follow_symlinks usage (each symlinked directory needs a watch)
follow_outside_symlinks "no"
follow_inside_symlinks "no"

# Disable auto-update if not needed
auto_update "no"
```

## Verification

After increasing limits, check:
```bash
# Count watches currently in use
find /proc/*/fd -lname 'anon_inode:inotify' -printf '%hinfo %f\n' 2>/dev/null | xargs -I{} grep -c inotify {}

# Or simpler:
cat /proc/sys/fs/inotify/max_user_watches
```

## Why Previous Analysis Was Partially Wrong

The connection liveness check bug I found IS still a bug and should be fixed, but it's not the main cause of this particular issue. The real culprit is the **inotify watch limit being exceeded**, which causes MPD to spin at 100% CPU.

However, the backend fixes I applied will still help:
- Reducing command rate means fewer potential database updates
- Preventing concurrent refreshes reduces the chance of triggering multiple rescan attempts
- Better connection handling reduces failed commands that might cause issues

## Recommended Actions

1. **Immediate:** Increase inotify limits:
   ```bash
   sudo sysctl fs.inotify.max_user_watches=524288
   ```

2. **Permanent:** Add to `/etc/sysctl.conf`:
   ```
   fs.inotify.max_user_watches = 524288
   fs.inotify.max_user_instances = 256
   ```

3. **Monitor:** After restarting MPD and backend, check CPU usage:
   ```bash
   top -p $(pgrep mpd)
   ```

4. **Optional:** Consider disabling MPD auto-update if not needed:
   ```conf
   auto_update "no"
   ```

## Why 524,288 Watches?

With ~28,000 albums, you likely have:
- ~28,000 album directories
- ~1-3 files per album = ~80,000 files
- Plus subdirectories = potentially 100,000+ watchable items

524,288 provides a comfortable buffer and is commonly used for media server setups.
