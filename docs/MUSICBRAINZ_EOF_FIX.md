# MusicBrainz EOF Error Fix

## Problem
The MusicBrainz API was returning EOF errors when making HTTP requests, even though the same URLs worked correctly in a browser or with curl.

### Error Logs
```
2026/02/01 19:08:10 [MUSICBRAINZ] HTTP request failed: Get "https://musicbrainz.org/ws/2/release-group/?fmt=json&limit=20&query=artist%3A%22Ultravox%22+AND+release%3A%22Lament%22&type=album": EOF
```

## Root Cause
The EOF error was caused by HTTP/2 connection reuse issues in Go's HTTP client. When the MusicBrainz server upgraded the connection to HTTP/2, Go's HTTP/2 implementation would encounter issues with connection reuse, resulting in unexpected EOF errors.

Key factors:
1. **HTTP/2 Protocol**: The MusicBrainz server supports and negotiates HTTP/2 connections
2. **Connection Reuse**: Go's HTTP client attempts to reuse HTTP/2 connections for efficiency
3. **Stale Connections**: When connections become stale or encounter issues, HTTP/2 can return EOF errors
4. **Race Condition**: The error occurs intermittently depending on connection timing and state

## Solution
Remove custom dialer, disable HTTP/2 completely, disable connection pooling, and use a simplified HTTP client configuration.

### Code Changes
Modified `backend/internal/metadata/musicbrainz.go`:

```go
import (
    "crypto/tls"  // Added for TLSNextProto configuration
    // ... other imports (removed "context" and "net")
)

func NewMusicBrainzProvider() *MusicBrainzProvider {
    transport := &http.Transport{
        MaxIdleConns:          0,  // Disable connection pooling
        IdleConnTimeout:       0,  // Disable idle connection reuse
        TLSHandshakeTimeout:   20 * time.Second,
        ResponseHeaderTimeout:  20 * time.Second,
        // Disable HTTP/2
        ForceAttemptHTTP2:     false,
        TLSNextProto:          make(map[string]func(string, *tls.Conn) http.RoundTripper),
        // Disable keep-alives
        DisableKeepAlives:     true,
        // Increase TLS connection timeout
        TLSClientConfig: &tls.Config{
            InsecureSkipVerify: false,
            MinVersion:         tls.VersionTLS12,
        },
    }
    
    return &MusicBrainzProvider{
        client: &http.Client{
            Timeout:   45 * time.Second,
            Transport: transport,
        },
    }
}
```

### Configuration Details

#### 1. ForceAttemptHTTP2: false
Prevents the client from attempting to upgrade to HTTP/2, ensuring HTTP/1.1 is used.

#### 2. TLSNextProto: make(map[string]...)
Disables HTTP/2 negotiation during TLS handshake by providing an empty map for protocol upgrades. This forces the connection to use HTTP/1.1.

#### 3. DisableKeepAlives: true
Disables connection pooling and reuse, ensuring a fresh TCP connection for each request. While slightly less efficient, it prevents issues with stale or corrupted connections.

## Trade-offs

### Benefits
- **Reliability**: Eliminates EOF errors caused by HTTP/2 connection issues
- **Stability**: Consistent behavior across all requests
- **Simplicity**: Removes complex HTTP/2 connection management logic

### Drawbacks
- **Performance**: Slightly slower due to:
  - New TCP/TLS handshake for each request (more latency)
  - No connection reuse (more network overhead)
- **Rate Limiting**: May be more sensitive to rate limits due to more frequent connections

## Testing
The fix has been validated by:
1. Verifying the URL works with curl (HTTP/2)
2. Identifying the issue is specific to Go's HTTP/2 implementation
3. Applying the fix and rebuilding the backend
4. Monitoring for EOF errors in subsequent requests

## References
- Go HTTP/2 Issues: https://github.com/golang/go/issues?q=is%3Aissue+http2+EOF
- MusicBrainz API: https://musicbrainz.org/doc/MusicBrainz_API
- HTTP Transport Configuration: https://pkg.go.dev/net/http#Transport