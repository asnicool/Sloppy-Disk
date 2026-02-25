# Visibility Management - Quick Start Guide

## TL;DR

The app now automatically refreshes data when users return to it, with special optimizations for iOS Safari.

## What Changed

### Before
- ❌ No visibility detection
- ❌ Stale data until 30-second poll or WebSocket update
- ❌ Broken WebSocket after backgrounding on iOS
- ❌ Poor experience when switching back to the app

### After
- ✅ Multi-layered visibility detection (visibilitychange + focus + pageshow)
- ✅ Immediate refresh when returning to the app
- ✅ WebSocket health check and auto-reconnection
- ✅ iOS Safari optimized with bfcache handling
- ✅ Smooth experience across all browsers

## Event Flow Diagram

```
User switches away from app
         ↓
   [page-hidden event]
         ↓
   App is in background
   (WebSocket may be frozen)
         ↓
User returns to app
         ↓
┌────────────────────────────────┐
│  Multiple Detection Methods    │
├────────────────────────────────┤
│ visibilitychange ← Primary     │
│ focus/blur      ← Supplement   │
│ pageshow        ← iOS bfcache  │
│ route-changed   ← Navigation   │
└────────────────────────────────┘
         ↓
   [page-visible event]
         ↓
┌────────────────────────────────┐
│  Refresh Actions               │
├────────────────────────────────┤
│ 1. Check WebSocket health      │
│ 2. Reconnect if needed         │
│ 3. Fetch fresh status          │
│ 4. Fetch fresh playlist        │
│ 5. Update UI immediately       │
└────────────────────────────────┘
         ↓
   User sees current data ✅
```

## Code Integration Points

### 1. Composable: `useVisibilityRefresh.js` (NEW)

**Purpose**: Reusable visibility detection logic

**Key Features**:
- Listens to all visibility-related events
- Dispatches custom events (`page-visible`, `page-hidden`)
- Configurable options (debug, debounce, callbacks)
- Automatic cleanup

**Usage**:
```javascript
import { useVisibilityRefresh } from '@/composables/useVisibilityRefresh'

const { setup, cleanup, triggerRefresh } = useVisibilityRefresh({
  debug: true,        // Enable logging
  debounceMs: 500     // Debounce rapid changes
})

// Call in onMounted
setup()

// Call in onUnmounted
cleanup()

// Manual refresh trigger
triggerRefresh()
```

### 2. Store: `mpdStore.js` (MODIFIED)

**Added**: `setupVisibilityRefresh()` function

**What it does**:
- Listens for `page-visible` event
- Checks WebSocket health
- Reconnects if needed
- Performs aggressive refresh (status + playlist)
- Listens for `route-changed` event
- Verifies WebSocket on navigation

**Called from**: `connect()` function (automatic)

### 3. App: `App.vue` (MODIFIED)

**Added**: Visibility refresh setup in `onMounted`

**What it does**:
- Initializes visibility listeners
- Enables debug mode in development
- Cleans up on unmount

## Custom Events

Components can listen to these events:

### `page-visible`
Fired when the page becomes visible (tab switch, app return)

```javascript
window.addEventListener('page-visible', () => {
  console.log('Page is now visible!')
  // Do something custom
})
```

### `page-hidden`
Fired when the page becomes hidden

```javascript
window.addEventListener('page-hidden', () => {
  console.log('Page is now hidden!')
  // Pause operations, save state, etc.
})
```

### `route-changed`
Fired when navigating within the app

```javascript
window.addEventListener('route-changed', (event) => {
  const { to, from } = event.detail
  console.log(`Navigated from ${from.path} to ${to.path}`)
})
```

## Testing Checklist

### Desktop (Chrome/Firefox/Safari)
- [ ] Switch to another tab → switch back → data refreshes
- [ ] Minimize window → restore → data refreshes
- [ ] Navigate between pages → WebSocket stays connected
- [ ] Check console logs with debug mode

### iOS Safari (Critical)
- [ ] Tab switching: Open multiple tabs, switch away and back → verify refresh
- [ ] App switching: Switch to another app, return → verify refresh
- [ ] Device lock: Lock device, unlock → verify refresh
- [ ] BFCache: Navigate away and back → verify `pageshow` fires

### Android Chrome
- [ ] Same tests as iOS Safari
- [ ] Verify background tab behavior

## Debug Mode

Enable debug logging to see all events:

```javascript
// In App.vue
const { setup } = useVisibilityRefresh({
  debug: true  // See all visibility events in console
})
```

**Console output example**:
```
[useVisibilityRefresh] Visibility change detected: {nowVisible: true, ...}
[useVisibilityRefresh] Page became visible
[MPD Store] Page became visible, refreshing data
[MPD Store] Aggressive refresh complete
```

## Performance Impact

- **Minimal overhead**: Only events, no polling
- **Debounced**: 500ms default prevents rapid refreshes
- **Efficient refresh**: Parallel status + playlist fetch
- **Smart reconnection**: Only reconnects if WebSocket is closed

## Browser Support

| Browser | Support | Notes |
|---------|---------|-------|
| iOS Safari 10+ | ✅ Full | BFCache handling |
| Chrome 90+ | ✅ Full | All features |
| Firefox 88+ | ✅ Full | All features |
| Safari macOS | ✅ Full | All features |
| Edge 90+ | ✅ Full | All features |

## Troubleshooting

**Problem**: Data not refreshing when returning to app

**Solutions**:
1. Enable debug mode and check console logs
2. Verify `page-visible` event is firing
3. Check WebSocket state (should reconnect)
4. Test `aggressiveRefresh()` manually from console

**Problem**: Too many refresh requests

**Solutions**:
1. Increase `debounceMs` to 1000-2000ms
2. Set `refreshOnRouteChange: false` for lighter approach

**Problem**: WebSocket doesn't reconnect

**Solutions**:
1. Check browser console for errors
2. Verify `connectWebSocket()` is being called
3. Test with network throttling

## Files Modified/Created

```
frontend/src/
├── composables/
│   └── useVisibilityRefresh.js    [NEW] - Visibility detection logic
├── stores/
│   └── mpdStore.js                [MODIFIED] - Added visibility refresh
├── App.vue                        [MODIFIED] - Initialize visibility
└── docs/
    ├── VISIBILITY_MANAGEMENT.md   [NEW] - Full documentation
    └── VISIBILITY_QUICK_START.md  [NEW] - This file
```

## Summary

The visibility management system ensures users always see fresh data when returning to the MPD web app, with special handling for iOS Safari's aggressive background tab throttling and bfcache behavior.

**Key Benefits**:
- ✅ Immediate data refresh on app return
- ✅ WebSocket health monitoring
- ✅ iOS Safari optimized
- ✅ Minimal performance overhead
- ✅ Developer-friendly with debug mode

**Result**: Users no longer see stale data after switching away from the app!
