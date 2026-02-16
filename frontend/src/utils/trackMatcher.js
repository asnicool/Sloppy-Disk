/**
 * Track matching utility for comparing library tracks with metadata provider tracks
 * Uses disc/track numbers as primary key, fuzzy title matching as secondary
 */

import { similarity } from './fuzzyMatch'

/**
 * Match library tracks to metadata provider tracks
 * @param {Array} libraryTracks - Tracks from the library (MPD)
 * @param {Array} metadataTracks - Tracks from metadata provider
 * @returns {Object} Matching result with pairs, extras, and analysis
 */
export function matchTracks(libraryTracks, metadataTracks) {
  // Build maps for quick lookup
  const libraryMap = new Map() // Key: "disc-track", Value: track object
  const metadataMap = new Map() // Key: "disc-track", Value: track object

  // Index library tracks
  libraryTracks.forEach((track, index) => {
    const key = getTrackKey(track)
    if (!libraryMap.has(key)) {
      libraryMap.set(key, { ...track, _originalIndex: index })
    }
  })

  // Index metadata tracks
  metadataTracks.forEach((track, index) => {
    const key = getTrackKey(track)
    if (!metadataMap.has(key)) {
      metadataMap.set(key, { ...track, _originalIndex: index })
    }
  })

  // Find exact matches (by disc + track number)
  const exactMatches = []
  const libraryOnly = []
  const metadataOnly = []

  // Check for exact matches
  libraryMap.forEach((libTrack, key) => {
    if (metadataMap.has(key)) {
      const metaTrack = metadataMap.get(key)
      const titleSimilarity = similarity(
        String(libTrack.title || ''),
        String(metaTrack.title || '')
      ) * 100 // Convert to percentage

      exactMatches.push({
        library: libTrack,
        metadata: metaTrack,
        matchType: 'exact',
        titleSimilarity,
        key
      })

      // Remove from metadata map to track unmatched
      metadataMap.delete(key)
    } else {
      libraryOnly.push(libTrack)
    }
  })

  // Remaining metadata tracks are unmatched
  metadataMap.forEach((metaTrack) => {
    metadataOnly.push(metaTrack)
  })

  // Try fuzzy matching for unmatched tracks
  const fuzzyMatches = fuzzyMatchUnmatched(libraryOnly, metadataOnly)

  // Rebuild lists after fuzzy matching
  const finalLibraryOnly = libraryOnly.filter(t => !t._matched)
  const finalMetadataOnly = metadataOnly.filter(t => !t._matched)

  // Check if tracks are in different order
  const orderChanged = checkOrderChanged(exactMatches, fuzzyMatches)

  return {
    matches: [...exactMatches, ...fuzzyMatches],
    libraryOnly: finalLibraryOnly,
    metadataOnly: finalMetadataOnly,
    stats: {
      totalLibrary: libraryTracks.length,
      totalMetadata: metadataTracks.length,
      matched: exactMatches.length + fuzzyMatches.length,
      exactMatches: exactMatches.length,
      fuzzyMatches: fuzzyMatches.length,
      libraryOnly: finalLibraryOnly.length,
      metadataOnly: finalMetadataOnly.length,
      orderChanged
    }
  }
}

/**
 * Generate a unique key for a track based on disc and track number
 */
function getTrackKey(track) {
  const disc = track.disc || 1
  const trackNum = track.track || 0
  return `${disc}-${trackNum}`
}

/**
 * Fuzzy match unmatched library tracks to unmatched metadata tracks
 * Uses title similarity as matching criteria
 */
function fuzzyMatchUnmatched(libraryOnly, metadataOnly) {
  const matches = []
  const usedMetadata = new Set()

  libraryOnly.forEach(libTrack => {
    let bestMatch = null
    let bestScore = 0
    let bestIndex = -1

    metadataOnly.forEach((metaTrack, index) => {
      if (usedMetadata.has(index)) return

      const score = similarity(
        String(libTrack.title || ''),
        String(metaTrack.title || '')
      ) * 100 // Convert to percentage

      // Require at least 70% similarity for fuzzy match
      if (score > 70 && score > bestScore) {
        bestScore = score
        bestMatch = metaTrack
        bestIndex = index
      }
    })

    if (bestMatch) {
      libTrack._matched = true
      bestMatch._matched = true
      usedMetadata.add(bestIndex)

      matches.push({
        library: libTrack,
        metadata: bestMatch,
        matchType: 'fuzzy',
        titleSimilarity: bestScore,
        key: `${libTrack.disc || 1}-${libTrack.track || 0} ≈ ${bestMatch.disc || 1}-${bestMatch.track || 0}`
      })
    }
  })

  return matches
}

/**
 * Check if tracks are in different order between library and metadata
 */
function checkOrderChanged(exactMatches, fuzzyMatches) {
  const allMatches = [...exactMatches, ...fuzzyMatches]
  if (allMatches.length < 2) return false

  // Compare original indices
  let orderChanged = false
  for (let i = 0; i < allMatches.length - 1; i++) {
    const current = allMatches[i]
    const next = allMatches[i + 1]

    // Check if library order matches metadata order
    const libOrder = current.library._originalIndex < next.library._originalIndex
    const metaOrder = current.metadata._originalIndex < next.metadata._originalIndex

    if (libOrder !== metaOrder) {
      orderChanged = true
      break
    }
  }

  return orderChanged
}

/**
 * Generate a human-readable match status
 */
export function getMatchStatus(match) {
  if (match.matchType === 'exact') {
    if (match.titleSimilarity >= 95) return 'Perfect Match'
    if (match.titleSimilarity >= 80) return 'Minor Title Difference'
    return 'Major Title Difference'
  } else {
    if (match.titleSimilarity >= 90) return 'Fuzzy Match (Excellent)'
    if (match.titleSimilarity >= 80) return 'Fuzzy Match (Good)'
    return 'Fuzzy Match (Fair)'
  }
}

/**
 * Get color class for match status
 */
export function getMatchStatusClass(match) {
  if (match.matchType === 'exact') {
    if (match.titleSimilarity >= 95) return 'text-green-400'
    if (match.titleSimilarity >= 80) return 'text-amber-400'
    return 'text-orange-400'
  } else {
    if (match.titleSimilarity >= 90) return 'text-green-400'
    if (match.titleSimilarity >= 80) return 'text-amber-400'
    return 'text-orange-400'
  }
}
