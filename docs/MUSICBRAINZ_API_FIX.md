# MusicBrainz API Fix - Release Details Issue

## Problem
When fetching release details for a MusicBrainz release-group, the application was failing with "no releases found" error for well-known albums that have multiple versions in the MusicBrainz database.

### Error Logs
```
2026/01/31 16:07:46 [MUSICBRAINZ] GetReleaseDetails called with externalID='e13af55a-84e8-31d5-a4cd-f5681616235b'
2026/01/31 16:07:46 [MUSICBRAINZ] Fetching release-group to get release ID: 'e13af55a-84e8-31d5-a4cd-f5681616235b'
2026/01/31 16:07:46 [MUSICBRAINZ] Sending request to: https://musicbrainz.org/ws/2/release-group/e13af55a-84e8-31d5-a4cd-f5681616235b?fmt=json
2026/01/31 16:07:46 [MUSICBRAINZ] Response status: 200 200 OK
2026/01/31 16:07:46 [MUSICBRAINZ] Parsing JSON response...
2026/01/31 16:07:46 [MUSICBRAINZ] No releases found for release-group 'e13af55a-84e8-31d5-a4cd-f5681616235b'
```

## Root Cause
The MusicBrainz API was being used incorrectly when fetching release-group details. According to the [MusicBrainz API documentation](https://musicbrainz.org/doc/MusicBrainz_API), the `releases` array is **not included by default** in release-group responses. You must explicitly request it using the `inc` parameter.

### Incorrect API Call
```
GET /ws/2/release-group/{id}?fmt=json
```

### Correct API Call
```
GET /ws/2/release-group/{id}?fmt=json&inc=releases
```

## Solution

### Fix 1: Force IPv4 connections
The HTTP client was attempting to use IPv6 which was causing connection failures in some network environments. Added a custom transport that forces IPv4 connections.

```go
// Create a custom dialer that forces IPv4
dialer := &net.Dialer{
    Timeout:   30 * time.Second,
    KeepAlive: 30 * time.Second,
}

// Create a custom transport that uses our dialer with IPv4 only
transport := &http.Transport{
    DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
        // Force IPv4 by using "tcp4" instead of "tcp"
        return dialer.DialContext(ctx, "tcp4", addr)
    },
    MaxIdleConns:        100,
    IdleConnTimeout:     90 * time.Second,
    TLSHandshakeTimeout: 10 * time.Second,
}
```

### Fix 2: Include releases in release-group response
Modified the `GetReleaseDetails` function in `backend/internal/metadata/musicbrainz.go` to include the `inc=releases` parameter when fetching release-group details.

```go
// Before
params := url.Values{
    "fmt": {"json"},
}

// After
params := url.Values{
    "fmt": {"json"},
    "inc":  {"releases"}, // Include releases in the response
}
```

### Fix 2: Track number type mismatch
The MusicBrainz API returns track numbers as strings (e.g., "A1", "A2" for vinyl releases), not integers. Updated the struct and track assignment:

```go
// Track struct field
TrackNumber string `json:"number"` // Changed from int to string

// Track assignment
tracks = append(tracks, models.Song{
    Title:    track.Title,
    Artist:   artistName.String(),
    Album:    release.Title,
    Track:    track.TrackNumber, // TrackNumber is already a string
    Disc:     fmt.Sprintf("%d", discNumber),
    Duration: track.Length / 1000,
})
```

## Testing
The fix has been verified with existing unit tests:
```bash
cd backend && go test -v -run TestMusicBrainzProvider ./internal/metadata/
```

Test output confirms the URL now includes the `inc=releases` parameter:
```
https://musicbrainz.org/ws/2/release-group/test-release-id?fmt=json&inc=releases
```

## Impact
This fix resolves the issue where:
- Albums with multiple release versions in MusicBrainz could not fetch details
- The error "MusicBrainz: no releases found" was incorrectly returned for valid release-groups

Now the application will:
- Correctly retrieve the list of releases associated with a release-group
- Select the first release to get detailed track information
- Successfully display metadata for well-known albums with multiple versions

## References
- MusicBrainz API Documentation: https://musicbrainz.org/doc/MusicBrainz_API
- Release-Group Lookup: https://musicbrainz.org/doc/Release_Group
- Include Parameters: https://musicbrainz.org/doc/MusicBrainz_API/Subqueries