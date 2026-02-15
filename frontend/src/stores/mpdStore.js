import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import axios from 'axios'
import { albumCache } from '@/services/albumCache'
import { n50Service } from '@/services/n50'

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
  
  // Database update notification
  const lastDatabaseUpdate = ref(null)
  
  // All albums for matrix generation (Phase 1)
  const allAlbums = ref([])
  const isLoadingAlbums = ref(false)
  
  // Search state
  const searchResults = ref({
    albums: [],
    artists: [],
    genres: [],
    dates: [],
    songs: []
  })
  const isSearching = ref(false)

  // N50 State
  const n50Status = ref(null)
  const n50Inputs = ref([])
  const isLoadingN50 = ref(false)

  // Computed: Genre/Date Matrix (Phase 1)
  const genreDateMatrix = computed(() => {
    if (!allAlbums.value || allAlbums.value.length === 0) {
      return { genres: [], dates: [], matrix: {} }
    }
    
    const matrix = {}
    const allGenres = new Set()
    const allDates = new Set()
    
    // Build matrix from albums
    for (const album of allAlbums.value) {
      const genre = album.genre || 'Unknown'
      const date = album.date ? String(album.date).substring(0, 4) : 'Unknown'
      
      allGenres.add(genre)
      allDates.add(date)
      
      if (!matrix[genre]) {
        matrix[genre] = {}
      }
      matrix[genre][date] = (matrix[genre][date] || 0) + 1
    }
    
    // Sort genres alphabetically
    const genres = Array.from(allGenres).sort()
    
    // Sort dates descending (newest first)
    const dates = Array.from(allDates).sort((a, b) => b.localeCompare(a))
    
    return { genres, dates, matrix }
  })

  // Getters
  const currentSong = computed(() => status.value?.currentSong)
  const isPlaying = computed(() => status.value?.state === 'play')
  const currentTime = computed(() => status.value?.elapsed || 0)
  const duration = computed(() => status.value?.duration || 0)
  const volume = computed(() => status.value?.volume || 0)
  const isN50Enabled = computed(() => config.value?.n50Enabled || false)

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
      
      // Load all albums for matrix (Phase 1)
      loadAllAlbums()
    } catch (error) {
      console.error('Failed to connect to MPD:', error)
      isConnected.value = false
      connectionError.value = error.message
    }
  }

  // Load all albums for matrix generation (Phase 1)
  const loadAllAlbums = async () => {
    if (isLoadingAlbums.value) return
    
    isLoadingAlbums.value = true
    try {
      console.log('[MPD Store] Loading all albums for matrix...')
      const response = await axios.get(`${API_BASE}/albums/all`)
      if (response.data.success) {
        allAlbums.value = response.data.data || []
        console.log(`[MPD Store] Loaded ${allAlbums.value.length} albums`)
      }
    } catch (error) {
      console.error('[MPD Store] Failed to load all albums:', error)
    } finally {
      isLoadingAlbums.value = false
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
          } else if (msg.type === 'database_update') {
            // Handle database update notification
            console.log('[MPD Store] Database update received:', msg.data)
            lastDatabaseUpdate.value = msg.data
            
            // Clear all caches
            albumCache.clear()
            console.log('[MPD Store] Album cache cleared due to database update')
            
            // Reload all albums for matrix (Phase 1)
            loadAllAlbums()
            
            // Dispatch custom event for components to listen to
            window.dispatchEvent(new CustomEvent('database-updated', { detail: msg.data }))
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

  const fetchGenreDateMatrix = async () => {
    try {
      const response = await axios.get(`${API_BASE}/genres/matrix`)
      return response.data
    } catch (error) {
      console.error('Fetch genre/date matrix failed:', error)
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
      let cached = null
      if (!forceRefresh) {
        cached = albumCache.get(artist, album)
        if (cached && !cached.isStale) {
          console.log('[MPD Store] Cache hit (fresh):', artist, '-', album)
          // Merge overlay if present
          const overlay = albumCache.getOverlay(artist, album)
          if (overlay) {
            console.log('[MPD Store] Applying overlay to cached data')
            return { ...cached.data, ...overlay, isOverlayActive: true }
          }
          return cached.data
        }
      }

      // Cache miss or force refresh or stale - fetch from API
      console.log('[MPD Store] Fetching from API:', artist, '-', album)
      const response = await axios.get(`${API_BASE}/album/${encodeURIComponent(artist)}/${encodeURIComponent(album)}?${params}`)
      
      if (response.data.success) {
        const serverData = response.data.data
        const overlay = albumCache.getOverlay(artist, album)
        
        if (overlay) {
          // Check if server data is now up to date with overlay
          if (isDataSynced(serverData, overlay)) {
            console.log('[MPD Store] Server caught up with overlay, clearing overlay')
            albumCache.clearOverlay(artist, album)
            albumCache.set(artist, album, serverData)
            return serverData
          } else {
            console.log('[MPD Store] Server still out of sync with overlay, using overlay')
            albumCache.set(artist, album, serverData)
            return { ...serverData, ...overlay, isOverlayActive: true }
          }
        }

        albumCache.set(artist, album, serverData)
        return serverData
      }
      
      return response.data
    } catch (error) {
      console.error('Fetch album songs failed:', error)
      throw error
    }
  }

  const isDataSynced = (serverData, overlay) => {
    // Basic comparison logic for sync detection
    if (overlay.album && serverData.album !== overlay.album) return false
    if (overlay.artist && serverData.artist !== overlay.artist) return false
    if (overlay.year && serverData.year !== overlay.year) return false
    if (overlay.genre && serverData.genre !== overlay.genre) return false
    // Track matching could be more complex, skipping for brevity
    return true
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
        let target = currentPos + 1
        for (let i = 0; i < addedCount; i++) {
           await axios.post(`${API_BASE}/playlist/move`, { from: startLength + i, to: target + i })
        }
      } else if (mode === 'play') {
          await playTrack(startLength)
      }
      
      await fetchPlaylist()
    } catch (error) {
      console.error('Add tracks failed:', error)
      throw error
    }
  }

  // Aggressive refresh after user actions for immediate UI feedback
  const aggressiveRefresh = async () => {
    console.log('[MPD Store] Aggressive refresh triggered after user action')
    try {
      // Fetch both status and playlist in parallel for speed
      const [statusResponse, playlistResponse] = await Promise.all([
        axios.get(`${API_BASE}/status`),
        axios.get(`${API_BASE}/playlist`)
      ])
      
      if (statusResponse.data.data) {
        status.value = statusResponse.data.data
        if (statusResponse.data.data.playlistPos !== undefined) {
          playlistCurrentPos.value = statusResponse.data.data.playlistPos
        }
      }
      
      if (playlistResponse.data.success) {
        playlist.value = playlistResponse.data.data.items
        playlistCurrentPos.value = playlistResponse.data.data.currentPos
      }
      
      console.log('[MPD Store] Aggressive refresh complete')
    } catch (error) {
      console.error('[MPD Store] Aggressive refresh failed:', error)
    }
  }

  const playTrack = async (pos) => {
    try {
      await axios.post(`${API_BASE}/play/${pos}`)
      // Trigger aggressive refresh for immediate UI update
      await aggressiveRefresh()
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
        discogsKey: newConfig.discogsKey,
        discogsSecret: newConfig.discogsSecret,
        albumArtApiKey: newConfig.albumArtApiKey,
        rsyncRemoteTarget: newConfig.rsyncRemoteTarget,
        rsyncOptions: newConfig.rsyncOptions,
        randomAlbumCount: newConfig.randomAlbumCount,
        enableActivityRefresh: newConfig.enableActivityRefresh,
        musicBrainzEnabled: newConfig.musicBrainzEnabled,
        discogsEnabled: newConfig.discogsEnabled,
        freeDbEnabled: newConfig.freeDbEnabled, // Legacy support
        gnuDbEnabled: newConfig.gnuDbEnabled,
        albumArtEnabled: newConfig.albumArtEnabled,
        n50Enabled: newConfig.n50Enabled,
        n50Host: newConfig.n50Host,
        n50Port: newConfig.n50Port,
        n50Input: newConfig.n50Input,
        n50AutoControl: newConfig.n50AutoControl,
        n50IgnoreOnStart: newConfig.n50IgnoreOnStart
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

  const uploadCoverArt = async (albumPath, file) => {
    try {
      const formData = new FormData()
      formData.append('albumPath', albumPath)
      formData.append('cover', file)
      const response = await axios.post(`${API_BASE}/coverart/upload`, formData, {
        headers: {
          'Content-Type': 'multipart/form-data'
        }
      })
      return response.data
    } catch (error) {
      console.error('Upload cover art failed:', error)
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
       if (response.data.success) {
         // Create overlay from metadata
         const overlay = {
           album: metadata.album,
           artist: metadata.artist,
           year: metadata.year,
           genre: metadata.genre,
           tracks: metadata.tracks,
           originalMetadata: metadata // Store full candidate for reapply
         }
         albumCache.setOverlay(metadata.artist, metadata.album, overlay)
       }
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

  // N50 Actions
  const fetchN50Status = async () => {
    try {
      isLoadingN50.value = true
      const response = await n50Service.getStatus()
      if (response.success) {
        n50Status.value = response.data
      }
      return response
    } catch (error) {
      console.error('Fetch N50 status failed:', error)
      throw error
    } finally {
      isLoadingN50.value = false
    }
  }

  const fetchN50Inputs = async () => {
    try {
      const response = await n50Service.getAvailableInputs()
      if (response.success) {
        n50Inputs.value = response.data.inputs || []
      }
      return response
    } catch (error) {
      console.error('Fetch N50 inputs failed:', error)
      throw error
    }
  }

  const n50PowerOn = async () => {
    try {
      const response = await n50Service.powerOn()
      await fetchN50Status()
      return response
    } catch (error) {
      console.error('N50 power on failed:', error)
      throw error
    }
  }

  const n50PowerOff = async () => {
    try {
      const response = await n50Service.powerOff()
      await fetchN50Status()
      return response
    } catch (error) {
      console.error('N50 power off failed:', error)
      throw error
    }
  }

  const n50SetInput = async (input) => {
    try {
      const response = await n50Service.setInput(input)
      await fetchN50Status()
      return response
    } catch (error) {
      console.error('N50 set input failed:', error)
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
    n50Status,
    n50Inputs,
    isLoadingN50,
    lastDatabaseUpdate,
    allAlbums,
    isLoadingAlbums,
    
    // Getters
    currentSong,
    isPlaying,
    currentTime,
    duration,
    volume,
    isN50Enabled,
    genreDateMatrix,
    
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
    startSync,
    fetchN50Status,
    fetchN50Inputs,
    n50PowerOn,
    n50PowerOff,
    n50SetInput,
    fetchGenreDateMatrix,
    loadAllAlbums
  }
})
