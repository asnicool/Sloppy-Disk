/**
 * Album List Service
 * 
 * Provides fast local caching and fuzzy search for albums.
 * - Fetches complete album list on app initialization
 * - Caches albums locally for instant search results
 * - Handles cache updates via WebSocket events
 */

import axios from 'axios'

const API_BASE = '/api'

class AlbumListService {
  constructor() {
    this.albums = []
    this.loaded = false
    this.loading = false
    this.listeners = []
  }

  /**
   * Load all albums from the backend
   */
  async loadAlbums() {
    if (this.loaded || this.loading) {
      return this.albums
    }

    this.loading = true
    try {
      const response = await axios.get(`${API_BASE}/albums/all`)
      if (response.data.success) {
        this.albums = response.data.data || []
        this.loaded = true
        console.log('[AlbumList] Loaded', this.albums.length, 'albums')
        this.notifyListeners()
      }
      return this.albums
    } catch (error) {
      console.error('[AlbumList] Failed to load albums:', error)
      return []
    } finally {
      this.loading = false
    }
  }

  /**
   * Get all cached albums
   */
  getAlbums() {
    return this.albums
  }

  /**
   * Check if albums are loaded
   */
  isLoaded() {
    return this.loaded
  }

  /**
   * Search albums by any field (album name, artist, genre, date)
   * Uses simple substring matching for instant results
   * @param {string} query - Search query
   * @param {number} limit - Maximum results to return
   * @returns {Array} - Matching albums
   */
  search(query, limit = 100) {
    if (!this.loaded || !query || query.length < 1) {
      return []
    }

    const queryLower = query.toLowerCase().trim()
    if (queryLower === '') {
      return this.albums.slice(0, limit)
    }

    const results = []
    for (const album of this.albums) {
      if (this.matchesQuery(album, queryLower)) {
        results.push(album)
      }
      if (results.length >= limit) {
        break
      }
    }
    return results
  }

  /**
   * Check if an album matches the search query
   */
  matchesQuery(album, query) {
    return (
      (album.album && album.album.toLowerCase().includes(query)) ||
      (album.artist && album.artist.toLowerCase().includes(query)) ||
      (album.genre && album.genre.toLowerCase().includes(query)) ||
      (album.date && album.date.includes(query))
    )
  }

  /**
   * Add a listener for album list updates
   */
  addListener(callback) {
    this.listeners.push(callback)
    return () => {
      this.listeners = this.listeners.filter(cb => cb !== callback)
    }
  }

  /**
   * Notify all listeners of updates
   */
  notifyListeners() {
    this.listeners.forEach(callback => callback(this.albums))
  }

  /**
   * Handle database change event from WebSocket
   * Triggers a reload of the album list
   */
  handleDatabaseChange() {
    console.log('[AlbumList] Database change detected, reloading albums...')
    this.loaded = false
    this.loadAlbums()
  }

  /**
   * Get albums grouped by artist
   */
  getAlbumsByArtist() {
    const grouped = {}
    for (const album of this.albums) {
      const artist = album.artist || 'Unknown Artist'
      if (!grouped[artist]) {
        grouped[artist] = []
      }
      grouped[artist].push(album)
    }
    return grouped
  }

  /**
   * Get albums grouped by genre
   */
  getAlbumsByGenre() {
    const grouped = {}
    for (const album of this.albums) {
      const genre = album.genre || 'Unknown Genre'
      if (!grouped[genre]) {
        grouped[genre] = []
      }
      grouped[genre].push(album)
    }
    return grouped
  }

  /**
   * Get albums grouped by date
   */
  getAlbumsByDate() {
    const grouped = {}
    for (const album of this.albums) {
      const date = album.date || 'Unknown Date'
      if (!grouped[date]) {
        grouped[date] = []
      }
      grouped[date].push(album)
    }
    return grouped
  }

  /**
   * Get unique artists
   */
  getArtists() {
    const artists = new Set()
    for (const album of this.albums) {
      if (album.artist) {
        artists.add(album.artist)
      }
    }
    return Array.from(artists).sort()
  }

  /**
   * Get unique genres
   */
  getGenres() {
    const genres = new Set()
    for (const album of this.albums) {
      if (album.genre) {
        genres.add(album.genre)
      }
    }
    return Array.from(genres).sort()
  }

  /**
   * Get unique dates
   */
  getDates() {
    const dates = new Set()
    for (const album of this.albums) {
      if (album.date) {
        dates.add(album.date)
      }
    }
    return Array.from(dates).sort((a, b) => b.localeCompare(a))
  }
}

// Export singleton instance
export const albumList = new AlbumListService()

// Export class for testing
export { AlbumListService }
