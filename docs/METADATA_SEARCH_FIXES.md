# Metadata Search Fixes Applied

## Summary

Fixed two critical issues in the metadata search system and added comprehensive logging throughout.

## Issues Fixed

### 1. MusicBrainz 404 Error - **RESOLVED**

**Problem:**
- The `Search` function returns a **release-group ID** (e.g., "12345678-1234-1234-1234-123456789012")
- The `GetReleaseDetails` function tried to use this ID with the `/release/` endpoint
- However, `/release/` expects a **release ID**, not a release-group ID
- These are different entities in MusicBrainz's data model:
  - **Release Group**: Represents a group of releases (different editions, regions, formats)
  - **Release**: Represents a specific edition of an album

**Example:**
```
Search returns: externalID = "release-group:12345678-..."
GetReleaseDetails tried: GET /release/12345678-...
Result: 404 Not Found ✗
```

**Solution:**
Modified `GetReleaseDetails` to:
1. Accept the release-group ID
2. Fetch the release-group to get a list of associated releases
3. Use the first release ID to fetch full release details
4. Return the complete metadata with track listings

**New Flow:**
```
Search returns: externalID = "release-group:12345678-..."
GetReleaseDetails:
  1. GET /release-group/12345678-... → Get releases list
  2. Extract first release ID: "87654321-..."
  3. GET /release/87654321-... → Get full details with tracks ✓
```

**Files Changed:**
- `backend/internal/metadata/musicbrainz.go`

### 2. Missing Logs for /api/metadata/details - **RESOLVED**

**Problem:**
The `/api/metadata/details` endpoint had no logging, making it impossible to debug when:
- The endpoint was called
- Which source/externalID was being requested
- Whether the request succeeded or failed

**Solution:**
Added comprehensive logging to the `GetMetadataDetails` handler:

```go
[METADATA DETAILS] GetMetadataDetails called
[METADATA DETAILS]   Source: 'MusicBrainz'
[METADATA DETAILS]   ExternalID: 'release-group:12345678-...'
[METADATA DETAILS] Creating aggregator and fetching details...
[METADATA DETAILS] Successfully fetched details for 'Pink Floyd - The Dark Side of the Moon' (1973)
[METADATA DETAILS]   Tracks: 10
```

**Files Changed:**
- `backend/internal/api/handlers.go`

## Enhanced Logging

### MusicBrainz Provider (`musicbrainz.go`)

Added detailed logging at every step:

**Search Function:**
- Input parameters (artist, album)
- Rate limiting delays
- Query construction
- HTTP request details
- Response status
- JSON parsing
- Release groups returned with full details
- Candidate creation

**GetReleaseDetails Function:**
- External ID received
- Rate limiting
- Release-group fetch to get release ID
- Release details fetch
- Track count and metadata
- Any errors at each step

### Metadata Handler (`handlers.go`)

**SearchMetadata:**
- Request parameters
- Provider selection
- Search initiation
- Results returned

**GetMetadataDetails:**
- Request parameters
- Fetch process
- Success/failure
- Results summary (artist, album, year, track count)

### Aggregator (`aggregator.go`)

- Provider availability
- Active providers after filtering
- Parallel search execution
- Per-provider results with confidence scores
- Deduplication process
- Final sorted results

## How to Use the Logs

### Monitor Real-Time
```bash
tail -f backend/server_output.log
```

### Filter by Category
```bash
# All metadata logs
grep "\[METADATA" backend/server_output.log

# Only MusicBrainz
grep "\[MUSICBRAINZ" backend/server_output.log

# Only metadata handlers
grep "\[METADATA HANDLER" backend/server_output.log
grep "\[METADATA DETAILS" backend/server_output.log

# Only aggregator
grep "\[METADATA SEARCH" backend/server_output.log
grep "\[CONFIDENCE" backend/server_output.log
```

### Debug a Specific Search

1. Find the search request:
```bash
grep "SearchMetadata called" backend/server_output.log
```

2. Follow the entire flow:
```bash
grep -A 50 "SearchMetadata called" backend/server_output.log | head -60
```

3. Check MusicBrainz specifically:
```bash
grep -A 10 "MUSICBRAINZ.*Search called" backend/server_output.log
```

4. Check details fetch:
```bash
grep -A 15 "METADATA DETAILS.*GetMetadataDetails called" backend/server_output.log
```

## Testing the Fix

### Test Search

```bash
curl "http://localhost:7070/api/metadata/search?artist=Pink%20Floyd&album=Dark%20Side%20of%20the%20Moon"
```

Expected logs:
```
[METADATA HANDLER] SearchMetadata called
[METADATA HANDLER]   Artist: 'Pink Floyd'
[METADATA HANDLER]   Album: 'Dark Side of the Moon'
[METADATA SEARCH] Starting search - Artist: 'Pink Floyd', Album: 'Dark Side of the Moon'
[METADATA SEARCH] Available providers: 1
[MUSICBRAINZ] Search called with artist='Pink Floyd', album='Dark Side of the Moon'
[MUSICBRAINZ] API returned X release groups (count=X)
[METADATA SEARCH] Final results (sorted by confidence):
```

### Test Details Fetch

First, get a search result and extract the `externalID` and `source`:
```bash
curl "http://localhost:7070/api/metadata/search?artist=The%20Beatles&album=Abbey%20Road"
```

Then use the external ID:
```bash
curl "http://localhost:7070/api/metadata/details?source=MusicBrainz&externalId=YOUR_EXTERNAL_ID"
```

Expected logs:
```
[METADATA DETAILS] GetMetadataDetails called
[METADATA DETAILS]   Source: 'MusicBrainz'
[METADATA DETAILS]   ExternalID: 'release-group:12345678-...'
[MUSICBRAINZ] GetReleaseDetails called with externalID='release-group:12345678-...'
[MUSICBRAINZ] Fetching release-group to get release ID: '12345678-...'
[MUSICBRAINZ] Found release ID: '87654321-...', now fetching release details
[MUSICBRAINZ] Release details: Title='Abbey Road', Date='1969-09-26', Tracks=17
[METADATA DETAILS] Successfully fetched details for 'The Beatles - Abbey Road' (1969)
[METADATA DETAILS]   Tracks: 17
```

## What Changed in the Fix

### Before (Broken)
```go
func (p *MusicBrainzProvider) GetReleaseDetails(externalID string) (*models.MetadataCandidate, error) {
    // Remove "release-group:" prefix if present
    id := strings.TrimPrefix(externalID, "release-group:")
    
    // Try to fetch release directly - THIS FAILS with 404
    req, err := p.newRequest("GET", "/release/"+id, params)
    // ...
}
```

### After (Fixed)
```go
func (p *MusicBrainzProvider) GetReleaseDetails(externalID string) (*models.MetadataCandidate, error) {
    log.Printf("[MUSICBRAINZ] GetReleaseDetails called with externalID='%s'", externalID)
    
    // 1. Fetch release-group to get release ID
    log.Printf("[MUSICBRAINZ] Fetching release-group to get release ID: '%s'", externalID)
    req, err := p.newRequest("GET", "/release-group/"+externalID, params)
    // ... fetch and extract releaseID from releases list ...
    
    // 2. Fetch actual release details
    log.Printf("[MUSICBRAINZ] Found release ID: '%s', now fetching release details", releaseID)
    req, err = p.newRequest("GET", "/release/"+releaseID, params)
    // ... fetch and return complete details with tracks ...
}
```

## Verification

To verify the fix works:

1. **Search for a well-known album:**
   ```bash
   curl "http://localhost:7070/api/metadata/search?artist=Pink%20Floyd&album=The%20Dark%20Side%20of%20the%20Moon"
   ```

2. **Check logs:**
   ```bash
   tail -50 backend/server_output.log | grep MUSICBRAINZ
   ```
   
   Should see:
   - API returning release groups (count > 0)
   - Candidates being created

3. **Fetch details:**
   ```bash
   # Use externalID from search results
   curl "http://localhost:7070/api/metadata/details?source=MusicBrainz&externalId=YOUR_ID"
   ```

4. **Check logs:**
   ```bash
   tail -50 backend/server_output.log | grep MUSICBRAINZ
   ```
   
   Should see:
   - Release-group fetch succeeded
   - Release ID extracted
   - Release details fetched with track count

## Summary

The metadata search system now:
- ✅ Properly handles MusicBrainz's release-group vs release distinction
- ✅ Returns track listings when fetching album details
- ✅ Provides comprehensive logging at every step
- ✅ Makes it easy to debug why searches fail
- ✅ Shows confidence scores and filtering decisions

All logs are prefixed with clear labels:
- `[METADATA HANDLER]` - API endpoint logs
- `[METADATA DETAILS]` - Details fetch logs
- `[METADATA SEARCH]` - Aggregator logs
- `[CONFIDENCE]` - Scoring logs
- `[MUSICBRAINZ]` - MusicBrainz provider logs