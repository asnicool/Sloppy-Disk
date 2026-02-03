import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import axios from 'axios'
import { albumCache } from '@/services/albumCache'

const API_BASE = '/api'

export const useMpdStore = defineStore('mpd', () => {
  // State
  const status = ref(null)
  const playlist = ref([])
  const playlistCurrentPos = ref(-1)
  const isConnected = ref(false)
  const ws = ref(null)
  const searchWs = ref(null)
  const connectionError = ref(null)
  const localElapsed = ref(0)
  const lastStatusTime = ref(0)
  const pollInterval = ref(null)
  const config = ref(null)
  const lastActivityRefreshTime = ref(0)
  const currentBackoffIndex = ref(0)
  const backoffLevels = [10000, 60000, 600000] // 10s, 1m, 10m
  
  // Search state
  const searchResults = ref({
    albums: [],
    artists: [],
    genres: [],
    dates: [],
    songs: []
  })
  const isSearching = ref(false)

  // Getters
  const currentSong = computed(() => status.value?.currentSong)
  const isPlaying = computed(() => status.value?.state === 'play')
  const currentTime = computed(() => status.value?.elapsed || 0)
  const duration = computed(() => status.value?.duration || 0)
  const volume = computed(() => status.value?.volume || 0)

  // Actions
  const connect = async () => {
    try {
      console.log('[MPD] Connecting...')
      const response = await axios.get(`${API_BASE}/status`)
      console.log('[MPD] Status response:', response.data)
      status.value = response.data.data
      isConnected.value = true
      console.log('[MPD] Status set:', status.value)
      
      connectWebSocket()
      connectSearchWebSocket()
    } catch (error) {
      console.error('Failed to connect to MPD:', error)
      isConnected.value = false
      connectionError.value = error.message
    }
  }

  const refreshStatus = async () => {
    try {
      const response = await axios.post(`${API_BASE}/status/refresh`)
      return response.data
    } catch (error) {
      console.error('Failed to refresh MPD status:', error)
      throw error
    }
  }

  const connectWebSocket = () => {
    if (ws.value && ws.value.readyState === WebSocket.OPEN) {
      console.log('WebSocket already connected, skipping')
      return
    }
    if (ws.value && ws.value.readyState === WebSocket.CONNECTING) {
      console.log('WebSocket already connecting, skipping')
      return
    }
    
    try {
      if (ws.value) {
        ws.value.close()
        ws.value = null
      }
      
      const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      ws.value = new WebSocket(`${wsProtocol}//${window.location.host}/ws`)
      
      ws.value.onopen = () => {
        console.log('WebSocket connected')
        if (ws.value && ws.value.readyState === WebSocket.OPEN) {
          ws.value.send(JSON.stringify({ type: 'get_status' }))
        }
      }
      
      ws.value.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data)
          if (msg.type === 'status' && msg.data) {
            if (msg.data.state !== undefined) {
              status.value = msg.data
              if (msg.data.playlistPos !== undefined) {
                playlistCurrentPos.value = msg.data.playlistPos
              }
              console.log('Status updated via WebSocket:', status.value?.state, status.value?.currentSong?.title)
            }
          }
        } catch (error) {
          console.error('WebSocket message parse error:', error)
        }
      }
      
      ws.value.onclose = () => {
        console.log('WebSocket disconnected')
        ws.value = null
        if (isConnected.value) {
          setTimeout(() => {
            if (isConnected.value && (!ws.value || ws.value.readyState !== WebSocket.OPEN)) {
              connectWebSocket()
            }
          }, 5000)
        }
      }
      
      ws.value.onerror = (error) => {
        console.error('WebSocket error:', error)
      }

      getConfig()
    } catch (error) {
      console.error('Failed to connect WebSocket:', error)
    }
  }

  const connectSearchWebSocket = () => {
    if (searchWs.value && searchWs.value.readyState === WebSocket.OPEN) {
      return
    }
    
    try {
      if (searchWs.value) {
        searchWs.value.close()
        searchWs.value = null
      }

      const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      searchWs.value = new WebSocket(`${wsProtocol}//${window.location.host}/ws/search`)
      
      searchWs.value.onopen = () => {
        console.log('Search WebSocket connected')
      }

      searchWs.value.onmessage = (event) => {
        try {
          const msg = JSON.parse(event.data)
          if (msg.type === 'search_results') {
            const { category, items } = msg.data
            if (searchResults.value[category]) {
              searchResults.value[category] = items
            }
          }
        } catch (error) {
          console.error('Search WebSocket message parse error:', error)
        }
      }

      searchWs.value.onclose = () => {
        console.log('Search WebSocket disconnected')
        searchWs.value = null
      }
    } catch (error) {
      console.error('Failed to connect Search WebSocket:', error)
    }
  }

  // Playback controls
  const play = async () => {
    try {
      await axios.post(`${API_BASE}/play`)
    } catch (error) {
      console.error('Play failed:', error)
    }
  }

  const pause = async () => {
    try {
      await axios.post(`${API_BASE}/pause`)
    } catch (error) {
      console.error('Pause failed:', error)
    }
  }

  const next = async () => {
    try {
      await axios.post(`${API_BASE}/next`)
    } catch (error) {
      console.error('Next failed:', error)
    }
  }

  const previous = async () => {
    try {
      await axios.post(`${API_BASE}/previous`)
    } catch (error) {
      console.error('Previous failed:', error)
    }
  }

  const setVolume = async (newVolume) => {
    try {
      await axios.post(`${API_BASE}/volume/${newVolume}`)
    } catch (error) {
      console.error('Set volume failed:', error)
    }
  }

  // Data fetching
  const fetchAlbums = async (page = 1, limit = 50, search = '', sort = '') => {
    try {
      const params = new URLSearchParams({
        page: page.toString(),
        limit: limit.toString()
      })
      
      if (search) params.append('search', search)
      if (sort) params.append('sort', sort)

      const response = await axios.get(`${API_BASE}/albums?${params}`)
      return response.data
    } catch (error) {
      console.error('Fetch albums failed:', error)
      throw error
    }
  }

  const fetchRandomAlbums = async (count, refresh = false) => {
    try {
      const params = new URLSearchParams()
      if (count) params.append('count', count.toString())
      if (refresh) params.append('refresh', 'true')
      const response = await axios.get(`${API_BASE}/albums/random?${params}`)
      return response.data
    } catch (error) {
      console.error('Fetch random albums failed:', error)
      throw error
    }
  }

  const fetchArtists = async (page = 1, limit = 50, search = '') => {
    try {
      const params = new URLSearchParams({
        page: page.toString(),
        limit: limit.toString()
      })
      
      if (search) params.append('search', search)

      const response = await axios.get(`${API_BASE}/artists?${params}`)
      return response.data
    } catch (error) {
      console.error('Fetch artists failed:', error)
      throw error
    }
  }

  const fetchAlbumsByDate = async (page = 1, limit = 50) => {
    try {
      const params = new URLSearchParams({
        page: page.toString(),
        limit: limit.toString()
      })
      const response = await axios.get(`${API_BASE}/dates?${params}`)
      return response.data
    } catch (error) {
      console.error('Fetch dates failed:', error)
      throw error
    }
  }

  const fetchAlbumsByGenre = async (page = 1, limit = 50) => {
    try {
      const params = new URLSearchParams({
        page: page.toString(),
        limit: limit.toString()
      })
      const response = await axios.get(`${API_BASE}/genres?${params}`)
      return response.data
    } catch (error) {
      console.error('Fetch genres failed:', error)
      throw error
    }
  }

  const fetchAlbumSongs = async (artist, album, page = 1, limit = 50, forceRefresh = false) => {
    try {
      const params = new URLSearchParams({
        page: page.toString(),
        limit: limit.toString()
      })

      // Check cache first (unless forceRefresh)
      if (!forceRefresh) {
        const cached = albumCache.get(artist, album)
        if (cached && !cached.isStale) {
          console.log('[MPD Store] Cache hit (fresh):', artist, '-', album)
          return cached.data
        }
        
        if (cached && cached.isStale) {
          console.log('[MPD Store] Cache hit (stale):', artist, '-', album, '- fetching fresh in background')
          // Return stale data immediately, then fetch fresh in background
          // Schedule background refresh without awaiting
          setTimeout(async () => {
            try {
              const response = await axios.get(`${API_BASE}/album/${encodeURIComponent(artist)}/${encodeURIComponent(album)}?${params}`)
              if (response.data.success) {
                // Check if data has changed
                const cachedData = albumCache.get(artist, album)
                if (cachedData && JSON.stringify(cachedData.data) !== JSON.stringify(response.data.data)) {
                  albumCache.set(artist, album, response.data.data)
                  // Emit event for UI refresh if needed
                  window.dispatchEvent(new CustomEvent('album-cache-updated', { 
                    detail: { artist, album, data: response.data.data } 
                  }))
                }
              }
            } catch (error) {
              console.error('[MPD Store] Background refresh failed:', error)
            }
          }, 0)
          
          // Return cached stale data immediately for responsiveness
          return cached.data
        }
      }

      // Cache miss or force refresh - fetch from API
      console.log('[MPD Store] Cache miss or force refresh:', artist, '-', album)
      const response = await axios.get(`${API_BASE}/album/${encodeURIComponent(artist)}/${encodeURIComponent(album)}?${params}`)
      console.log('[MPD Store] API response status:', response.status, response.data)
      
      if (response.data.success) {
        // Cache the response
        albumCache.set(artist, album, response.data.data)
        console.log('[MPD Store] Cached data:', response.data.data)
        return response.data.data
      }
      
      console.error('[MPD Store] API returned success=false:', response.data)
      return response.data
    } catch (error) {
      console.error('Fetch album songs failed:', error)
      throw error
    }
  }

  const search = async (query, type = 'song', page = 1, limit = 30) => {
    try {
      const params = new URLSearchParams({
        q: query,
        type: type,
        page: page.toString(),
        limit: limit.toString()
      })

      const response = await axios.get(`${API_BASE}/search?${params}`)
      return response.data
    } catch (error) {
      console.error('Search failed:', error)
      throw error
    }
  }

  const addToPlaylist = async (uri) => {
    try {
      await axios.post(`${API_BASE}/playlist/add/${encodeURIComponent(uri)}`)
      await fetchPlaylist()
    } catch (error) {
      console.error('Add to playlist failed:', error)
      throw error
    }
  }

  const addAlbumToPlaylist = async (artist, album, mode = 'append') => {
    try {
      await axios.post(`${API_BASE}/playlist/album`, { artist, album, mode })
      await fetchPlaylist()
    } catch (error) {
      console.error('Add album to playlist failed:', error)
      throw error
    }
  }

  const removeFromPlaylist = async (position) => {
    try {
      await axios.post(`${API_BASE}/playlist/remove/${position}`)
      await fetchPlaylist()
    } catch (error) {
      console.error('Remove from playlist failed:', error)
      throw error
    }
  }

  const fetchPlaylist = async () => {
    try {
      console.log('[MPD Store] Fetching playlist...')
      const response = await axios.get(`${API_BASE}/playlist`)
      if (response.data.success) {
        console.log('[MPD Store] Playlist fetched, items:', response.data.data.items.length)
        playlist.value = response.data.data.items
        playlistCurrentPos.value = response.data.data.currentPos
      }
      return response.data
    } catch (error) {
      console.error('Fetch playlist failed:', error)
      throw error
    }
  }

  const moveTrack = async (from, to) => {
    try {
      await axios.post(`${API_BASE}/playlist/move`, { from, to })
      await fetchPlaylist()
    } catch (error) {
      console.error('Move track failed:', error)
      throw error
    }
  }

  const moveAlbum = async (start, length, to) => {
    try {
      await axios.post(`${API_BASE}/playlist/move`, { from: start, to, length })
      await fetchPlaylist()
    } catch (error) {
      console.error('Move album failed:', error)
      throw error
    }
  }

  const addTracks = async (uris, mode = 'append') => {
    try {
      // Get current state for Play Next calculation
      await fetchPlaylist()
      const startLength = playlist.value.length
      const currentPos = playlistCurrentPos.value

      // Add all tracks
      // Note: This is sequential to ensure order. Ideally backend should support batch add.
      for (const uri of uris) {
        await axios.post(`${API_BASE}/playlist/add/${encodeURIComponent(uri)}`)
      }

      await fetchPlaylist()
      
      if (mode === 'append') {
        return // Done
      }

      const newLength = playlist.value.length
      const addedCount = newLength - startLength

      if (addedCount === 0) return

      if (mode === 'next') {
        // Move added tracks to after current song
        // Added tracks are at [startLength, ..., newLength - 1]
        // Target is currentPos + 1
        let target = currentPos + 1
        
        // We need to move them one by one. 
        // Note: Moving an item shifts indices.
        // If we move the last item to target, it shifts everything down.
        // Let's iterate and move each added track to target + i
        
        // Strategy:
        // We have tracks at the end.
        // Move the first added track (at startLength) to target.
        // Now the second added track is at startLength + 1? No, it shifted?
        // Wait, if startLength=100, target=5.
        // Move 100 -> 5. 100 is now at 5. Old 5 is at 5? No, old 5 is at 4? Wait.
        // Insert AT target.
        // If I move 100 to 5. Item at 5 shifts to 6.
        // So effectively, I want to move startLength to target.
        // Then startLength + 1 (which is now at startLength + 1 because we inserted before it? No.)
        // Let's look at indexes.
        // [A, B, C ... X, Y, Z] (Length L)
        // Add [1, 2]. -> [A..Z, 1, 2]. (1 at L, 2 at L+1).
        // Move 1 (from L) to 5. -> [A..4, 1, 5..Z, 2].
        // Now 2 is at L+1.
        // Move 2 (from L+1) to 6. -> [A..4, 1, 2, 5..Z].
        // So yes, I can just move the calculated indices.
        
        // However, we must be careful if target > startLength (e.g. playing near end).
        // If playing at 99 (of 100). Add 2 -> 102.
        // Target 100.
        // Move 100 to 100? No-op.
        // Move 101 to 101? No-op.
        // It works naturally.

        for (let i = 0; i < addedCount; i++) {
           // The track to move is always at startLength + i (initially).
           // BUT, after moving previous tracks, does it shift?
           // If we move from L to T (T < L).
           // Everything from T to L-1 shifts +1.
           // L+1 shifts? No. L+1 stays at L+1?
           // Actually, if we use IDs it's easier, but we use Pos.
           
           // Let's try separate moves.
           // We are moving `startLength + i` to `target + i`.
            
           // Example: [0, 1, 2, 3]. Curr=1. Target=2.
           // Add [A, B]. -> [0, 1, 2, 3, A, B]. (A at 4, B at 5).
           // Move A (4) to 2.
           // Result: [0, 1, A, 2, 3, B]. (2->3, 3->4, B->5).
           // B is still at 5.
           // Move B (5) to 3.
           // Result: [0, 1, A, B, 2, 3].
           // DONE.
           
           // So yes, `move startLength + i` to `target + i` works IF target <= startLength.
           // IF target > startLength (impossible if adding to end, unless startLength IS target).
           
           await axios.post(`${API_BASE}/playlist/move`, { from: startLength + i, to: target + i })
        }
      } else if (mode === 'play') {
          // Play the first added track.
          // It is at startLength.
          await playTrack(startLength)
      }
      
      await fetchPlaylist()
    } catch (error) {
      console.error('Add tracks failed:', error)
      throw error
    }
  }

  const playTrack = async (pos) => {
    try {
      await axios.post(`${API_BASE}/play/${pos}`)
    } catch (error) {
      console.error('Play track failed:', error)
      throw error
    }
  }

  const triggerStreamingSearch = (query, exact = false, category = '') => {
    if (!searchWs.value || searchWs.value.readyState !== WebSocket.OPEN) {
      connectSearchWebSocket()
      if (!searchWs.value || searchWs.value.readyState !== WebSocket.OPEN) {
        setTimeout(() => triggerStreamingSearch(query, exact, category), 500)
        return
      }
    }
    
    if (!searchWs.value || searchWs.value.readyState !== WebSocket.OPEN) return
    
    searchResults.value = {
      albums: [],
      artists: [],
      genres: [],
      dates: [],
      songs: []
    }
    isSearching.value = true
    
    searchWs.value.send(JSON.stringify({
      type: 'search',
      query: query,
      exact: exact,
      category: category
    }))
  }

  const getConfig = async () => {
    try {
      const response = await axios.get(`${API_BASE}/config`)
      if (response.data.success) {
        config.value = response.data.data
      }
      return response.data
    } catch (error) {
      console.error('Get config failed:', error)
      throw error
    }
  }

  const updateConfig = async (newConfig) => {
    try {
      const configToSend = {
        mpdHost: newConfig.mpdHost || newConfig.host || newConfig.MPDHost,
        mpdPort: newConfig.mpdPort || newConfig.port || newConfig.MPDPort,
        mpdPassword: newConfig.mpdPassword || newConfig.password || newConfig.MPDPassword,
        musicRoot: newConfig.musicRoot,
        coverArtRoot: newConfig.coverArtRoot,
        coverArtBaseUrl: newConfig.coverArtBaseUrl,
        discogsToken: newConfig.discogsToken,
        albumArtApiKey: newConfig.albumArtApiKey,
        rsyncRemoteTarget: newConfig.rsyncRemoteTarget,
        rsyncOptions: newConfig.rsyncOptions,
        enableActivityRefresh: newConfig.enableActivityRefresh,
        musicBrainzEnabled: newConfig.musicBrainzEnabled,
        discogsEnabled: newConfig.discogsEnabled,
        freeDbEnabled: newConfig.freeDbEnabled, // Legacy support
        gnuDbEnabled: newConfig.gnuDbEnabled,
        albumArtEnabled: newConfig.albumArtEnabled
      }
      
      const response = await axios.post(`${API_BASE}/config`, configToSend)
      return response.data
    } catch (error) {
      console.error('Update config failed:', error)
      throw error
    }
  }

  const disconnect = () => {
    if (ws.value) {
      ws.value.close()
      ws.value = null
    }
    if (searchWs.value) {
      searchWs.value.close()
      searchWs.value = null
    }
    isConnected.value = false
  }

  // Cache management functions
  const getCacheStats = () => albumCache.getStats()
  
  const clearCache = () => {
    albumCache.clear()
    console.log('[MPD Store] Cache cleared')
  }
  
  const invalidateAlbumCache = (artist, album) => {
    albumCache.invalidate(artist, album)
    console.log('[MPD Store] Cache invalidated for:', artist, '-', album)
  }

  // Cover Art functions
  const fetchCoverArtCandidates = async (artist, album) => {
    try {
      const params = new URLSearchParams({
        artist: artist,
        album: album
      })
      const response = await axios.get(`${API_BASE}/coverart/candidates?${params}`)
      return response.data
    } catch (error) {
      console.error('Fetch cover art candidates failed:', error)
      throw error
    }
  }

  const applyCoverArt = async (albumPath, imageUrl) => {
    try {
      const response = await axios.post(`${API_BASE}/coverart/apply`, {
        albumPath: albumPath,
        imageUrl: imageUrl
      })
      return response.data
    } catch (error) {
      console.error('Apply cover art failed:', error)
      throw error
    }
  }

   // Metadata search functions
   const fetchMetadataCandidates = async (artist, album, providers = []) => {
     try {
       const params = new URLSearchParams({
         artist: artist,
         album: album
       })
       if (providers.length > 0) {
         params.append('providers', providers.join(','))
       }
       const response = await axios.get(`${API_BASE}/metadata/search?${params}`)
       return response.data
     } catch (error) {
       console.error('Fetch metadata candidates failed:', error)
       throw error
     }
   }

   const fetchMetadataDetails = async (source, externalId) => {
     try {
       const params = new URLSearchParams({
         source: source,
         externalId: externalId
       })
       const response = await axios.get(`${API_BASE}/metadata/details?${params}`)
       return response.data
     } catch (error) {
       console.error('Fetch metadata details failed:', error)
       throw error
     }
   }

   const applyMetadata = async (albumPath, metadata, coverArtUrl = '') => {
     try {
       const response = await axios.post(`${API_BASE}/metadata/apply`, {
         albumPath: albumPath,
         metadata: metadata,
         coverArtUrl: coverArtUrl
       })
       return response.data
     } catch (error) {
       console.error('Apply metadata failed:', error)
       throw error
     }
   }

  // Polling for status updates (fallback for when WebSocket fails)
  const startPolling = () => {
    if (pollInterval.value) {
      console.log('[MPD Store] Polling already active, skipping')
      return
    }
    
    console.log('[MPD Store] Starting status polling')
    
    // Poll every 30 seconds as fallback
    pollInterval.value = setInterval(async () => {
      try {
        await refreshStatus()
      } catch (error) {
        console.error('[MPD Store] Polling refresh failed:', error)
      }
    }, 30000)
  }

  const stopPolling = () => {
    if (pollInterval.value) {
      console.log('[MPD Store] Stopping status polling')
      clearInterval(pollInterval.value)
      pollInterval.value = null
    }
  }

  // Sync functions
  const getSyncStatus = async () => {
    try {
      const response = await axios.get(`${API_BASE}/sync/status`)
      return response.data
    } catch (error) {
      console.error('Get sync status failed:', error)
      throw error
    }
  }

  const startSync = async () => {
    try {
      const response = await axios.post(`${API_BASE}/sync/start`)
      return response.data
    } catch (error) {
      console.error('Start sync failed:', error)
      throw error
    }
  }

  return {
    // State
    status,
    playlist,
    playlistCurrentPos,
    isConnected,
    connectionError,
    config,
    searchResults,
    isSearching,
    
    // Getters
    currentSong,
    isPlaying,
    currentTime,
    duration,
    volume,
    
    // Actions
    connect,
    disconnect,
    refreshStatus,
    play,
    pause,
    next,
    previous,
    setVolume,
    fetchAlbums,
    fetchRandomAlbums,
    fetchArtists,
    fetchAlbumsByDate,
    fetchAlbumsByGenre,
    fetchAlbumSongs,
    search,
    triggerStreamingSearch,
    addToPlaylist,
    addAlbumToPlaylist,
    removeFromPlaylist,
    fetchPlaylist,
    moveTrack,
    moveAlbum,
    addTracks,
    playTrack,
    getConfig,
    updateConfig,
    getCacheStats,
    clearCache,
    invalidateAlbumCache,
     fetchCoverArtCandidates,
     applyCoverArt,
     fetchMetadataCandidates,
     fetchMetadataDetails,
     applyMetadata,
    startPolling,
    stopPolling,
    getSyncStatus,
    startSync
  }
})