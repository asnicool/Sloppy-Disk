import Fuse from 'fuse.js'

/**
 * Fuzzy matching utilities for search relevance scoring
 */

/**
 * Calculate Levenshtein distance between two strings
 * Note: Keep for backward compatibility or simple uses, but Fuse.js is preferred for main search.
 */
function levenshteinDistance(a, b) {
  const matrix = []
  
  // Initialize matrix
  for (let i = 0; i <= b.length; i++) {
    matrix[i] = [i]
  }
  
  for (let j = 0; j <= a.length; j++) {
    matrix[0][j] = j
  }
  
  // Fill matrix
  for (let i = 1; i <= b.length; i++) {
    for (let j = 1; j <= a.length; j++) {
      if (b.charAt(i - 1) === a.charAt(j - 1)) {
        matrix[i][j] = matrix[i - 1][j - 1]
      } else {
        matrix[i][j] = Math.min(
          matrix[i - 1][j - 1] + 1, // substitution
          matrix[i][j - 1] + 1,     // insertion
          matrix[i - 1][j] + 1      // deletion
        )
      }
    }
  }
  
  return matrix[b.length][a.length]
}

/**
 * Calculate similarity score between two strings (0 to 1, where 1 is exact match)
 */
export function similarity(str1, str2) {
  if (!str1 || !str2) return 0
  if (str1 === str2) return 1
  
  const longer = str1.length > str2.length ? str1 : str2
  const shorter = str1.length > str2.length ? str2 : str1
  
  if (longer.length === 0) return 1.0
  
  const editDistance = levenshteinDistance(longer, shorter)
  return (longer.length - editDistance) / longer.length
}

/**
 * Calculate a relevance score for an item against a search query
 * @param {Object} item - The item to score
 * @param {string|Array} query - The search query or queries
 * @param {Array} fields - Array of field names to search in
 * @param {boolean} strict - Whether to use strict matching (exact or starts-with only)
 * @returns {number} - Relevance score (higher is better)
 */
export function calculateRelevanceScore(item, query, fields, strict = false) {
  if (!query || !fields || fields.length === 0) return 0
  
  // Use Fuse for individual item scoring by wrapping it in an array
  // This is slightly less efficient than bulk search but keeps the API consistent
  const results = sortByRelevance([item], query, fields, strict)
  return results.length > 0 ? results[0]._relevance : 0
}

/**
 * Sort items by relevance to a query
 * @param {Array} items - Array of items to sort
 * @param {string|Array} query - The search query or queries
 * @param {Array} fields - Array of field names to search in
 * @param {boolean} strict - Whether to use strict matching
 * @returns {Array} - Sorted array with relevance scores attached
 */
export function sortByRelevance(items, query, fields, strict = false) {
  if (!query || !items || items.length === 0 || !fields || fields.length === 0) {
    return (items || []).map(item => ({ ...item, _relevance: 0 }))
  }

  // Ensure query is a string and not empty
  const queryStr = (Array.isArray(query) ? query.join(' ') : String(query || '')).trim()
  if (!queryStr) {
     return items.map(item => ({ ...item, _relevance: 0 }))
  }

  // Escape special characters that have meaning in Fuse.js extended search
  // Characters: ! " ^ $ | and space (for AND/OR operators)
  // Single quote ' is also special in extended search
  
  // Check if query contains special characters that need escaping
  const specialCharsRegex = /[!"^$| ]/
  
  let escapedQuery
  if (specialCharsRegex.test(queryStr)) {
    // Use = prefix to force "include" matching, which treats special chars literally
    // This is more reliable than backslash escaping for strict mode
    escapedQuery = '=' + queryStr
  } else {
    // No special chars - use as-is for normal fuzzy matching
    escapedQuery = queryStr
  }

  // Fuse.js options
  const options = {
    keys: fields,
    includeScore: true,
    threshold: strict ? 0.05 : 0.8, // Extremely lenient for initial candidate gathering
    distance: 100,
    ignoreLocation: true,
    findAllMatches: false,
    minMatchCharLength: 1,
    useExtendedSearch: true
  }

  const fuse = new Fuse(items, options)
  const fuseResults = fuse.search(escapedQuery)

  // Split query into tokens for coverage scoring
  const tokens = queryStr.toLowerCase().split(/\s+/).filter(t => t.length > 1)
  
  // Map results and apply Token Coverage Boost
  const matchedItems = fuseResults.slice(0, 1000).map(result => {
    let relevance = 1 - result.score
    
    if (tokens.length > 1 && !strict) {
        // Boost relevance if it matches multiple tokens
        const itemText = fields.map(f => {
            const fieldName = typeof f === 'object' ? f.name : f
            return String(result.item[fieldName] || '').toLowerCase()
        }).join(' ')
        
        let tokenMatches = 0
        tokens.forEach(token => {
            if (itemText.includes(token)) {
                tokenMatches += 1
            } else {
                // Check for close fuzzy match for this token specifically
                // This is a partial check to boost even if token has a typo
                if (token.length > 3) {
                    const subTokens = [
                        token.substring(0, Math.floor(token.length * 0.6)), // First halfish
                        token.substring(1), // Shifted
                        token.substring(0, token.length - 1) // Truncated
                    ]
                    if (subTokens.some(st => st.length >= 3 && itemText.includes(st))) {
                        tokenMatches += 0.5
                    }
                }
            }
        })
        
        // Boost factor: matching 2/3 words is much better than matching 1/3
        const coverageBoost = 1 + (tokenMatches / tokens.length)
        relevance *= coverageBoost
    }

    return {
      ...result.item,
      _relevance: relevance
    }
  })

  // Re-sort based on boosted relevance
  matchedItems.sort((a, b) => b._relevance - a._relevance)

  // Add items that didn't match with 0 relevance if needed, 
  // but existing sortByRelevance filtered them out.
  // SearchView expects only matches.

  return matchedItems
}

/**
 * Filter items by exact match on a field
 */
export function filterByExactMatch(items, field, value) {
  if (!field || !value) return items
  
  return items.filter(item => {
    const fieldValue = item[field]
    if (!fieldValue) return false
    
    return String(fieldValue).toLowerCase() === String(value).toLowerCase()
  })
}

/**
 * Sort items by date (newest first)
 */
export function sortByDateDesc(items, dateField = 'date') {
  return [...items].sort((a, b) => {
    const dateA = a[dateField] || ''
    const dateB = b[dateField] || ''
    
    // Extract year (first 4 characters)
    const yearA = parseInt(dateA.substring(0, 4)) || 0
    const yearB = parseInt(dateB.substring(0, 4)) || 0
    
    // If years are different, sort by year
    if (yearA !== yearB) {
      return yearB - yearA
    }
    
    // If same year, compare full date strings
    return dateB.localeCompare(dateA)
  })
}