/**
 * Album Cache Service
 * 
 * Provides LRU caching with stale-while-revalidate pattern for album details.
 * - Cache size: 200 albums
 * - Stale threshold: 10 minutes
 * - Lazy cleanup when cache is full
 */

const CACHE_SIZE = 200
const STALE_THRESHOLD = 10 * 60 * 1000 // 10 minutes in milliseconds

class AlbumCache {
  constructor() {
    // Map maintains insertion order, used for LRU tracking
    // Key: artist|album, Value: { data, timestamp, lastAccess, etag }
    this.cache = new Map()
    this.stats = {
      hits: 0,
      misses: 0,
      staleHits: 0,
      evictions: 0
    }
  }

  /**
   * Generate cache key from artist and album
   */
  generateKey(artist, album) {
    return `${artist}|${album}`.toLowerCase()
  }

  /**
   * Get cached album data
   * @param {string} artist - Artist name
   * @param {string} album - Album name
   * @returns {Object|null} - Cached data or null if not found
   */
  get(artist, album) {
    const key = this.generateKey(artist, album)
    const entry = this.cache.get(key)
    
    if (!entry) {
      this.stats.misses++
      return null
    }
    
    // Update lastAccess timestamp (LRU tracking)
    entry.lastAccess = Date.now()
    
    // Move to end of map to mark as recently accessed
    this.cache.delete(key)
    this.cache.set(key, entry)
    
    const age = Date.now() - entry.timestamp
    const isStale = age > STALE_THRESHOLD
    
    if (isStale) {
      this.stats.staleHits++
    } else {
      this.stats.hits++
    }
    
    return {
      data: entry.data,
      isStale,
      age
    }
  }

  /**
   * Set album data in cache
   * @param {string} artist - Artist name
   * @param {string} album - Album name
   * @param {Object} data - Album details data
   * @param {string} etag - Optional ETag for change detection
   */
  set(artist, album, data, etag = null) {
    const key = this.generateKey(artist, album)
    const now = Date.now()
    
    // Lazy cleanup if cache is full
    if (this.cache.size >= CACHE_SIZE) {
      this._evictOldest()
    }
    
    this.cache.set(key, {
      data,
      timestamp: now,
      lastAccess: now,
      etag
    })
  }

  /**
   * Update existing cache entry without changing timestamp
   * @param {string} artist - Artist name
   * @param {string} album - Album name
   * @param {Object} data - Updated album details
   */
  update(artist, album, data) {
    const key = this.generateKey(artist, album)
    const entry = this.cache.get(key)
    
    if (entry) {
      entry.data = data
      // Don't update timestamp - keep original access time
    }
  }

  /**
   * Check if cache entry exists and is fresh (not stale)
   * @param {string} artist - Artist name
   * @param {string} album - Album name
   * @returns {boolean}
   */
  isFresh(artist, album) {
    const entry = this.cache.get(this.generateKey(artist, album))
    if (!entry) return false
    return (Date.now() - entry.timestamp) <= STALE_THRESHOLD
  }

  /**
   * Get cache entry age in milliseconds
   * @param {string} artist - Artist name
   * @param {string} album - Album name
   * @returns {number|null} - Age in ms or null if not found
   */
  getAge(artist, album) {
    const entry = this.cache.get(this.generateKey(artist, album))
    if (!entry) return null
    return Date.now() - entry.timestamp
  }

  /**
   * Remove entry from cache
   * @param {string} artist - Artist name
   * @param {string} album - Album name
   */
  invalidate(artist, album) {
    const key = this.generateKey(artist, album)
    this.cache.delete(key)
  }

  /**
   * Clear all cached data
   */
  clear() {
    this.cache.clear()
  }

  /**
   * Get cache statistics
   * @returns {Object} - Cache stats
   */
  getStats() {
    const total = this.stats.hits + this.stats.misses
    const hitRate = total > 0 ? (this.stats.hits / total * 100).toFixed(1) : 0
    
    return {
      size: this.cache.size,
      maxSize: CACHE_SIZE,
      hitRate: `${hitRate}%`,
      hits: this.stats.hits,
      misses: this.stats.misses,
      staleHits: this.stats.staleHits,
      evictions: this.stats.evictions
    }
  }

  /**
   * Reset statistics
   */
  resetStats() {
    this.stats = {
      hits: 0,
      misses: 0,
      staleHits: 0,
      evictions: 0
    }
  }

  /**
   * Evict oldest accessed entry (LRU)
   * @private
   */
  _evictOldest() {
    // Map maintains insertion order, first key is oldest
    const firstKey = this.cache.keys().next().value
    if (firstKey) {
      this.cache.delete(firstKey)
      this.stats.evictions++
    }
  }

  /**
   * Force cleanup of oldest entries to reduce cache to target size
   * @param {number} targetSize - Target cache size (default: 80% of max)
   */
  trim(targetSize = Math.floor(CACHE_SIZE * 0.8)) {
    while (this.cache.size > targetSize) {
      this._evictOldest()
    }
  }
}

// Export singleton instance
export const albumCache = new AlbumCache()

// Export class for testing
export { AlbumCache, CACHE_SIZE, STALE_THRESHOLD }
