# Visibility Management System

## Overview

This document describes the visibility management system implemented for the MPD client modern frontend, with special attention to iOS/iPhone Safari compatibility.

## Problem Statement

When users switch away from the MPD web app and return later, they expect to see current playback status, queue state, and other fresh data. Without proper visibility detection:

- WebSocket connections may be frozen or dropped by the browser
- Users see stale data until the next polling interval (30 seconds)
- iOS Safari aggressively suspends background tabs
- The UI feels "broken" when returning to the app

## Solution Architecture

The visibility management system uses **multiple complementary detection strategies** for maximum reliability across all browsers, especially iOS Safari:

### 1. Page Visibility API (Primary Detection)

**Events**: `visibilitychange`

**Browser Support**: All modern browsers including iOS Safari (since iOS 10)

**Pros**:
- Standard API designed for this purpose
- Distinguishes between visible and hidden states
- Efficient (doesn't fire unnecessarily)

**Cons (iOS Safari)**:
- May not fire reliably when switching between apps
- May not fire when locking/unlocking device
- Requires supplementary detection methods

**Implementation**:
```javascript
document.addEventListener('visibilitychange', () => {
  const isVisible = !document.hidden
  // Trigger refresh when becoming visible
})
```

### 2. Window Focus/Blur Events (Supplementary Detection)

**Events**: `focus`, `blur`

**Browser Support**: All browsers including iOS Safari

**Pros**:
- More reliable for tab switching within Safari
- Good complement to visibility API
- Fires when user returns to tab

**Cons**:
- Doesn't distinguish between visible and hidden
- May fire in situations where page is still visible
- Should not be used alone

**Implementation**:
```javascript
window.addEventListener('focus', () => {
  // Check if visibility API missed this
  if (document.hidden) {
    // Force update visibility state
  }
})
```

### 3. Page Show/Hide Events (iOS BFCache Handling)

**Events**: `pageshow`, `pagehide`

**Browser Support**: iOS Safari uses bfcache (back/forward cache)

**Pros**:
- Detects when page is restored from bfcache (iOS Safari feature)
- `event.persisted` indicates bfcache restoration
- Essential for iOS Safari app switching

**Implementation**:
```javascript
window.addEventListener('pageshow', (event) => {
  if (event.persisted) {
    // Page restored from bfcache
    // Trigger data refresh
  }
})
```

### 4. Router Navigation Guards

**Events**: Vue Router `afterEach`

**Browser Support**: All browsers

**Pros**:
- Detects page navigation within the SPA
- Ensures fresh data when changing views
- WebSocket health check on route changes

**Implementation**:
```javascript
router.afterEach((to, from) => {
  // Verify WebSocket connection
  // Optionally refresh data
})
```

## iOS Safari Specific Behavior

### Background Tab Throttling

iOS Safari aggressively throttles background tabs:

1. **Timers**: `setTimeout`/`setInterval` are throttled to ~1 second minimum
2. **WebSocket**: May be frozen or disconnected entirely
3. **Network Requests**: Queued and delayed significantly
4. **RAF**: `requestAnimationFrame` stops completely

### BFCache (Back/Forward Cache)

iOS Safari caches pages in memory for fast back/forward navigation:

- Page state is preserved (JavaScript state, DOM, etc.)
- **BUT** WebSocket connections may be frozen
- Need to detect `pageshow` with `event.persisted === true`

### App Switching

When user switches to another app and returns:

1. `visibilitychange` may or may not fire (unreliable)
2. `pageshow` with `event.persisted` is more reliable
3. `focus` event provides additional signal

## Implementation Details

### Files Created/Modified

1. **`frontend/src/composables/useVisibilityRefresh.js`** (NEW)
   - Reusable composable for visibility management
   - Integrates all detection strategies
   - Dispatches custom events for app-wide handling

2. **`frontend/src/stores/mpdStore.js`** (MODIFIED)
   - Added `setupVisibilityRefresh()` function
   - Listens for `page-visible` and `route-changed` events
   - Performs aggressive refresh and WebSocket health check

3. **`frontend/src/App.vue`** (MODIFIED)
   - Initializes visibility refresh on mount
   - Cleans up listeners on unmount

### Custom Events

The system dispatches custom events that components can listen to:

#### `page-visible`

**Fired when**: Page becomes visible (tab switch, app return, navigation)

**Usage**:
```javascript
window.addEventListener('page-visible', () => {
  // Refresh data, reconnect WebSockets, etc.
})
```

#### `page-hidden`

**Fired when**: Page becomes hidden (tab switch, app switch)

**Usage**:
```javascript
window.addEventListener('page-hidden', () => {
  // Pause non-essential operations, save state, etc.
})
```

#### `route-changed`

**Fired when**: Router navigation occurs

**Usage**:
```javascript
window.addEventListener('route-changed', (event) => {
  const { to, from } = event.detail
  // Verify connections, refresh data if needed
})
```

## Data Refresh Strategy

### On Page Visible

When the page becomes visible:

1. **WebSocket Health Check**:
   - Check if WebSocket is open (`readyState === OPEN`)
   - Reconnect if closed or connecting

2. **Aggressive Refresh**:
   - Fetch both status and playlist in parallel
   - Update all state immediately
   - Ensures UI shows fresh data

3. **Search WebSocket**:
   - Check and reconnect if needed
   - Ensures search functionality works

### On Route Change

When navigating within the SPA:

1. **Lightweight Check**:
   - Verify WebSocket connection only
   - Don't full refresh (to avoid excessive API calls)

2. **WebSocket Reconnect**:
   - Only if disconnected
   - Prevents stale connections

## Configuration Options

The `useVisibilityRefresh` composable accepts options:

```javascript
useVisibilityRefresh({
  onVisible: () => {},          // Custom callback on visible
  onHidden: () => {},           // Custom callback on hidden
  onRouteChange: (to, from) => {}, // Custom callback on route change
  refreshOnVisible: true,       // Auto-trigger refresh on visible
  refreshOnRouteChange: true,   // Auto-trigger refresh on route change
  debug: false,                 // Enable debug logging
  debounceMs: 500               // Debounce rapid visibility changes
})
```

## Testing

### Desktop Browsers

1. **Chrome/Edge**:
   - Switch to another tab and back
   - Minimize and restore window
   - Verify data refreshes immediately

2. **Firefox**:
   - Same tests as Chrome
   - Test with multiple windows

### iOS Safari (Critical)

1. **Tab Switching**:
   - Open multiple tabs
   - Switch away and back to MPD app
   - Verify `visibilitychange` fires
   - Verify data refreshes

2. **App Switching**:
   - Switch to another app (e.g., Messages)
   - Return to Safari
   - Verify `pageshow` with `event.persisted` fires
   - Verify data refreshes

3. **Device Lock**:
   - Lock device while app is open
   - Unlock and return to app
   - Verify data refreshes

4. **Multiple Background Tabs**:
   - Open multiple tabs
   - Navigate through them
   - Verify each refreshes correctly

### Debug Mode

Enable debug logging to see all visibility events:

```javascript
// In App.vue or component
const { setup } = useVisibilityRefresh({
  debug: true  // Enable console logging
})
```

Debug output shows:
```
[useVisibilityRefresh] Visibility change detected: {nowVisible, wasVisible, timeSinceLastChange}
[useVisibilityRefresh] Page became visible
[useVisibilityRefresh] Window focused
[MPD Store] Page became visible, refreshing data
[MPD Store] Visibility refresh complete
```

## Performance Considerations

### Debouncing

Rapid visibility changes are debounced by default (500ms):

- Prevents excessive refreshes during app switching
- Configurable via `debounceMs` option

### Network Efficiency

- **Page visible**: Full refresh (status + playlist)
- **Route change**: WebSocket check only (no API call unless needed)
- **Parallel requests**: Status and playlist fetched simultaneously

### WebSocket Reconnection

- Only reconnects if needed (`readyState !== OPEN`)
- Prevents unnecessary reconnections
- Maintains connection health

## Browser Compatibility

| Browser | visibilitychange | focus/blur | pageshow | Notes |
|---------|-----------------|------------|----------|-------|
| iOS Safari 10+ | ✅ | ✅ | ✅ | Uses bfcache |
| Chrome 90+ | ✅ | ✅ | ✅ | Full support |
| Firefox 88+ | ✅ | ✅ | ✅ | Full support |
| Safari macOS | ✅ | ✅ | ✅ | Full support |
| Edge 90+ | ✅ | ✅ | ✅ | Full support |

## Future Enhancements

### Potential Improvements

1. **Intelligent Refresh**:
   - Only refresh if data is stale (timestamp check)
   - Skip refresh if WebSocket was recently active

2. **Progressive Enhancement**:
   - Use Page Lifecycle API (`freeze`/`resume`) when available
   - Detect system suspend/resume on mobile devices

3. **User Preferences**:
   - Allow users to disable auto-refresh
   - Configure refresh behavior per view

4. **Analytics**:
   - Track how often visibility changes occur
   - Measure refresh effectiveness

### Network-Aware Refreshing

- Skip refresh on slow networks (2G/3G)
- Queue refresh for when network improves
- Show loading state during refresh

## Troubleshooting

### Data Not Refreshing on iOS Safari

**Symptom**: Stale data when returning to app

**Solutions**:
1. Check debug logs: Enable `debug: true` in `useVisibilityRefresh`
2. Verify events firing: Look for `pageshow` with `event.persisted`
3. Check WebSocket state: Should reconnect if closed
4. Test aggressive refresh: Call `mpdStore.aggressiveRefresh()` manually

### Excessive Refresh Calls

**Symptom**: Too many API calls when navigating

**Solutions**:
1. Increase `debounceMs` to 1000-2000ms
2. Set `refreshOnRouteChange: false` for lighter approach
3. Implement timestamp-based refresh (only if data is old)

### WebSocket Reconnection Issues

**Symptom**: WebSocket doesn't reconnect after tab switch

**Solutions**:
1. Check `ws.value.readyState` in debug logs
2. Verify `connectWebSocket()` is being called
3. Check for browser console errors
4. Test with network throttling enabled

## References

- [Page Visibility API](https://developer.mozilla.org/en-US/docs/Web/API/Page_Visibility_API)
- [Page Lifecycle API](https://developer.mozilla.org/en-US/docs/Web/API/Page_Lifecycle_API)
- [iOS Safari BFCache](https://webkit.org/blog/516/webkit-page-cache-ii-the-bfcache/)
- [WebSocket Best Practices](https://developer.mozilla.org/en-US/docs/Web/API/WebSocket)

## Summary

The visibility management system provides:

✅ **Reliable detection** across all browsers including iOS Safari
✅ **Immediate refresh** when user returns to the app
✅ **WebSocket health checks** and reconnection
✅ **Minimal overhead** with debouncing and efficient refresh
✅ **Developer-friendly** with custom events and debug mode

The multi-layered approach ensures that regardless of how the user interacts with the app (tab switching, app switching, page navigation), the data stays fresh and the WebSocket connections remain healthy.
