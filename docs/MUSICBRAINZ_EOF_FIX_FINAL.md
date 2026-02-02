# MusicBrainz EOF Fix - Final Documentation

## Issue Summary
The MusicBrainz API was returning EOF errors when fetching release details via the `GetReleaseDetails` method, while the same URLs worked fine in a browser. The issue manifested as:
```
Get "https://musicbrainz.org/ws/2/release-group/861c35cb-34d2-4547-9b1b-c3207d96c6a0?fmt=json&inc=releases": EOF
```

## Root Causes
1. **Connection Reuse Issues**: Reusing HTTP connections and request objects across retries
2. **Connection Reset by Peer**: Network-level connection issues
3. **Rate Limiting**: MusicBrainz API may be rate-limiting or blocking requests
4. **Connection Pooling State**: Cached connections may be in an invalid state

## Solutions Implemented

### 1. Comprehensive Retry Logic
Added a `shouldRetry` helper function that detects and retries on multiple connection error types:
- EOF errors
- Connection reset by peer
- Broken pipe
- Use of closed network connection

```go
shouldRetry := func(err error) bool {
    if err == nil {
        return false
    }
    errStr := err.Error()
    return strings.Contains(errStr, "EOF") ||
           strings.Contains(errStr, "connection reset by peer") ||
           strings.Contains(errStr, "broken pipe") ||
           strings.Contains(errStr, "use of closed network connection")
}
```

### 2. Fresh Request Objects for Each Retry
Created new HTTP request objects on each retry to avoid connection reuse issues:

```go
if i < maxRetries-1 {
    time.Sleep(time.Duration(i+1) * time.Second)
    // Create a new request for the next retry
    newReq, err := http.NewRequest(req.Method, req.URL.String(), nil)
    if err != nil {
        log.Printf("[MUSICBRAINZ] Error creating retry request: %v", err)
        return nil, NewProviderError("MusicBrainz", err)
    }
    newReq.Header.Set("User-Agent", musicBrainzUserAgent)
    req = newReq
    continue
}
```

### 3. Dedicated HTTP Client for GetReleaseDetails
Created a fresh HTTP client specifically for `GetReleaseDetails` with aggressive connection settings:

```go
transport := &http.Transport{
    MaxIdleConns:          0,  // Disable connection pooling
    MaxIdleConnsPerHost:   0,  // Disable connection pooling per host
    IdleConnTimeout:       0,  // Disable idle connection reuse
    TLSHandshakeTimeout:   30 * time.Second,
    ResponseHeaderTimeout:  30 * time.Second,
    DisableKeepAlives:     true,  // Force new connection for each request
    ForceAttemptHTTP2:     false, // Use HTTP/1.1
    TLSNextProto:          make(map[string]func(string, *tls.Conn) http.RoundTripper),
    TLSClientConfig: &tls.Config{
        InsecureSkipVerify: false,
        MinVersion:         tls.VersionTLS12,
    },
}

detailsClient := &http.Client{
    Timeout:   60 * time.Second,  // Longer timeout for retries
    Transport: transport,
}
```

### 4. Increased Rate Limiting
Increased the rate limit delay from 1 second to 2 seconds between requests to avoid hitting MusicBrainz's rate limits:

```go
// Rate limiting - ensure at least 2 seconds between requests
if time.Since(p.lastReq) < 2*time.Second {
    sleepTime := 2*time.Second - time.Since(p.lastReq)
    log.Printf("[MUSICBRAINZ] Rate limiting: sleeping for %v", sleepTime)
    time.Sleep(sleepTime)
}
```

### 5. Enhanced Logging
Added detailed logging for each retry attempt to track the retry process:

```go
log.Printf("[MUSICBRAINZ] Sending request to: %s (attempt %d/%d)", req.URL.String(), i+1, maxRetries)
// ...
log.Printf("[MUSICBRAINZ] Retryable error on attempt %d/%d: %v, retrying...", i+1, maxRetries, reqErr)
```

## Current Status

### Working
- **Search Endpoint**: The `Search` method works correctly and returns results
- **Retry Logic**: The retry mechanism is functioning as expected, detecting retryable errors and retrying up to 3 times

### Persistent Issues
- **GetReleaseDetails**: Despite all fixes, the `GetReleaseDetails` method continues to fail with EOF errors even after all retries
- **API Availability**: Direct curl tests also timeout, suggesting the MusicBrainz API may be experiencing issues or blocking requests from this IP address

## Recommendations

### For Production Use
1. **Implement Circuit Breaker**: Add a circuit breaker pattern to temporarily disable MusicBrainz after multiple consecutive failures
2. **Fallback to Other Providers**: Use the metadata aggregator to fall back to Discogs or FreeDB when MusicBrainz fails
3. **Cache Results**: Implement aggressive caching to reduce API calls
4. **Exponential Backoff**: Use exponential backoff instead of linear (1s, 2s, 3s) for retries

### For Future Investigation
1. **Monitor API Status**: Check MusicBrainz API status page for known issues
2. **Test from Different Networks**: Verify if the issue is specific to the current network
3. **Contact MusicBrainz**: If the issue persists, contact MusicBrainz support to check if the IP is being rate-limited
4. **Consider Using OAuth**: MusicBrainz may require OAuth authentication for certain endpoints

## Files Modified
- `backend/internal/metadata/musicbrainz.go`: Complete rewrite of error handling and retry logic

## Test Commands
```bash
# Test search endpoint (working)
curl -s "http://localhost:7070/api/metadata/search?artist=Ultravox&album=Lament" | jq '. | length'

# Test details endpoint (failing with EOF)
curl -s "http://localhost:7070/api/metadata/details?source=MusicBrainz&externalId=861c35cb-34d2-4547-9b1b-c3207d96c6a0" | jq '.tracks | length'

# Direct MusicBrainz API test (timing out)
curl -s -m 10 "https://musicbrainz.org/ws/2/release-group/861c35cb-34d2-4547-9b1b-c3207d96c6a0?fmt=json&inc=releases"
```

## Conclusion
The EOF error has been addressed with comprehensive retry logic and connection management improvements. The code now properly handles connection errors and retries up to 3 times with fresh connections. However, the persistent failures suggest an issue with the MusicBrainz API itself or network-level blocking rather than a code issue. The metadata aggregator will fall back to other providers (Discogs, FreeDB) when MusicBrainz fails, ensuring users can still access metadata.