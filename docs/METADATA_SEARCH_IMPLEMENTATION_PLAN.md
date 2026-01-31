# Metadata Search Implementation Plan

## Current State Analysis

### Existing Providers (in `backend/internal/metadata/`)

| Provider | Search | GetReleaseDetails | GetCoverArt | Status |
|----------|--------|-------------------|-------------|--------|
| Discogs | ✅ | ✅ | ✅ | Complete |
| MusicBrainz | ✅ | ✅ | ✅ | Complete |
| FreeDB | ✅ | ✅ | ✅ (returns empty) | Complete |
| AlbumArt.digital | ❌ | ❌ | ✅ | Partial |

### Missing Components

1. **Unified Metadata Aggregator** - No service to query all providers simultaneously
2. **Backend API Endpoints** - No REST/WebSocket endpoints for metadata search
3. **Frontend UI** - No interface for users to search and apply metadata
4. **Tag Writing** - No capability to write retrieved metadata back to files
5. **Configuration** - No provider settings UI

---

## Implementation Plan

### Phase 1: Backend - Metadata Aggregator Service

Create `backend/internal/metadata/aggregator.go`:

```go
// Aggregator queries multiple providers and merges results
type Aggregator struct {
    providers []Provider
}

func (a *Aggregator) Search(artist, album string) ([]MetadataCandidate, error)
func (a *Aggregator) GetCoverArt(artist, album string) ([]CoverArtCandidate, error)
```

**Key Features:**
- Parallel execution with configurable concurrency limit
- Result deduplication based on title/artist similarity
- Confidence scoring for candidate ranking
- Timeout handling per provider

### Phase 2: Backend - API Endpoints

Add to `backend/internal/api/handlers.go`:

```go
// Search metadata across all providers
GET /api/metadata/search?artist=...&album=...

// Get detailed metadata for a specific candidate
GET /api/metadata/details?provider=...&externalId=...

// Fetch cover art candidates
GET /api/metadata/coverart?artist=...&album=...

// Apply metadata to files
POST /api/metadata/apply
Body: { candidateId, paths[], options }
```

### Phase 3: Backend - Tag Writing Service

Create `backend/internal/metadata/tagwriter.go`:

```go
// TagWriter writes metadata to audio files
type TagWriter struct{}

func (w *TagWriter) ApplyMetadata(path string, metadata MetadataCandidate) error
func (w *TagWriter) ApplyCoverArt(path string, coverURL string) error
func (w *TagWriter) SupportsFormat(path string) bool
```

**Format Support:**
- FLAC (Vorbis comments)
- MP3 (ID3v1, ID3v2.3/2.4)
- OGG (Vorbis comments)
- APE, M4A (MP4 metadata)

### Phase 4: Frontend - Metadata Search UI

Add to `frontend/src/views/`:

**`MetadataSearchView.vue`:**
- Search form (artist, album inputs)
- Results grid with candidate cards
- Provider badges and confidence scores
- Cover art preview
- "Apply" action for selected candidates

**`MetadataCandidateModal.vue`:**
- Detailed view of selected candidate
- Track listing comparison
- Metadata diff (current vs proposed)
- Apply button with options

### Phase 5: Configuration

Update `backend/config.json` and add UI:

```json
{
  "metadata": {
    "providers": {
      "discogs": { "enabled": true, "token": "" },
      "musicbrainz": { "enabled": true },
      "freedb": { "enabled": true },
      "albumart": { "enabled": true, "apiKey": "" }
    },
    "search": {
      "timeout": 10,
      "maxResults": 20,
      "minConfidence": 0.5
    },
    "tagWriting": {
      "enabled": true,
      "createBackups": true,
      "backupDir": "./backups"
    }
  }
}
```

### Phase 6: Integration Points

1. **Album Detail View** - Add "Fix Metadata" button
2. **Queue View** - Bulk metadata fix for selected tracks
3. **Album Card** - Show metadata status indicator
4. **Settings** - Provider configuration UI

---

## Technical Decisions

### Search Strategy

1. **Parallel Provider Queries** - All providers queried simultaneously
2. **Early Termination** - Return when minResults found with high confidence
3. **Fallback Chain** - If primary providers fail, try alternatives

### Result Ranking

- Exact title match: +50 points
- Artist match: +30 points
- Year match: +10 points
- Provider reliability weight
- Recency preference

### Conflict Resolution

- Prefer higher-scoring candidates
- Allow manual selection when confidence is similar
- Show diff before applying

---

## Provider API Requirements

| Provider | API Token | Rate Limit | Data Quality |
|----------|-----------|------------|--------------|
| Discogs | Required | 60/min | High |
| MusicBrainz | Optional | 1/sec | Very High |
| FreeDB | Optional | None | Medium |
| AlbumArt.digital | Optional | Unknown | High (covers only) |

---

## Implementation Order

1. **Week 1:** Aggregator service + API endpoints
2. **Week 2:** Tag writing service + file backup
3. **Week 3:** Frontend search UI + candidate modal
4. **Week 4:** Integration + settings + testing

---

## Files to Create/Modify

### Backend
- `backend/internal/metadata/aggregator.go` (new)
- `backend/internal/metadata/tagwriter.go` (new)
- `backend/internal/api/handlers_metadata.go` (new)
- `backend/internal/api/handlers.go` (extend)
- `backend/internal/config/config.go` (extend)
- `backend/config.json` (extend)

### Frontend
- `frontend/src/views/MetadataSearchView.vue` (new)
- `frontend/src/components/MetadataCandidateModal.vue` (new)
- `frontend/src/components/MetadataStatusBadge.vue` (new)
- `frontend/src/views/AlbumDetailView.vue` (extend)
- `frontend/src/router/index.js` (extend)
- `frontend/src/services/metadata.js` (new)

### Documentation
- `docs/METADATA_SEARCH_API.md` (new)
- `docs/METADATA_SEARCH_UI.md` (new)
