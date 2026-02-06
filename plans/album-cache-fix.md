# Album Cache Bug Fix - Albums/Artists Mixed Up

## Problem Description

Albums and artists are being incorrectly associated in the backend cache. Albums from a given artist are appearing as if they belong to a different artist.

## Root Cause Analysis

### Location
- **File**: [`backend/internal/mpd/client.go`](backend/internal/mpd/client.go:1146-1250)
- **Function**: [`GetAllAlbumKeys()`](backend/internal/mpd/client.go:1146-1250)

### The Bug

The original MPD command was:
```go
cmd := "list genre group albumartist group date group album"
```

The original parsing logic:
```go
var currentGenre, currentArtist, currentDate string

for _, line := range lines {
    // ... parse line ...
    switch key {
    case "Genre":
        currentGenre = value
    case "AlbumArtist":
        currentArtist = value
    case "Date":
        currentDate = value
    case "Album":
        keys = append(keys, models.AlbumKey{
            Album:       value,
            AlbumArtist: currentArtist,
            Date:        currentDate,
            Genre:       currentGenre,
        })
    }
}
```

### Why This Causes Data Corruption

1. **Stateful Parsing Issue**: The parser uses cached state variables (`currentGenre`, `currentArtist`, `currentDate`) that persist until explicitly overwritten by a new tag value.

2. **Group Order Problem**: The command groups by `genre` first, then `albumartist`, then `date`. When the genre changes:
   - `currentGenre` is updated (correctly)
   - BUT `currentArtist` and `currentDate` **retain their old values** from the previous genre group

3. **Example Scenario**:
```
Genre: Jazz
AlbumArtist: Miles Davis
Date: 1959
Album: Kind of Blue          ← Correct: Miles Davis, 1959, Jazz

Genre: Rock                   ← Genre changes
Album: Led Zeppelin IV       ← WRONG: Uses old Artist=Miles Davis, Date=1959!
```

4. **Date Group Issue**: Looking at MPD raw output, `Date` is the innermost group that changes for each album. When we see a new Date:
   - The album's Artist/AlbumArtist tag should be reset
   - Otherwise albums without explicit artist tags inherit from previous album

## The Fix

### Correct MPD Command

Changed from:
```go
cmd := "list genre group albumartist group date group album"
```

To:
```go
cmd := "list album group date group albumartist group artist group genre"
```

This matches the actual MPD output structure where:
- `album` is the item being listed
- `date`, `albumartist`, `artist` are grouped fields
- `genre` is the outermost group (reset when changed)

### Update Parsing Logic

```go
var currentGenre, currentArtist, currentDate string

for _, line := range lines {
    if line == "" {
        continue
    }
    parts := strings.SplitN(line, ": ", 2)
    if len(parts) != 2 {
        continue
    }

    key, value := parts[0], parts[1]

    switch key {
    case "Genre":
        // Genre is the outermost group - reset all inner state when it changes to non-empty value
        if currentGenre != "" && currentGenre != value {
            currentArtist = ""
            currentDate = ""
        }
        currentGenre = value
    case "AlbumArtist":
        currentArtist = value
    case "Artist":
        // Use Artist only if AlbumArtist hasn't been set (fallback)
        if currentArtist == "" {
            currentArtist = value
        }
    case "Date":
        // Date is the innermost group - reset artist when it changes
        currentArtist = ""
        currentDate = value
    case "Album":
        // When we see an Album, emit a complete album key with all context
        // But only emit if Album name is not empty
        if value != "" {
            keys = append(keys, models.AlbumKey{
                Album:       value,
                AlbumArtist: currentArtist,
                Date:        currentDate,
                Genre:       currentGenre,
            })
        }
    }
}
```

### Why This Works

1. **Genre change (outermost group)**: When Genre changes to a non-empty value, both `currentArtist` and `currentDate` are reset. This prevents albums in a new genre from inheriting artist/date values from the previous genre group.

2. **Date change (innermost group)**: When Date changes for a new album, `currentArtist` is reset. This prevents albums without explicit Artist/AlbumArtist tags from inheriting artist from the previous album.

3. **Artist fallback**: By including "artist" in the command, if a song has no AlbumArtist tag but has an Artist tag, the Artist value will be used. The AlbumArtist value takes precedence if present.

4. **Skip empty Album names**: Albums with empty names are skipped to prevent invalid entries.

## Files Modified

1. [`backend/internal/mpd/client.go`](backend/internal/mpd/client.go:1161-1250) - Updated `GetAllAlbumKeys()` function

## Changes Made

### Line 1161: Corrected MPD command
```diff
- cmd := "list genre group albumartist group date group album"
+ cmd := "list album group date group albumartist group artist group genre"
```

### Lines 1217-1250: Complete parsing fix
- Line 1217-1223: Genre state reset with non-empty check
- Lines 1224-1230: Artist fallback case (empty check)
- Lines 1231-1234: Date artist reset
- Line 1237: Skip empty Album entries

## Testing

### Test Endpoint
```bash
curl http://localhost:8080/api/albums/all | jq '.data[] | {album, artist, date, genre}'
```

### Verify Results
Check that albums are correctly associated with their artists:
- Albums from different genres have correct artist associations
- Albums with same name but different artists are distinguished
- Date is correctly populated for each album
- Genre is correctly populated for each album
- Artist fallback works when AlbumArtist is missing
- AlbumArtist takes precedence when both are present
- Empty Album names are skipped

### Compare Before/After
```bash
# Before restart (using old cached data)
curl http://localhost:8080/api/albums/all | jq '.data[] | {album, artist, date, genre}' > before.json

# Restart backend server to trigger cache refresh

# After restart (newly built cache)
curl http://localhost:8080/api/albums/all | jq '.data[] | {album, artist, date, genre}' > after.json

# Compare
diff before.json after.json
```
