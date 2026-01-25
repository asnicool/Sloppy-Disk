import { ref, computed, reactive, watch } from 'vue'
import axios from 'axios'

// Reactive store state
const storeState = reactive({
  status: null,
  playlist: [],
  playlistCurrentPos: -1,
  isConnected: false,
  ws: null,
  connectionError: null,
  localElapsed: 0,
  lastStatusTime: 0,
  lastStatusTime: 0,
  pollInterval: null,
  config: null,
  lastActivityRefreshTime: 0,
  currentBackoffIndex: 0,
  backoffLevels: [10000, 60000, 600000] // 10s, 1m, 10m
})

// Time tracking for smooth progress bar (client-side only)
// Note: These are now part of storeState, use storeState.localElapsed etc.

// Streaming Search Results
const searchResults = ref({
  albums: [],
  artists: [],
  genres: [],
  dates: [],
  songs: []
})
const isSearching = ref(false)

// API base URL
const API_BASE = '/api'

// Computed properties
const currentSong = computed(() => storeState.status?.currentSong)
const isPlaying = computed(() => storeState.status?.state === 'play')
const currentTime = computed(() => storeState.status?.elapsed || 0)
const duration = computed(() => storeState.status?.duration || 0)
const volume = computed(() => storeState.status?.volume || 0)

// Actions
const connect = async () => {
  try {
    console.log('[MPD] Connecting...')
    // Test connection with status endpoint
    const response = await axios.get(`${API_BASE}/status`)
    console.log('[MPD] Status response:', response.data)
    storeState.status = response.data.data
    storeState.isConnected = true
    console.log('[MPD] Status set:', storeState.status)
    
    // Connect WebSocket for real-time updates
    connectWebSocket()
  } catch (error) {
    console.error('Failed to connect to MPD:', error)
    storeState.isConnected = false
    storeState.connectionError = error.message
  }
}

// Function to manually refresh status (for explicit user requests)
const refreshStatus = async () => {
  try {
    // Use the new refresh endpoint which will broadcast to all WebSocket clients
    const response = await axios.post(`${API_BASE}/status/refresh`)
    // The status will be updated via WebSocket push
    return response.data
  } catch (error) {
    console.error('Failed to refresh MPD status:', error)
    throw error
  }
}

const connectWebSocket = () => {
  // Prevent multiple concurrent connections
  if (storeState.ws && storeState.ws.readyState === WebSocket.OPEN) {
    console.log('WebSocket already connected, skipping')
    return
  }
  if (storeState.ws && storeState.ws.readyState === WebSocket.CONNECTING) {
    console.log('WebSocket already connecting, skipping')
    return
  }
  
  try {
    // Close any existing ws connection
    if (storeState.ws) {
      storeState.ws.close()
      storeState.ws = null
    }
    
    storeState.ws = new WebSocket(`ws://${window.location.host}/ws`)
    
    storeState.ws.onopen = () => {
      console.log('WebSocket connected')
      // Request current status to ensure we have the latest data
      if (storeState.ws && storeState.ws.readyState === WebSocket.OPEN) {
        storeState.ws.send(JSON.stringify({ type: 'get_status' }))
      }
    }
    
    storeState.ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data)
        if (msg.type === 'status' && msg.data) {
          // Only update if we have valid status data
          if (msg.data.state !== undefined) {
            storeState.status = msg.data
            // Sync playlistCurrentPos from status if available
            if (msg.data.playlistPos !== undefined) {
              storeState.playlistCurrentPos = msg.data.playlistPos
            }
            console.log('Status updated via WebSocket:', storeState.status?.state, storeState.status?.currentSong?.title, 'playlist version:', storeState.status?.playlistVersion)
          }
        } else if (msg.type === 'search_results') {
          const { category, items } = msg.data
          if (searchResults.value[category]) {
            searchResults.value[category] = items
          }
        }
      } catch (error) {
        console.error('WebSocket message parse error:', error)
      }
    }
    
    storeState.ws.onclose = () => {
      console.log('WebSocket disconnected')
      storeState.ws = null
      // Reconnect after delay if still connected
      if (storeState.isConnected) {
        setTimeout(() => {
          if (storeState.isConnected && (!storeState.ws || storeState.ws.readyState !== WebSocket.OPEN)) {
            connectWebSocket()
          }
        }, 5000)
      }
    }
    
    storeState.ws.onerror = (error) => {
      console.error('WebSocket error:', error)
    }

    // Note: Backend manages keep-alive via WebSocket Pings every 4.5 minutes.
    // Browsers automatically respond with Pongs, keeping the connection alive.

    // Load initial config to know if activity refresh is enabled
    getConfig()
  } catch (error) {
    console.error('Failed to connect WebSocket:', error)
  }
}

// Playback controls
const play = async () => {
  try {
    await axios.post(`${API_BASE}/play`)
    // Status will update via WebSocket
  } catch (error) {
    console.error('Play failed:', error)
  }
}

const pause = async () => {
  try {
    await axios.post(`${API_BASE}/pause`)
    // Status will update via WebSocket
  } catch (error) {
    console.error('Pause failed:', error)
  }
}

const next = async () => {
  try {
    await axios.post(`${API_BASE}/next`)
    // Status will update via WebSocket
  } catch (error) {
    console.error('Next failed:', error)
  }
}

const previous = async () => {
  try {
    await axios.post(`${API_BASE}/previous`)
    // Status will update via WebSocket
  } catch (error) {
    console.error('Previous failed:', error)
  }
}

const setVolume = async (newVolume) => {
  try {
    await axios.post(`${API_BASE}/volume/${newVolume}`)
    // Status will update via WebSocket
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
    
    if (search) {
      params.append('search', search)
    }
    if (sort) {
      params.append('sort', sort)
    }

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
    if (count) {
      params.append('count', count.toString())
    }
    if (refresh) {
      params.append('refresh', 'true')
    }
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
    
    if (search) {
      params.append('search', search)
    }

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

const enhancedSearch = async (query, page = 1, limit = 30) => {
  try {
    const params = new URLSearchParams({
      q: query,
      page: page.toString(),
      limit: limit.toString()
    })

    const response = await axios.get(`${API_BASE}/search/enhanced?${params}`)
    return response.data
  } catch (error) {
    console.error('Enhanced search failed:', error)
    throw error
  }
}

const fetchAlbumSongs = async (artist, album, page = 1, limit = 50) => {
  try {
    const params = new URLSearchParams({
      page: page.toString(),
      limit: limit.toString()
    })

    const response = await axios.get(`${API_BASE}/album/${encodeURIComponent(artist)}/${encodeURIComponent(album)}?${params}`)
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

const triggerStreamingSearch = (query, exact = false, category = '') => {
  if (!storeState.ws || storeState.ws.readyState !== WebSocket.OPEN) return
  
  // Reset results if no category or all categories
  if (!category) {
    searchResults.value = {
      albums: [],
      artists: [],
      genres: [],
      dates: [],
      songs: []
    }
  } else {
    // If specific category, only reset that one (or maybe reset all to be safe? 
    // Usually a category search is a fresh start)
    searchResults.value = {
      albums: [],
      artists: [],
      genres: [],
      dates: [],
      songs: []
    }
  }
  isSearching.value = true
  
  storeState.ws.send(JSON.stringify({
    type: 'search',
    query: query,
    exact: exact,
    category: category
  }))
}

const fetchMetadataCandidates = async (artist, album) => {
  try {
    const response = await axios.get(`${API_BASE}/metadata/search?artist=${encodeURIComponent(artist)}&album=${encodeURIComponent(album)}`)
    return response.data
  } catch (error) {
    console.error('Fetch metadata candidates failed:', error)
    throw error
  }
}

const fetchCoverArtCandidates = async (artist, album) => {
  try {
    const response = await axios.get(`${API_BASE}/coverart/candidates?artist=${encodeURIComponent(artist)}&album=${encodeURIComponent(album)}`)
    return response.data
  } catch (error) {
    console.error('Fetch cover art candidates failed:', error)
    throw error
  }
}

const applyCoverArt = async (albumPath, imageUrl) => {
  try {
    await axios.post(`${API_BASE}/coverart/apply`, { albumPath, imageUrl })
  } catch (error) {
    console.error('Apply cover art failed:', error)
    throw error
  }
}

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
    await axios.post(`${API_BASE}/sync/start`)
  } catch (error) {
    console.error('Start sync failed:', error)
    throw error
  }
}

const getConfig = async () => {
  try {
    const response = await axios.get(`${API_BASE}/config`)
    if (response.data.success) {
      storeState.config = response.data.data
    }
    return response.data
  } catch (error) {
    console.error('Get config failed:', error)
    throw error
  }
}

const updateConfig = async (config) => {
  try {
    // Ensure the config object has the correct field names for the backend
    const configToSend = {
      mpdHost: config.mpdHost || config.host || config.MPDHost,
      mpdPort: config.mpdPort || config.port || config.MPDPort,
      mpdPassword: config.mpdPassword || config.password || config.MPDPassword,
      musicRoot: config.musicRoot,
      coverArtRoot: config.coverArtRoot,
      discogsToken: config.discogsToken,
      rsyncRemoteTarget: config.rsyncRemoteTarget,
      rsyncOptions: config.rsyncOptions,
      enableActivityRefresh: config.enableActivityRefresh
    };
    
    const response = await axios.post(`${API_BASE}/config`, configToSend)
    return response.data
  } catch (error) {
    console.error('Update config failed:', error)
    throw error
  }
}

// Function to enrich a list of albums with metadata (covers, etc.)
const enrichAlbums = async (albums) => {
  try {
    const response = await axios.post(`${API_BASE}/albums/enrich`, { albums })
    if (response.data.success) {
      return response.data.data
    }
    return albums
  } catch (error) {
    console.error('Enrich albums failed:', error)
    return albums
  }
}

// Playlist functions
const fetchPlaylist = async () => {
  try {
    console.log('[MPD Store] Fetching playlist...')
    const response = await axios.get(`${API_BASE}/playlist`)
    if (response.data.success) {
      console.log('[MPD Store] Playlist fetched, items:', response.data.data.items.length)
      storeState.playlist = response.data.data.items
      storeState.playlistCurrentPos = response.data.data.currentPos
    }
    return response.data
  } catch (error) {
    console.error('Fetch playlist failed:', error)
    throw error
  }
}

// Playlist functions
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
    // Status will update via WebSocket
  } catch (error) {
    console.error('Play track failed:', error)
    throw error
  }
}

// Polling for accurate progress (every 10 seconds)
const startPolling = () => {
  if (storeState.pollInterval) return
  
  // Initial refresh
  refreshStatus()
  
  storeState.pollInterval = setInterval(() => {
    if (storeState.isConnected && storeState.status?.state === 'play') {
      refreshStatus()
    }
  }, 10000)
}

const stopPolling = () => {
  if (storeState.pollInterval) {
    clearInterval(storeState.pollInterval)
    storeState.pollInterval = null
  }
}

// Handle visibility change for power optimization
const handleVisibilityChange = () => {
  if (document.hidden) {
    console.log('[MPD Store] App hidden, stopping polling')
    stopPolling()
  } else {
    // When returning to the app, always refresh everything immediately
    // "no need to bother with disablement and reeanblement" for visibility change
    console.log('[MPD Store] App visible, immediate refresh')
    refreshStatus()
    fetchPlaylist()
    if (storeState.status?.state === 'play') {
      startPolling()
    }
  }
}

// Activity detection with tiered backoff
const handleActivity = () => {
  if (!storeState.config?.enableActivityRefresh) {
    return
  }

  const now = Date.now()
  const timeSinceLastRefresh = now - storeState.lastActivityRefreshTime
  const currentBackoff = storeState.backoffLevels[storeState.currentBackoffIndex]

  // If we are currently in a "disabled" period, skip
  if (timeSinceLastRefresh < currentBackoff) {
    return
  }

  // Refresh immediately upon detecting activity after a "disabled" period
  console.log('[MPD Store] Activity detected (scroll/touch/click), refreshing...')
  refreshStatus()
  fetchPlaylist()
  
  const oldBackoffIndex = storeState.currentBackoffIndex
  const timeSinceReenablement = timeSinceLastRefresh - currentBackoff
  
  // Update backoff strategy:
  // "The detection is re-disabled if some activity is detected within 10 secs after last re-enablement."
  if (timeSinceReenablement < 10000 && storeState.lastActivityRefreshTime > 0) {
    // Activity happened shortly after re-enablement, increase backoff
    if (storeState.currentBackoffIndex < storeState.backoffLevels.length - 1) {
      storeState.currentBackoffIndex++
      console.log(`[MPD Store] Backoff increased to ${storeState.backoffLevels[storeState.currentBackoffIndex]/1000}s`)
    }
  } else if (timeSinceReenablement > 600000) {
    // Reset backoff if 10 mins passed since re-enablement without activity detection
    storeState.currentBackoffIndex = 0
    console.log('[MPD Store] Backoff reset to 10s')
  }

  storeState.lastActivityRefreshTime = now
}

// Listen for visibility and activity
if (typeof document !== 'undefined') {
  document.addEventListener('visibilitychange', handleVisibilityChange)
  
  // Activity events
  const activityEvents = ['scroll', 'touchstart', 'mousedown', 'keydown']
  activityEvents.forEach(event => {
    // Use passive true for scroll/touchstart to avoid performance warnings
    window.addEventListener(event, handleActivity, { passive: true })
  })
}

// Cleanup
const disconnect = () => {
  if (storeState.ws) {
    storeState.ws.close()
    storeState.ws = null
  }
  storeState.isConnected = false
}

// Watch for playlist version changes to sync across clients
watch(() => storeState.status?.playlistVersion, (newVal, oldVal) => {
  console.log('[MPD Store] Playlist watcher triggered (version):', newVal, 'old:', oldVal)
  // Trigger on any change, including first one if it's not null
  if (newVal !== undefined && newVal !== oldVal) {
    console.log('[MPD Store] Playlist version changed (remote update), refreshing...')
    fetchPlaylist()
  }
})

// Export the store interface
export function useMpdStore() {
  return reactive({
    // State from reactive store
    get status() { return storeState.status },
    get playlist() { return storeState.playlist },
    get playlistCurrentPos() { return storeState.playlistCurrentPos },
    get isConnected() { return storeState.isConnected },
    get connectionError() { return storeState.connectionError },
    get localElapsed() { return storeState.localElapsed },
    
    // Computed
    currentSong,
    isPlaying,
    currentTime,
    duration,
    volume,
    
    // Actions
    connect,
    disconnect,
    play,
    pause,
    next,
    previous,
    setVolume,
    refreshStatus,
    
    // Data fetching
    fetchAlbums,
    fetchRandomAlbums,
    fetchArtists,
    fetchAlbumsByDate,
    fetchAlbumsByGenre,
    enhancedSearch,
    fetchAlbumSongs,
    search,
    triggerStreamingSearch,
    searchResults,
    isSearching,
    fetchMetadataCandidates,
    fetchCoverArtCandidates,
    applyCoverArt,
    getSyncStatus,
    startSync,
    getConfig,
    updateConfig,
    addToPlaylist,
    addAlbumToPlaylist,
    removeFromPlaylist,
    fetchPlaylist,
    addToPlaylist,
    addAlbumToPlaylist,
    removeFromPlaylist,
    moveTrack,
    moveAlbum,
    fetchPlaylist,
    playTrack,
    startPolling,
    stopPolling,
    enrichAlbums
  })
}
