/**
 * Test with larger dataset to simulate real scenario
 */

import Fuse from 'fuse.js'

// Generate 29025 albums similar to user's dataset
const generateAlbums = () => {
  const artists = [
    'Bob Dylan', 'Elton John', 'Pink Floyd', 'Fleetwood Mac', 'Led Zeppelin',
    'Joni Mitchell', 'Billy Joel', 'Neil Young', 'The Beatles', 'The Rolling Stones',
    'Robert Lester Folsom', 'The Man with the Flower in His Nose', 'Prince', 
    'Michael Jackson', 'Madonna', 'David Bowie', 'Queen', 'Bruce Springsteen'
  ]
  const albums = ['Greatest Hits', 'Live', 'The Best of', 'Name And Number', 'First Album', 'Second Chance']
  
  const result = []
  for (let i = 0; i < 29025; i++) {
    result.push({
      album: `${albums[i % albums.length]} ${i}`,
      artist: artists[i % artists.length]
    })
  }
  return result
}

const testAlbums = generateAlbums()
console.log('Generated', testAlbums.length, 'albums')

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

function sortByRelevance(items, query, fields, strict = false) {
  const queryStr = (Array.isArray(query) ? query.join(' ') : String(query || '')).trim()
  if (!queryStr) {
    return items.map(item => ({ ...item, _relevance: 0 }))
  }

  const escapedQuery = escapeQuery(queryStr)

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
  
  return results.map(result => ({
    ...result.item,
    _relevance: 1 - result.score
  }))
}

console.log('\n=== Testing with 29025 albums ===')

console.log('\n1. Searching for "Robert Lester Folsom"')
let start = Date.now()
let results = sortByRelevance(testAlbums, ['Robert Lester Folsom'], fields, false)
let elapsed = Date.now() - start
console.log(`   Found ${results.length} results in ${elapsed}ms`)
if (results.length > 0) {
  console.log(`   First result: ${results[0].artist} - ${results[0].album}`)
}

console.log('\n2. Searching for "Bob Dylan"')
start = Date.now()
results = sortByRelevance(testAlbums, ['Bob Dylan'], fields, false)
elapsed = Date.now() - start
console.log(`   Found ${results.length} results in ${elapsed}ms`)
if (results.length > 0) {
  console.log(`   First result: ${results[0].artist} - ${results[0].album}`)
}

console.log('\n3. Searching for "BobDylan" (no space)')
start = Date.now()
results = sortByRelevance(testAlbums, ['BobDylan'], fields, false)
elapsed = Date.now() - start
console.log(`   Found ${results.length} results in ${elapsed}ms`)
if (results.length > 0) {
  console.log(`   First result: ${results[0].artist} - ${results[0].album}`)
}

console.log('\n4. Testing without extended search for comparison')
const optionsNoExt = {
  keys: fields,
  includeScore: true,
  threshold: 0.8,
  ignoreLocation: true,
  useExtendedSearch: false
}
const fuseNoExt = new Fuse(testAlbums, optionsNoExt)
start = Date.now()
const resultsNoExt = fuseNoExt.search('Robert Lester Folsom')
elapsed = Date.now() - start
console.log(`   Found ${resultsNoExt.length} results in ${elapsed}ms`)
if (resultsNoExt.length > 0) {
  console.log(`   First result: ${resultsNoExt[0].item.artist} - ${resultsNoExt[0].item.album}`)
}