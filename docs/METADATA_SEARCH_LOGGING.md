# Metadata Search Logging Guide

This document explains the comprehensive logging added to the metadata search system to help debug why well-known albums may not appear in search results.

## Overview

Detailed logging has been added at three levels:

1. **API Handler Level** (`backend/internal/api/handlers.go`)
2. **Aggregator Level** (`backend/internal/metadata/aggregator.go`)
3. **Provider Level** (`backend/internal/metadata/musicbrainz.go`)

## Log Categories

### 1. METADATA HANDLER Logs

These logs appear when the `/api/metadata/search` endpoint is called.

```
[METADATA HANDLER] SearchMetadata called
[METADATA HANDLER]   Artist: 'XXX'
[METADATA HANDLER]   Album: 'XXX'
[METADATA HANDLER]   Providers param: 'XXX'
[METADATA HANDLER]   Parsed providers: [XXX]
[METADATA HANDLER] Creating aggregator and starting search...
[METADATA HANDLER] Search completed successfully. Returning X candidates
```

**What to check:**
- Are the artist and album names correctly passed?
- Are providers being specified?
- How many candidates are returned total?

### 2. METADATA SEARCH Logs (Aggregator)

These logs show the search orchestration across providers.

```
[METADATA SEARCH] Starting search - Artist: 'XXX', Album: 'XXX'
[METADATA SEARCH] Available providers: X
[METADATA SEARCH]   - Provider1
[METADATA SEARCH]   - Provider2
[METADATA SEARCH] Active providers after filtering: X
[METADATA SEARCH]   - Provider1
[METADATA SEARCH] Starting parallel search across X providers
[METADATA SEARCH] [ProviderName] Starting search
[METADATA SEARCH] [ProviderName] Found X candidates
[METADATA SEARCH] [ProviderName]   Candidate 0: Artist='XXX', Album='XXX', Year='XXXX', ExternalID='XXX'
[METADATA SEARCH] [ProviderName]   Candidate 0 confidence: XX.XX
[METADATA SEARCH] All providers completed. Total candidates before deduplication: X
[METADATA SEARCH] After deduplication: X candidates
[METADATA SEARCH] Final results (sorted by confidence):
[METADATA SEARCH]   1. [XX.XX] Artist - Album (Year) from Source
```

**What to check:**
- Which providers are available vs active?
- Do any providers return 0 candidates?
- What are the confidence scores?
- Are results being deduplicated away?

### 3. CONFIDENCE Logs

These logs show how confidence scores are calculated for each candidate.

```
[CONFIDENCE] Artist: 'normalized' vs 'normalized' = X.XXXX (XX.X pts)
[CONFIDENCE] Album: 'normalized' vs 'normalized' = X.XXXX (XX.X pts)
[CONFIDENCE] Source bonus: Source = X.X pts
```

**What to check:**
- Are the normalized strings matching well?
- Are similarity scores too low (below 0.5)?
- Is the source bonus being applied?

### 4. MUSICBRAINZ Logs

These logs show detailed MusicBrainz API interaction.

```
[MUSICBRAINZ] Search called with artist='XXX', album='XXX'
[MUSICBRAINZ] Query URL params: query='artist:"XXX" AND release:"XXX"', type='album', limit='20'
[MUSICBRAINZ] Sending request to: https://musicbrainz.org/ws/2/...
[MUSICBRAINZ] Response status: 200 OK
[MUSICBRAINZ] Parsing JSON response...
[MUSICBRAINZ] API returned X release groups (count=X)
[MUSICBRAINZ] Release group 0: Title='XXX', Date='XXXX', ID='XXX'
[MUSICBRAINZ]   Artist credits: X artists
[MUSICBRAINZ]     Artist 0: 'XXX' (ID: XXX)
[MUSICBRAINZ]   Releases: X available
[MUSICBRAINZ] Creating candidate: Artist='XXX', Album='XXX', Year='XXXX', ReleaseID='XXX'
[MUSICBRAINZ] Returning X candidates
```

**What to check:**
- Is the query being constructed correctly?
- Is the API returning results (count > 0)?
- What release groups are being returned?
- Are the artist names matching?

## Common Issues and What to Look For

### Issue 1: No candidates returned

**Possible causes:**
1. **Provider disabled**: Check "Available providers" logs - is MusicBrainz listed?
2. **API returned nothing**: Check MUSICBRAINZ logs - what is the count?
3. **Query too strict**: The query might be too specific
4. **Network issues**: Check for HTTP request errors

**What to look for in logs:**
```
[METADATA SEARCH] Available providers: 0  ← Problem!
[MUSICBRAINZ] API returned 0 release groups (count=0)  ← Problem!
[MUSICBRAINZ] HTTP request failed: ...  ← Problem!
```

### Issue 2: Results returned but filtered out by confidence

**Possible causes:**
1. **String normalization issues**: Special characters, diacritics
2. **Artist name mismatch**: Different spellings, "The" prefix
3. **Album name mismatch**: Different editions, re-releases

**What to look for in logs:**
```
[CONFIDENCE] Artist: 'pink floyd' vs 'the pink floyd' = 0.5000 (15.0 pts)  ← Low similarity!
[CONFIDENCE] Album: 'dark side' vs 'the dark side of the moon' = 0.3333 (10.0 pts)  ← Low similarity!
```

### Issue 3: Results deduplicated away

**Possible causes:**
1. **Same release from multiple providers**
2. **Multiple versions of same album**

**What to look for in logs:**
```
[METADATA SEARCH] All providers completed. Total candidates before deduplication: 10
[METADATA SEARCH] After deduplication: 2 candidates  ← Many duplicates removed!
```

## How to View Logs

The logs are written to `backend/server_output.log`. To monitor them in real-time:

```bash
tail -f backend/server_output.log
```

To search for metadata-related logs:

```bash
grep "\[METADATA" backend/server_output.log
grep "\[CONFIDENCE" backend/server_output.log
grep "\[MUSICBRAINZ" backend/server_output.log
```

To see only a specific search attempt:

```bash
grep -A 20 "\[METADATA HANDLER\] SearchMetadata called" backend/server_output.log
```

## Testing Scenarios

### Test 1: Well-known album (e.g., "Dark Side of the Moon" by Pink Floyd)

Expected to see:
```
[METADATA SEARCH] Available providers: 1+
[MUSICBRAINZ] API returned X release groups (count>0)
[METADATA SEARCH] Final results (sorted by confidence):
[METADATA SEARCH]   1. [XX.XX] Pink Floyd - The Dark Side of the Moon (1973) from MusicBrainz
```

If you see:
```
[METADATA SEARCH] Final results (sorted by confidence):
[METADATA SEARCH]   (nothing)
```

Then check earlier logs for the failure point.

### Test 2: Unknown album

Expected to see:
```
[METADATA SEARCH] Available providers: 1+
[MUSICBRAINZ] API returned 0 release groups (count=0)
[METADATA SEARCH] Final results (sorted by confidence):
[METADATA SEARCH]   (nothing)
```

## Configuration Check

Ensure metadata providers are enabled in `backend/config.json`:

```json
{
  "musicBrainzEnabled": true,
  "discogsEnabled": false,
  "freeDBEnabled": false,
  "albumArtEnabled": false
}
```

Check if enabled:
```
[METADATA SEARCH] Available providers: 1
[METADATA SEARCH]   - MusicBrainz
```

## Debugging Workflow

When metadata search doesn't work for a known album:

1. **Check the handler logs** - Verify the search parameters are correct
2. **Check provider availability** - Ensure MusicBrainz is enabled
3. **Check MusicBrainz logs** - See if the API returns any results
4. **Check candidate details** - See what's actually being returned
5. **Check confidence scores** - See if results are being filtered out
6. **Check final results** - See what makes it through the pipeline

## Example: Debugging a Failed Search

**Scenario:** Search for "Abbey Road" by "The Beatles" returns no results

**Step 1:** Check handler logs
```
[METADATA HANDLER] SearchMetadata called
[METADATA HANDLER]   Artist: 'The Beatles'
[METADATA HANDLER]   Album: 'Abbey Road'
```
✓ Parameters look correct

**Step 2:** Check available providers
```
[METADATA SEARCH] Available providers: 1
[METADATA SEARCH]   - MusicBrainz
[METADATA SEARCH] Active providers after filtering: 1
```
✓ MusicBrainz is available and active

**Step 3:** Check MusicBrainz response
```
[MUSICBRAINZ] API returned 0 release groups (count=0)
```
⚠️ MusicBrainz returned nothing - why?

**Step 4:** Check the query
```
[MUSICBRAINZ] Query URL params: query='artist:"The Beatles" AND release:"Abbey Road"'
```
The query looks correct. This suggests either:
- Network issue
- MusicBrainz API rate limiting
- MusicBrainz service down

**Step 5:** Check for errors
```
[MUSICBRAINZ] HTTP request failed: ...  ← If present
[MUSICBRAINZ] Rate limiting: sleeping for ...  ← If present
```

## Summary

The logging system provides visibility into every step of the metadata search pipeline:

1. **API Handler** → Entry point, parameters
2. **Aggregator** → Provider selection, parallel execution, deduplication
3. **Provider** → API calls, raw results
4. **Confidence** → Scoring and filtering

By following the logs from top to bottom, you can pinpoint exactly where and why a metadata search is failing.