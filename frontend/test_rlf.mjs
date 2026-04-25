/**
 * Test specific case that user reported: "Robert Lester Folsom"
 */

import Fuse from 'fuse.js'

// Simulate what SearchView sends - array of terms
const testAlbums = [
  { album: 'Name And Number', artist: 'Robert Lester Folsom' },
  { album: 'Blood on the Tracks', artist: 'Bob Dylan' },
  { album: 'Goodbye Yellow Brick Road', artist: 'Elton John' }
]

const fields = ['album', 'artist']

// The FIXED escaping logic from fuzzyMatch.js
function escapeQuery(queryStr) {
  const specialCharsRegex = /[!"^$| ]/
  if (specialCharsRegex.test(queryStr)) {
    return "'" + queryStr
  } else {
    return queryStr
  }
}

// Replicate sortByRelevance from fuzzyMatch.js
function sortByRelevance(items, query, fields, strict = false) {
  if (!query || !items || items.length === 0 || !fields || fields.length === 0) {
    return items.map(item => ({ ...item, _relevance: 0 }))
  }

  const queryStr = (Array.isArray(query) ? query.join(' ') : String(query || '')).trim()
  if (!queryStr) {
    return items.map(item => ({ ...item, _relevance: 0 }))
  }

  console.log('queryStr:', queryStr)
  
  const escapedQuery = escapeQuery(queryStr)
  console.log('escapedQuery:', escapedQuery)

  const options = {
    keys: fields,
    includeScore: true,
    threshold: strict ? 0.05 : 0.8,
    distance: 100,
    ignoreLocation: true,
    findAllMatches: false,
    minMatchCharLength: 1,
    useExtendedSearch: true
  }

  const fuse = new Fuse(items, options)
  const results = fuse.search(escapedQuery)
  
  console.log('Fuse results:', results.length)
  results.forEach(r => console.log('  -', r.item.artist, r.item.album))

  return results.map(result => ({
    ...result.item,
    _relevance: 1 - result.score
  }))
}

console.log('=== Testing "Robert Lester Folsom" ===')
const results = sortByRelevance(testAlbums, ['Robert Lester Folsom'], fields, false)
console.log('Final results:', results.length)
results.forEach(r => console.log('  -', r.artist, r.album))

// Also test without escaping - maybe Fuse handles it natively without extended search?
console.log('\n=== Test without extended search ===')
const options2 = {
  keys: fields,
  includeScore: true,
  threshold: 0.8,
  ignoreLocation: true,
  useExtendedSearch: false
}
const fuse2 = new Fuse(testAlbums, options2)
const results2 = fuse2.search('Robert Lester Folsom')
console.log('Results without extended:', results2.length)
results2.forEach(r => console.log('  -', r.item.artist, r.item.album))