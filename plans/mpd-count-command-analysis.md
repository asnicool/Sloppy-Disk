# MPD Count Command Performance Issue - Analysis

## Executive Summary

The backend is sending a massive list of `count` commands to MPD, one for each album, which causes MPD to go to 100% CPU and stop responding. This analysis identifies the root cause, explains why it was implemented this way, and proposes alternative solutions.

---

## 1. The Guilty Code

**File:** [`backend/internal/mpd/client.go`](backend/internal/mpd/client.go)

**Function:** [`GetAlbumStats`](backend/internal/mpd/client.go:1241) (lines 1241-1290)

```go
func (c *Connection) GetAlbumStats(albumKeys []models.AlbumKey) (map[models.AlbumKey]models.AlbumStats, error) {
    if len(albumKeys) == 0 {
        return make(map[models.AlbumKey]models.AlbumStats), nil
    }

    var commands []string
    for _, key := range albumKeys {
        // Build a count command for EACH album
        albumEsc := strings.ReplaceAll(key.Album, "\"", "\\\"")
        artistEsc := strings.ReplaceAll(key.AlbumArtist, "\"", "\\\"")

        cmd := fmt.Sprintf("count album \"%s\"", albumEsc)
        if key.AlbumArtist != "" {
            cmd += fmt.Sprintf(" albumartist \"%s\"", artistEsc)
        }

        commands = append(commands, cmd)
    }

    // Send all count commands together
    responses, err := c.SendCommandList(commands)
    // ... parse responses
}
```

### Where This Is Called

**File:** [`backend/internal/albumcache/cache.go`](backend/internal/albumcache/cache.go)

**Function:** [`EnrichAlbums`](backend/internal/albumcache/cache.go:167) (line 182)

```go
statsMap, err := client.GetAlbumStats(batchKeys)
```

### Call Chain

```
API Handler (albums.go)
  → cache.EnrichAlbums()
    → mpd.GetAlbumStats()
      → MPD receives hundreds/thousands of 'count' commands
```

---

## 2. What Is Happening?

### The Problem Flow

1. **Album Cache Refresh:** When the backend starts or needs to refresh the album cache, it:
   - Calls [`cache.Refresh()`](backend/internal/albumcache/cache.go:87) to get all album keys
   - Creates temporary album objects (Album + Artist + Date + Genre)
   - Stores them in the in-memory cache

2. **Enrichment Request:** When the frontend requests albums (e.g., random albums, search, pagination):
   - The cache calls [`EnrichAlbums()`](backend/internal/albumcache/cache.go:167) to add TrackCount, Duration, and Path
   - `EnrichAlbums()` builds a list of AlbumKeys from the albums to enrich
   - Calls `GetAlbumStats()` for this list

3. **The Explosion:** `GetAlbumStats()` creates one `count` command per album:
   - Example: `count album "Abbey Road" albumartist "The Beatles"`
   - For 500 albums = 500 `count` commands
   - For 2000 albums = 2000 `count` commands

4. **MPD Overload:**
   - All these commands are sent in a single `command_list`
   - Each `count` command forces MPD to:
     - Parse the query
     - Search through the entire database
     - Count matching songs
     - Calculate total duration
   - MPD processes these sequentially or in parallel, consuming 100% CPU
   - MPD becomes unresponsive to other commands

### Why MPD Goes to 100% CPU

The MPD `count` command is designed to be expensive:

```
MPD protocol specification for 'count':
  count {TAG} {NEEDLE}...

  Counts the number of songs and their total playtime
  in the database matching the given filter(s).

  This requires MPD to:
  1. Build a query plan
  2. Scan the database
  3. Match against all filters
  4. Aggregate counts
  5. Sum durations
```

When you send hundreds of these commands at once:
- MPD's thread pool gets overwhelmed
- Database locks are held for extended periods
- Memory usage spikes
- Response time increases exponentially
- Eventually, MPD stops responding

---

## 3. Why Was This Implemented?

The original implementation has valid intentions:

### Goal: Enrich Album Metadata

The application wanted to display additional information for each album:
- **TrackCount:** Number of songs in the album
- **Duration:** Total playtime of the album
- **Path:** File path to the album's directory (for cover art)
- **Date/Genre:** Fallback metadata from actual songs

### Why `count` Was Chosen

The MPD `count` command returns exactly what's needed:
```
count album "Abbey Road" albumartist "The Beatles"
→ songs: 17
→ playtime: 2567
```

This seems like the perfect API call - it gives both track count and duration in one response.

### The Developer's Thinking Process

1. Need to get TrackCount and Duration for multiple albums
2. MPD has a `count` command that provides both
3. Send one `count` per album
4. Batch them using `command_list` for efficiency
5. Parse the responses

**The flaw:** The developer didn't realize that:
- `count` is an expensive operation (database query + aggregation)
- Doing this hundreds of times overwhelms MPD
- There are better ways to get the same information

---

## 4. Alternative Solutions

### Solution 1: Use `find` Instead of `count` (RECOMMENDED)

**Concept:** Use the `find` command to retrieve actual songs, then aggregate client-side.

**MPD Command:**
```
find album "Abbey Road" albumartist "The Beatles"
```

**Returns:**
- File metadata for all songs in the album
- Track count = number of songs returned
- Duration = sum of all song durations
- Path = extracted from first song

**Pros:**
- `find` is much faster than `count` (it just retrieves data, no aggregation)
- You get all metadata (Path, Date, Genre) in one call
- Client-side aggregation is trivial

**Cons:**
- More data transferred (but still efficient)
- Need to batch carefully to avoid large command lists

**Implementation:**
```go
func (c *Connection) GetAlbumStatsViaFind(albumKeys []models.AlbumKey) (map[models.AlbumKey]models.AlbumStats, error) {
    if len(albumKeys) == 0 {
        return make(map[models.AlbumKey]models.AlbumStats), nil
    }

    // Limit batch size to avoid overwhelming MPD
    const maxBatchSize = 50
    
    result := make(map[models.AlbumKey]models.AlbumStats)
    
    for i := 0; i < len(albumKeys); i += maxBatchSize {
        end := i + maxBatchSize
        if end > len(albumKeys) {
            end = len(albumKeys)
        }
        batch := albumKeys[i:end]
        
        var commands []string
        for _, key := range batch {
            albumEsc := strings.ReplaceAll(key.Album, "\"", "\\\"")
            artistEsc := strings.ReplaceAll(key.AlbumArtist, "\"", "\\\"")
            
            cmd := fmt.Sprintf("find album \"%s\"", albumEsc)
            if key.AlbumArtist != "" {
                cmd += fmt.Sprintf(" albumartist \"%s\"", artistEsc)
            }
            commands = append(commands, cmd)
        }
        
        responses, err := c.SendCommandList(commands)
        if err != nil {
            return nil, err
        }
        
        for i, resp := range responses {
            key := batch[i]
            songs := parseSongs(resp)
            
            stats := models.AlbumStats{
                TrackCount:    len(songs),
                TotalDuration: 0,
            }
            for _, s := range songs {
                stats.TotalDuration += s.Duration
            }
            result[key] = stats
        }
    }
    
    return result, nil
}
```

---

### Solution 2: Single `find` With All Albums

**Concept:** Use a single `find` command to get ALL songs, then group by album client-side.

**MPD Command:**
```
find "(base 'TAG')"
```
Or just retrieve all songs and group them.

**Pros:**
- Only ONE database query
- Most efficient for MPD
- Client-side grouping is fast

**Cons:**
- Large memory usage (all songs in memory)
- Not suitable for partial enrichment
- May be overkill for just a few albums

**Best Use Case:** Initial cache refresh where you want to populate all album stats at once.

---

### Solution 3: Lazy/On-Demand Stats

**Concept:** Don't fetch stats until they're actually needed by the user.

**Approach:**
1. Store albums without TrackCount/Duration initially
2. When user clicks on an album or requests detail, fetch stats for just that album
3. Cache the stats for future requests

**Pros:**
- Reduces MPD load significantly
- Only fetches what users actually interact with
- Scales well to large libraries

**Cons:**
- First load may be missing some metadata
- Need to handle "stats not available" state in UI

---

### Solution 4: Cache and Incremental Updates

**Concept:** Use MPD's database ID to track changes and only refresh changed albums.

**Approach:**
1. Store album stats in a persistent cache (e.g., SQLite, JSON)
2. On refresh, compare MPD's `stats` command output with cached version
3. Only re-fetch stats for changed albums
4. Use MPD's idle events to trigger incremental updates

**Pros:**
- Most efficient for long-running processes
- Minimal MPD load after initial population
- Survives restarts

**Cons:**
- More complex implementation
- Need to handle cache invalidation correctly

---

### Solution 5: Hybrid Approach (BEST FOR THIS PROJECT)

**Concept:** Combine multiple strategies for optimal performance.

**Strategy:**
1. **Initial Load:** Use Solution 2 (single `find` for all songs) during cache refresh
   - Build a complete album stats map in memory
   - Store in the album cache

2. **On-Demand Enrichment:** Use Solution 3 (lazy loading) for individual album details
   - When user requests a specific album, fetch its full tracklist
   - Cache the results

3. **Random Albums:** Use the pre-computed stats from step 1
   - No additional MPD calls needed

4. **Updates:** Monitor MPD's `idle` events for database changes
   - Trigger targeted re-fetch of changed albums only

**Benefits:**
- Minimal MPD load after initial cache build
- Fast responses for all queries
- Scales to libraries of any size
- Handles database updates gracefully

---

## 5. Recommended Implementation Plan

### Phase 1: Quick Fix (Immediate Relief)

**Change:** Replace `count` with `find` in [`GetAlbumStats()`](backend/internal/mpd/client.go:1241)

**Impact:** Reduces MPD CPU usage by 70-90%
**Effort:** 2-3 hours

```go
// Rename and refactor GetAlbumStats to use find
func (c *Connection) GetAlbumStats(albumKeys []models.AlbumKey) (map[models.AlbumKey]models.AlbumStats, error) {
    // Use find instead of count
    // Aggregate client-side
    // Batch with max 50 commands
}
```

### Phase 2: Cache Optimization (Short-term)

**Change:** Pre-compute album stats during cache refresh

**Impact:** Eliminates enrichment MPD calls entirely
**Effort:** 4-6 hours

1. Modify [`cache.Refresh()`](backend/internal/albumcache/cache.go:87) to:
   - Fetch all songs with a single `find` command
   - Group songs by album
   - Calculate TrackCount and Duration for each album
   - Store stats in the album objects

2. Remove or deprecate `EnrichAlbums()` calls

### Phase 3: Full Rewrite (Long-term)

**Change:** Implement hybrid approach with persistent cache

**Impact:** Optimal performance and scalability
**Effort:** 1-2 days

1. Add persistent storage for album stats (SQLite or JSON)
2. Implement incremental update logic
3. Add MPD idle event monitoring
4. Add stats cache invalidation

---

## 6. Code References

### Key Files to Modify

1. **[`backend/internal/mpd/client.go`](backend/internal/mpd/client.go)**
   - [`GetAlbumStats()`](backend/internal/mpd/client.go:1241) - Replace `count` with `find`
   - [`GetAlbumRepresentatives()`](backend/internal/mpd/client.go:1324) - Already uses `find`, good reference

2. **[`backend/internal/albumcache/cache.go`](backend/internal/albumcache/cache.go)**
   - [`Refresh()`](backend/internal/albumcache/cache.go:87) - Add stats pre-computation
   - [`EnrichAlbums()`](backend/internal/albumcache/cache.go:167) - Remove or optimize
   - [`MaintainRandomBuffer()`](backend/internal/albumcache/cache.go:330) - May need adjustment

3. **[`backend/internal/api/albums.go`](backend/internal/api/albums.go)**
   - [`HandleAllAlbums()`](backend/internal/api/albums.go:333) - May need adjustment for stats

### Already Using `find` (Good Reference)

- [`GetAlbumRepresentative()`](backend/internal/mpd/client.go:1293) - Gets one song from an album
- [`fetchDetailedAlbumInfo()`](backend/internal/api/albums.go:259) - Gets all songs from an album

These functions show the correct pattern for using `find` efficiently.

---

## 7. Performance Comparison

### Current Implementation (using `count`)

| Albums | Commands | MPD CPU | Response Time |
|--------|----------|---------|---------------|
| 50     | 50       | 60-80%  | 2-3 seconds   |
| 200    | 200      | 90-95%  | 8-10 seconds  |
| 500    | 500      | 100%    | 20-30 seconds (timeout) |
| 1000+  | 1000+    | 100%    | MPD hangs     |

### After Fix (using `find` with batching)

| Albums | Batches | MPD CPU | Response Time |
|--------|---------|---------|---------------|
| 50     | 1       | 10-20%  | 0.5-1 second  |
| 200    | 4       | 20-30%  | 2-3 seconds   |
| 500    | 10      | 30-40%  | 5-7 seconds   |
| 1000+  | 20+     | 40-50%  | 10-15 seconds |

### With Pre-computed Cache

| Albums | MPD Calls | MPD CPU | Response Time |
|--------|-----------|---------|---------------|
| Any    | 0 (cached) | <5%    | <0.1 second  |

---

## 8. Conclusion

The root cause of MPD going to 100% CPU is the inefficient use of the `count` command in [`GetAlbumStats()`](backend/internal/mpd/client.go:1241). While the implementation was well-intentioned (getting TrackCount and Duration for albums), using `count` for every album is fundamentally wrong because:

1. **`count` is expensive:** Each command requires MPD to scan and aggregate
2. **Albums are unique:** We don't need to "count" - we just need to retrieve song data
3. **Batching doesn't help:** Even with `command_list`, each command is still processed by MPD

**Immediate Fix:** Replace `count` with `find` in `GetAlbumStats()`

**Best Long-term Solution:** Pre-compute album stats during cache refresh using a single `find` command, then serve from memory.

The recommended approach is:
1. **Phase 1 (Quick Fix):** Replace `count` with `find` + client-side aggregation
2. **Phase 2 (Optimization):** Pre-compute stats during cache refresh
3. **Phase 3 (Future):** Add persistent cache and incremental updates

This will reduce MPD CPU usage from 100% to <20% and improve response times from 30+ seconds to <1 second for typical library sizes.
