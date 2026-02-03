import { ref } from 'vue'

// Metadata search composable
export function useMetadata() {
  const candidates = ref([])
  const selectedCandidate = ref(null)
  const loading = ref(false)
  const error = ref(null)
  const applyResult = ref(null)

  const searchMetadata = async (artist, album, providers = [], trackCount = 0, duration = 0) => {
    loading.value = true
    error.value = null
    candidates.value = []

    try {
      const params = new URLSearchParams({
        artist,
        album
      })
      if (providers.length > 0) {
        params.append('providers', providers.join(','))
      }
      if (trackCount > 0) {
        params.append('trackCount', trackCount)
      }
      if (duration > 0) {
        params.append('duration', duration)
      }

      const response = await fetch(`/api/metadata/search?${params}`)
      const data = await response.json()

      if (data.success) {
        candidates.value = data.data
      } else {
        throw new Error(data.error || 'Search failed')
      }
    } catch (e) {
      error.value = e.message
    } finally {
      loading.value = false
    }
  }

  const getMetadataDetails = async (source, externalId) => {
    loading.value = true
    error.value = null

    try {
      const params = new URLSearchParams({
        source,
        externalId
      })

      const response = await fetch(`/api/metadata/details?${params}`)
      const data = await response.json()

      if (data.success) {
        selectedCandidate.value = data.data
        return data.data
      } else {
        throw new Error(data.error || 'Failed to get details')
      }
    } catch (e) {
      error.value = e.message
      return null
    } finally {
      loading.value = false
    }
  }

  const applyMetadata = async (albumPath, metadata, coverArtUrl = '') => {
    loading.value = true
    error.value = null
    applyResult.value = null

    try {
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

      if (data.success) {
        applyResult.value = data.data
        return data.data
      } else {
        throw new Error(data.error || 'Apply failed')
      }
    } catch (e) {
      error.value = e.message
      return null
    } finally {
      loading.value = false
    }
  }

  const searchCoverArt = async (artist, album) => {
    try {
      const params = new URLSearchParams({ artist, album })
      const response = await fetch(`/api/coverart/candidates?${params}`)
      const data = await response.json()
      return data.success ? data.data : []
    } catch (e) {
      console.error('Failed to search cover art:', e)
      return []
    }
  }

  const clearSelection = () => {
    selectedCandidate.value = null
    applyResult.value = null
  }

  return {
    candidates,
    selectedCandidate,
    loading,
    error,
    applyResult,
    searchMetadata,
    getMetadataDetails,
    applyMetadata,
    searchCoverArt,
    clearSelection
  }
}
