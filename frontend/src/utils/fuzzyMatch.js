/**
 * Fuzzy matching utilities for search relevance scoring
 */

/**
 * Calculate Levenshtein distance between two strings
 * @param {string} a - First string
 * @param {string} b - Second string
 * @returns {number} - The edit distance
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
 * @param {string} str1 - First string
 * @param {string} str2 - Second string
 * @returns {number} - Similarity score
 */
function similarity(str1, str2) {
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
 * @param {string} query - The search query
 * @param {Array} fields - Array of field names to search in
 * @returns {number} - Relevance score (higher is better)
 */
export function calculateRelevanceScore(item, query, fields) {
  if (!query || !fields || fields.length === 0) return 0
  
  const queryLower = query.toLowerCase()
  let maxScore = 0
  
  for (const field of fields) {
    const fieldValue = item[field]
    if (!fieldValue) continue
    
    const fieldValueLower = String(fieldValue).toLowerCase()
    
    // Exact match gets highest score
    if (fieldValueLower === queryLower) {
      return 1.0
    }
    
    // Starts with query gets high score
    if (fieldValueLower.startsWith(queryLower)) {
      maxScore = Math.max(maxScore, 0.8)
      continue
    }
    
    // Contains query gets medium score
    if (fieldValueLower.includes(queryLower)) {
      maxScore = Math.max(maxScore, 0.6)
      continue
    }
    
    // Calculate fuzzy similarity
    const sim = similarity(fieldValueLower, queryLower)
    if (sim > 0.3) {
      // Boost the similarity score for better differentiation
      maxScore = Math.max(maxScore, sim * 0.8)
    }
  }
  
  return maxScore
}

/**
 * Sort items by relevance to a query
 * @param {Array} items - Array of items to sort
 * @param {string} query - The search query
 * @param {Array} fields - Array of field names to search in
 * @returns {Array} - Sorted array with relevance scores attached
 */
export function sortByRelevance(items, query, fields) {
  if (!query || !fields || fields.length === 0) {
    return items.map(item => ({ ...item, _relevance: 0 }))
  }
  
  return items
    .map(item => ({
      ...item,
      _relevance: calculateRelevanceScore(item, query, fields)
    }))
    .sort((a, b) => {
      // Sort by relevance (descending)
      if (a._relevance !== b._relevance) {
        return b._relevance - a._relevance
      }
      
      // If same relevance, sort alphabetically
      return String(fields[0] in a ? a[fields[0]] : '')
        .localeCompare(String(fields[0] in b ? b[fields[0]] : ''))
    })
}

/**
 * Filter items by exact match on a field
 * @param {Array} items - Array of items to filter
 * @param {string} field - Field name to match exactly
 * @param {string} value - Value to match
 * @returns {Array} - Filtered array
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
 * @param {Array} items - Array of items to sort
 * @param {string} dateField - Field name containing the date
 * @returns {Array} - Sorted array
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