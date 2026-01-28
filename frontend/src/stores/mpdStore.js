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
      
      ws.value = new WebSocket(`ws://${window.location.host}/ws`)
      
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

      searchWs.value = new WebSocket(`ws://${window.location.host}/ws/search`)
      
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
      
      if (response.data.success) {
        // Cache the response
        albumCache.set(artist, album, response.data.data)
        return response.data.data
      }
      
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
        rsyncRemoteTarget: newConfig.rsyncRemoteTarget,
        rsyncOptions: newConfig.rsyncOptions,
        enableActivityRefresh: newConfig.enableActivityRefresh
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
    playTrack,
    getConfig,
    updateConfig,
    getCacheStats,
    clearCache,
    invalidateAlbumCache
  }
})