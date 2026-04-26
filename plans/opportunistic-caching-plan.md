# Opportunistic Caching Fix Plan

## Problem Summary

The application is experiencing severe MPD server overload due to **aggressive cache building**:

1. **`HandleAllAlbums`** (`/api/albums/all`) calls `EnrichAlbums()` for ALL albums at once
   - This triggers ~50+ MPD `find album` commands simultaneously
   - Each command scans the entire MPD database in alphabetical order
   - Causes 100% CPU usage for hours or days
   - Blocks other operations like `/api/status` and `/api/playlist`

2. **`EnrichAlbums()`** is called from multiple places:
   - `BackgroundEnrichAndCache()` - enriches pages in background
   - `GetRandomAlbums()` - enriches random albums
   - All these use `GetAlbumStats()` which does expensive queries

3. **Current batch size**: 20 albums per batch (recently reduced from unlimited)
   - Still too aggressive for MPD's capabilities

## Current Architecture

### UI Loading Flow
```
AlbumsView.vue
    ├─→ mpdStore.fetchAlbums() (name/date sort)
    └─→ mpdStore.fetchRandomAlbums() (random)
            └─→ Backend: /api/albums or /api/albums/random
```

### Backend Flow
```
/api/albums
    ├─→ HandleAlbumList
    │   ├─→ Returns basic albums (fast, from memory)
    │   └─→ BackgroundEnrichAndCache() [async goroutine]
    │           └─→ EnrichAlbums()
    │               └─→ GetAlbumStats() ❌ EXPENSIVE
    │                   └─→ MPD: find album (50+ commands)
    │
/api/albums/random
    ├─→ HandleRandomAlbums
    │   └─→ GetRandomAlbums(enrich=true)
    │           └─→ EnrichAlbums()
    │               └─→ GetAlbumStats() ❌ EXPENSIVE

/api/albums/all ❌ CULPRIT
    └─→ HandleAllAlbums
        └─→ EnrichAlbums(allAlbums)
            └─→ GetAlbumStats(allKeys) ❌ VERY EXPENSIVE
                └─→ MPD: find album (ALL albums at once)
```

### WebSocket & Cache Invalidation
- ✅ Already implemented: WebSocket sends `database_update` events
- ✅ Already implemented: `albumCache.clear()` on database update
- ✅ Cache has expiration: 15 minutes for pages, 10 minutes for details
- ✅ Connection pool limits: 8 concurrent connections, semaphore with 500ms timeout

## Proposed Solution

### Principle: Opportunistic Caching

> "Load albums only when needed, cache for later reuse, invalidate on database changes"

### Keep These Aggressive Caches

1. **Random Albums Buffer** (`MaintainRandomBuffer()`)
   - Keep 120 pre-enriched random albums in memory
   - Refilled with small batches (5 albums every 200ms)
   - Provides instant loading for "Random" page
   - ✅ KEEP

2. **All Albums List for Matrix** (`/api/albums/all`)
   - Returns basic album info (artist, album, date, genre)
   - NO enrichment (no GetAlbumStats, no track counts, no durations)
   - Used for instant search by artist/genre/date
   - ✅ KEEP BUT REMOVE ENRICHMENT

### Remove These Aggressive Caches

1. **Remove GetAlbumStats() entirely**
   - Albums will have `TrackCount: 0` and `Duration: 0`
   - Stats can be shown in album detail view if needed
   - Never call GetAlbumStats() from EnrichAlbums()

2. **Remove enrichment from HandleAllAlbums**
   - Return only basic album data from cache
   - No background enrichment triggered

3. **Remove enrichment from HandleAlbumList**
   - Return basic albums immediately
   - Remove BackgroundEnrichAndCache() calls

4. **Remove enrichment from HandleRandomAlbums**
   - Use buffer directly (already enriched)
   - No enrichment triggered when requesting random albums

### Implement Opportunistic Caching

When album cards are displayed, lazy-load details:

#### Backend Changes

**New Endpoint: `/api/albums/enrich` (POST)**
```go
// Accepts up to 3 album keys, enriches them, returns result
// Max batch size: 3 albums
```

**New Endpoint: `/api/album/{artist}/{album}/details` (GET)**
```go
// Already exists: HandleAlbumDetails
// Returns tracks for a specific album when viewing album detail page
// Caches for 10 minutes
```

**Remove from EnrichAlbums():**
```go
// OLD CODE:
statsMap, err := client.GetAlbumStats(batchKeys)  // ❌ REMOVE THIS

// NEW CODE:
// Don't call GetAlbumStats at all
// TrackCount and Duration will remain 0
```

#### Frontend Changes

**AlbumsView.vue:**
- Load first 9 albums (one by one, lazy)
- When cards are visible, enrich them in batches of 3

**New Frontend Service: albumEnrichment.js**
```javascript
// Lazy enrichment service
// When album cards appear in viewport, call /api/albums/enrich
// Batch up to 3 albums per request
// Show loading indicator only for cards being enriched
```

## Implementation Plan

### Step 1: Backend - Remove GetAlbumStats
**File**: `backend/internal/albumcache/cache.go`

```go
// In EnrichAlbums(), remove the GetAlbumStats call
// Line 199: statsMap, err := client.GetAlbumStats(batchKeys)  ← DELETE
// Lines 218-220, 232-235: Remove stats application
```

### Step 2: Backend - Remove Aggressive Enrichment
**File**: `backend/internal/api/albums.go`

```go
// HandleAllAlbums: Remove EnrichAlbums() call
// Line 340: enriched, err := cache.EnrichAlbums(allAlbums)  ← DELETE
// Return allAlbums directly

// HandleAlbumList: Remove BackgroundEnrichAndCache() calls
// Lines 67-78: Remove background enrichment goroutines

// HandleRandomAlbums: Use buffer directly
// Line 388: Remove enrich=true parameter
```

### Step 3: Backend - Add Opportunistic Enrichment Endpoint
**File**: `backend/internal/api/albums.go`

```go
// New function: HandleAlbumEnrichOpportunistic
// Accepts: { albums: [{artist, album}, ...] }
// Max: 3 albums per batch
// Enriches using only GetAlbumRepresentatives (no stats)
// Returns: enriched albums
```

### Step 4: Frontend - Add Lazy Enrichment Service
**New File**: `frontend/src/services/albumEnrichment.js`

```javascript
// Service for lazy enrichment of album cards
// Batches up to 3 albums per request
// Uses IntersectionObserver to detect visible cards
```

### Step 5: Frontend - Update AlbumsView
**File**: `frontend/src/views/AlbumsView.vue`

```javascript
// Load albums without enrichment (basic info only)
// Use IntersectionObserver to trigger enrichment when visible
// Load first 9 cards one-by-one (lazy mode)
// Each enrichment call batches up to 3 albums
```

### Step 6: Cache Invalidation
**Already implemented** ✅

- WebSocket `database_update` event
- `albumCache.clear()` on database update
- Custom event `database-updated` for UI listeners

## Benefits

1. **MPD Server Health**
   - No more 100% CPU for hours
   - Status and playlist queries work reliably
   - Responsive UI

2. **Better User Experience**
   - Initial album lists load instantly (basic info)
   - Details appear progressively as cards become visible
   - No long loading screens

3. **Scalability**
   - Works with any library size
   - Batch size limited to 3 (safe for MPD)
   - Connection pool ensures fair resource usage

## Testing Checklist

- [ ] Startup: MPD CPU stays below 50%
- [ ] `/api/status` responds in <100ms
- [ ] `/api/playlist` responds in <100ms
- [ ] `/api/albums/all` responds in <200ms
- [ ] Album cards show basic info immediately
- [ ] Album details load progressively as cards scroll into view
- [ ] Database update clears cache and triggers re-enrichment
- [ ] Random albums page loads instantly from buffer
- [ ] Search by artist/genre/date works instantly

## Files to Modify

### Backend
1. `backend/internal/albumcache/cache.go` - Remove GetAlbumStats from EnrichAlbums
2. `backend/internal/api/albums.go` - Remove aggressive enrichment calls

### Frontend
1. `frontend/src/services/albumEnrichment.js` - New lazy enrichment service
2. `frontend/src/views/AlbumsView.vue` - Use lazy enrichment
3. Optional: Other views that display albums

### Optional Cleanup
1. `backend/internal/mpd/client.go` - Can remove GetAlbumStats entirely
2. Backend internal - Remove any unused stats aggregation
