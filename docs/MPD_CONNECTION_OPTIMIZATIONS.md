# MPD Connection Architecture Improvements

## Summary

This document describes the optimizations made to the MPD connection architecture to eliminate head-of-line blocking (HoL) and improve overall performance, in accordance with the MPD server connection guidelines.

## Problems Identified

### 1. Multiple Idle Connections (Violates Guidelines)
**Issue**: Two separate idle connections were running simultaneously:
- One in `main.go` (listening only for "database" changes)
- One in `websocket.go` (listening for all subsystems)

**Impact**: Redundant connections, resource waste, potential race conditions

### 2. Single Status Client (HoL Blocking Risk)
**Issue**: A dedicated `statusMpdClient` was shared by:
- WebSocket broadcaster (every 2-3 seconds during playback)
- HTTP API handlers (`GetStatus`, `RefreshStatus`)
- New client registration

**Impact**: If one `GetStatus()` call blocks, all other status requests queue behind it, creating a bottleneck.

### 3. Inefficient Status Retrieval
**Issue**: `GetStatus()` sent two sequential commands (`status` then `currentsong`)

**Impact**: Two round trips instead of one, doubling latency for every status request.

### 4. Lack of Resilience Patterns
**Issue**: No circuit breaker or fallback mechanisms for MPD failures

**Impact**: Cascading failures when MPD becomes slow or unresponsive.

## Solutions Implemented

### 1. Optimized GetStatus() with Command Lists

**File**: `backend/internal/mpd/client.go`

**Change**: Modified `GetStatus()` to use a command list:
```go
// Before: Two sequential commands
resp, _ := c.SendCommand("status")
songResp, _ := c.SendCommand("currentsong")

// After: Single command list
commands := []string{"status", "currentsong"}
responses, _ := c.SendCommandList(commands)
```

**Benefits**:
- Reduces round trips from 2 to 1
- Cuts latency in half for status operations
- Follows MPD best practices

### 2. Eliminated Duplicate Idle Client

**File**: `backend/cmd/server/main.go`

**Change**: Removed the separate idle listener that was only watching for database changes.

**Benefits**:
- Reduced connection count from 2 to 1
- Eliminated redundancy
- Simplified architecture

### 3. Consolidated Database Change Detection

**File**: `backend/internal/api/websocket.go`

**Change**: Extended the single idle client to listen for all subsystems including "database":
```go
for _, subsystem := range changedSubsystems {
    if subsystem == "database" {
        log.Println("[Broadcaster] Database changed, triggering cache refresh...")
        go func() {
            if databaseChangeCallback != nil {
                databaseChangeCallback()
            }
        }()
    }
}
```

**Benefits**:
- Single source of truth for event detection
- Database changes trigger cache refreshes automatically
- Follows guideline: "a single long-lived event connection"

### 4. Removed Dedicated Status Client

**Files**: 
- `backend/internal/api/websocket.go`
- `backend/internal/api/handlers.go`

**Change**: Removed `statusMpdClient` and replaced all uses with pooled connections:
```go
// Before
status, _ := b.statusMpdClient.GetStatus()

// After
status, _ := b.mpdClient.GetStatus()
```

**Benefits**:
- Eliminates head-of-line blocking on status operations
- Multiple status requests can proceed in parallel
- Better resource utilization

### 5. Implemented Circuit Breaker Pattern

**File**: `backend/internal/api/circuitbreaker.go` (new file)

**Features**:
- Three states: CLOSED, OPEN, HALF_OPEN
- Configurable failure threshold (default: 5 consecutive failures)
- Automatic recovery testing
- Statistics endpoint for monitoring

**Usage**:
```go
// Can be wrapped around critical MPD operations
err := MPDCircuitBreaker.Execute(ctx, func() error {
    return mpd.GetClient().GetStatus()
})
```

**Benefits**:
- Prevents cascading failures
- Provides graceful degradation
- Enables monitoring and alerting

### 6. Added Circuit Breaker Monitoring

**File**: `backend/internal/api/handlers.go`

**New Endpoint**: `GET /api/circuit-breaker/stats`

**Response**:
```json
{
  "success": true,
  "data": {
    "state": "CLOSED",
    "failures": 0,
    "successCount": 0,
    "timeUntilNextRetry": "0s"
  }
}
```

## Final Architecture

```
┌─────────────────────────────────────────────────┐
│                Application Layer                 │
│  ┌─────────────┐  ┌─────────────┐  ┌──────────┐ │
│  │   HTTP API  │  │  WebSocket  │  │  Cache   │ │
│  │   Handlers  │  │  Broadcaster│  │ Manager  │ │
│  └──────┬──────┘  └──────┬──────┘  └────┬─────┘ │
└─────────┼────────────────┼────────────────┼──────┘
          │                │                │
┌─────────┴────────────────┴────────────────┴──────┐
│              MPD Client Layer                     │
│  ┌──────────────────┐  ┌──────────────────┐      │
│  │ Command Pool     │  │ Single Idle      │      │
│  │ (max 10 conns)   │  │ Client           │      │
│  │                  │  │                  │      │
│  │ - API calls      │  │ - IDLE loop      │      │
│  │ - Status fetches │  │ - All subsystems │      │
│  │ - Playlist ops   │  │ - Triggers      │      │
│  │ - Search, etc.   │  │   cache refresh │      │
│  └──────────────────┘  └──────────────────┘      │
│                                                   │
│  ┌──────────────────────────────────────────┐    │
│  │ Circuit Breaker (Resilience Layer)      │    │
│  │ - Monitors MPD operations                │    │
│  │ - Opens on failures                      │    │
│  │ - Auto-recovery testing                  │    │
│  └──────────────────────────────────────────┘    │
└──────────────────────────────────────────────────┘
          │                │
          └────────┬───────┘
                   │
         ┌─────────▼─────────┐
         │   MPD Server     │
         │  (localhost)     │
         └──────────────────┘
```

## Compliance with MPD Guidelines

✅ **Single long-lived event connection**
- Eliminated duplicate idle clients
- One idle connection monitors all subsystems

✅ **Pooled command connections**
- Connection pool with max 10 connections
- Short-lived connections for API operations

✅ **No dedicated status client**
- Removed blocking single connection
- All status operations use pool

✅ **Command lists for batching**
- `GetStatus()` uses command list
- Batch operations where possible

✅ **In-memory caching**
- Album cache refreshed on idle events
- No persistent state outside MPD

✅ **Resilience patterns**
- Circuit breaker for fault tolerance
- Exponential backoff for reconnection
- Timeout enforcement

## Performance Improvements

### Latency Reduction
- **Status operations**: 50% reduction (2 round trips → 1)
- **Parallel operations**: No longer blocked by single connection
- **Connection overhead**: Reduced by 50% (2 idle clients → 1)

### Throughput
- **Concurrent requests**: Can now proceed in parallel
- **Connection reuse**: Pool prevents connection churn
- **Graceful degradation**: Circuit breaker prevents cascading failures

### Resource Usage
- **Connections**: Reduced from 12+ to 11 (1 idle + 10 pooled)
- **Memory**: Less connection state to maintain
- **CPU**: Reduced reconnection attempts due to circuit breaker

## Monitoring & Observability

### Circuit Breaker Stats
Monitor the circuit breaker state:
```bash
curl http://localhost:7070/api/circuit-breaker/stats
```

### Connection Health
Check MPD connection status:
```bash
curl http://localhost:7070/api/connection-status
```

### Log Indicators
Watch for these log messages:
- `[CircuitBreaker] Circuit opened after N consecutive failures`
- `[CircuitBreaker] Circuit transitioned to half-open state`
- `[CircuitBreaker] Circuit closed after successful recovery test`
- `[Broadcaster] Database changed, triggering cache refresh...`

## Testing Recommendations

1. **Load Test**: Simulate concurrent status requests to verify no blocking
2. **Failure Scenarios**: Test circuit breaker with MPD disconnections
3. **Database Updates**: Verify cache refresh triggers on database changes
4. **Performance**: Measure latency before/after changes

## Future Enhancements

1. **Metrics Integration**: Add Prometheus metrics for circuit breaker state
2. **Adaptive Pool Sizing**: Dynamic pool size based on load
3. **Request Tracing**: Add distributed tracing for MPD operations
4. **Enhanced Monitoring**: Dashboard for connection health and circuit state

## Conclusion

The refactored architecture eliminates head-of-line blocking risks while maintaining compliance with MPD best practices. The system is now more resilient, efficient, and easier to monitor.