import { useMetadata } from '../composables/useMetadata'

// Metadata service
class MetadataService {
  constructor() {
    // We'll use the composable internally for consistency
    this.composable = useMetadata()
  }

  /**
   * Search for metadata candidates
   * @param {string} artist - Artist name
   * @param {string} album - Album name
   * @param {string[]} providers - List of providers to use (e.g., ['MusicBrainz', 'Discogs'])
   * @returns {Promise<Array>} List of metadata candidates
   */
  async search(artist, album, providers = []) {
    const response = await fetch(
      `/api/metadata/search?${new URLSearchParams({
        artist,
        album,
        providers: providers.join(',')
      })}`
    )
    const data = await response.json()
    if (!data.success) {
      throw new Error(data.error || 'Search failed')
    }
    return data.data
  }

  /**
   * Get detailed metadata for a specific candidate
   * @param {string} source - Source provider (e.g., 'MusicBrainz')
   * @param {string} externalId - External ID of the release
   * @returns {Promise<Object>} Detailed metadata
   */
  async getDetails(source, externalId) {
    const response = await fetch(
      `/api/metadata/details?${new URLSearchParams({
        source,
        externalId
      })}`
    )
    const data = await response.json()
    if (!data.success) {
      throw new Error(data.error || 'Failed to get details')
    }
    return data.data
  }

  /**
   * Apply metadata to album
   * @param {string} albumPath - Path to the album directory
   * @param {Object} metadata - Metadata candidate to apply
   * @param {string} coverArtUrl - URL of cover art to apply (optional)
   * @returns {Promise<Object>} Apply result
   */
  async apply(albumPath, metadata, coverArtUrl = '') {
    const response = await fetch('/api/metadata/apply', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        albumPath,
        metadata,
        coverArtUrl
      })
    })
    const data = await response.json()
    if (!data.success) {
      throw new Error(data.error || 'Apply failed')
    }
    return data.data
  }

  /**
   * Search for cover art candidates
   * @param {string} artist - Artist name
   * @param {string} album - Album name
   * @returns {Promise<Array>} List of cover art candidates
   */
  async searchCoverArt(artist, album) {
    const response = await fetch(
      `/api/coverart/candidates?${new URLSearchParams({ artist, album })}`
    )
    const data = await response.json()
    if (!data.success) {
      throw new Error(data.error || 'Cover art search failed')
    }
    return data.data
  }

  /**
   * Apply cover art to album
   * @param {string} albumPath - Path to the album directory
   * @param {string} imageUrl - URL of cover art to apply
   * @returns {Promise<void>}
   */
  async applyCoverArt(albumPath, imageUrl) {
    const response = await fetch('/api/coverart/apply', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        albumPath,
        imageUrl
      })
    })
    const data = await response.json()
    if (!data.success) {
      throw new Error(data.error || 'Cover art apply failed')
    }
  }
}

// Export singleton instance
export default new MetadataService()
