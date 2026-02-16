<template>
  <div 
    ref="cardRef" 
    class="album-card group bg-neutral-800/40 rounded-xl overflow-hidden border border-neutral-700/50 hover:border-primary-500/50 transition-all duration-300 flex flex-col h-full"
    :class="{ 'ring-2 ring-primary-500/20': expanded }"
  >
    <!-- Cover Image Section -->
    <div 
      class="relative aspect-square overflow-hidden"
      :style="{ backgroundColor: bgColor }"
      @click="triggerOverlay"
    >
      <img 
        v-if="currentImageSrc" 
        :src="currentImageSrc" 
        :alt="album"
        class="w-full h-full object-cover transition-all duration-700 group-hover:scale-110"
        :class="{ 'opacity-0 scale-95': !imageLoaded, 'opacity-100 scale-100': imageLoaded }"
        @load="handleImageLoad"
        @error="handleImageError"
      />
      <!-- Default Icon (shown when loading or on final error) -->
      <div v-else class="w-full h-full flex items-center justify-center text-white/20">
        <svg class="w-16 h-16" fill="currentColor" viewBox="0 0 20 20">
          <path d="M18 3a1 1 0 00-1.196-.98l-10 2A1 1 0 006 5v9.114A4.369 4.369 0 005 14c-1.657 0-3 .895-3 2s1.343 2 3 2 3-.895 3-2V7.82l8-1.6v5.894A4.369 4.369 0 0015 12c-1.657 0-3 .895-3 2s1.343 2 3 2 3-.895 3-2V3z" />
        </svg>
      </div>

      <!-- Actions Overlay -->
      <div 
        class="absolute inset-0 bg-black/60 opacity-0 group-hover:opacity-100 transition-opacity backdrop-blur-sm grid grid-cols-2 grid-rows-2 p-4 gap-4 pointer-events-none group-hover:pointer-events-auto"
        :class="{ '!opacity-100 !pointer-events-auto': showOverlay }"
      >
        <!-- Top Left: Add to end -->
        <button 
          @click.stop="mpdStore.addAlbumToPlaylist(artist, album, 'append')"
          class="flex items-center justify-center text-white/70 hover:text-white hover:scale-110 transition-all"
          title="Add to end of playlist"
        >
          <svg class="w-10 h-10" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v3m0 0v3m0-3h3m-3 0H9m12 0a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        </button>

        <!-- Top Right: Insert after current -->
        <button 
          @click.stop="mpdStore.addAlbumToPlaylist(artist, album, 'insert')"
          class="flex items-center justify-center text-white/70 hover:text-white hover:scale-110 transition-all"
          title="Insert after current track"
        >
          <svg class="w-10 h-10" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 5l7 7-7 7M5 5l7 7-7 7" />
          </svg>
        </button>

        <!-- Bottom Left: Show details -->
        <button 
          @click.stop="navigateToDetails"
          class="flex items-center justify-center text-white/70 hover:text-white hover:scale-110 transition-all"
          title="Show details"
        >
          <svg class="w-10 h-10" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        </button>

        <!-- Bottom Right: Replace -->
        <button 
          @click.stop="mpdStore.addAlbumToPlaylist(artist, album, 'replace')"
          class="flex items-center justify-center text-white/70 hover:text-white hover:scale-110 transition-all"
          title="Replace playlist and play"
        >
          <svg class="w-10 h-10" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
        </button>
      </div>

      <!-- Loading Spinner for Background fetching -->
      <div v-if="loading && isVisible" class="absolute top-2 right-2">
        <div class="animate-spin h-4 w-4 border-2 border-primary-500 border-t-transparent rounded-full"></div>
      </div>
      <div v-if="isBusy && isVisible" class="absolute top-2 right-2 bg-yellow-500/20 text-yellow-500 text-[10px] px-1 rounded animate-pulse">
        BUSY
      </div>
    </div>

    <!-- Basic Info (always visible) & Lazy Metadata -->
    <div 
      class="p-4 flex-1 flex flex-col"
    >
      <div class="flex-1 min-w-0">
        <h3 class="text-sm font-bold text-gray-100 truncate group-hover:text-primary-400 transition-colors" :title="album">{{ album }}</h3>
        <p 
          class="text-xs text-gray-400 truncate mt-0.5 hover:text-primary-400 cursor-pointer transition-colors"
          @click.stop="filterBy('artist', artist)"
        >
          {{ artist }}
        </p>
      </div>

      <!-- Immediate Metadata (from props) -->
      <div v-if="date || genre" class="flex flex-wrap items-center gap-2">
        <span
          v-if="date"
          class="px-2 py-0.5 bg-neutral-700/50 text-[10px] text-neutral-300 rounded-md border border-neutral-600/30 hover:bg-neutral-600 cursor-pointer transition-colors"
          @click.stop="filterBy('date', date)"
        >
          {{ date }}
        </span>
        <span
          v-if="genre"
          class="px-2 py-0.5 max-w-[100px] truncate bg-neutral-700/50 text-[10px] text-neutral-300 rounded-md border border-neutral-600/30 hover:bg-neutral-600 cursor-pointer transition-colors"
          @click.stop="filterBy('genre', genre)"
          :title="genre"
        >
          {{ genre }}
        </span>
      </div>

      <!-- Lazy Loaded Additional Metadata (from fullDetails) -->
      <div v-if="fullDetails && (fullDetails.album.trackCount || fullDetails.album.duration)" class="mt-2 flex flex-wrap items-center gap-2 animate-in fade-in slide-in-from-bottom-1 duration-500">
        <button 
          @click.stop="navigateToDetails"
          class="px-2 py-0.5 bg-primary-500/20 text-[10px] text-primary-400 rounded-md border border-primary-500/30 hover:bg-primary-500/40 hover:text-white transition-all flex items-center font-bold gap-1"
        >
          #{{ fullDetails.album.trackCount || '?' }} / {{ formatDuration(fullDetails.album.duration) }}
          <button 
            @click.stop="openMetadataSearch"
            class="hover:scale-110 transition-transform"
            title="Click to find metadata"
          >
            <MetadataStatusBadge :album="{ name: album, artist, date, genre, coverUrl }" />
          </button>
        </button>
      </div>

    </div>

  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, watch, computed, onBeforeUnmount } from 'vue'
import { useRouter } from 'vue-router'
import { useMpdStore } from '@/stores/mpd'
import { generateHashColor } from '@/utils/color'
import { albumBatchLoader } from '@/services/albumBatchLoader'
import axios from 'axios'
import MetadataStatusBadge from './MetadataStatusBadge.vue'

const props = defineProps({
  album: {
    type: String,
    required: true
  },
  artist: {
    type: String,
    required: true
  },
  coverUrl: {
    type: String,
    default: ''
  },
  date: {
    type: String,
    default: ''
  },
  genre: {
    type: String,
    default: ''
  },
  albumDetails: {
    type: Object,
    default: null
  }
})

const emit = defineEmits(['open-metadata-search'])

const router = useRouter()
const mpdStore = useMpdStore()

const cardRef = ref(null)
const isVisible = ref(false)
const loading = ref(false)
const isBusy = ref(false)
const fullDetails = ref(null)
const showOverlay = ref(false)
const overlayTimer = ref(null)

const triggerOverlay = () => {
  showOverlay.value = true
  if (overlayTimer.value) clearTimeout(overlayTimer.value)
  overlayTimer.value = setTimeout(() => {
    showOverlay.value = false
  }, 3000)
}

const filterBy = (type, value) => {
  if (!value) return
  // Navigate to search with the filter query and type
  router.push({ 
    name: 'search', 
    query: { q: value, type: type, strict: 'true' } 
  })
}

// Smart initialization: Use what we have immediately
const initData = () => {
    if (props.albumDetails) {
        // If we have tracks in details (rare but possible), use them.
        // Otherwise structure it as expected: { album: props.albumDetails, tracks: [] }
        if (props.albumDetails.album && props.albumDetails.tracks) {
             fullDetails.value = props.albumDetails
        } else {
             fullDetails.value = {
                 album: props.albumDetails,
                 tracks: []
             }
        }
    }
}

const expanded = ref(false)
const imageLoaded = ref(false)
const currentImageSrc = ref('')
const usingFolderUrl = ref(false)
const folderUrlChecked = ref(false)

// Generate consistent background color
const bgColor = computed(() => {
  // Use what we have: Artist + Album + Date (if available)
  const dateSeed = fullDetails.value?.album?.date || ''
  return generateHashColor(props.artist + props.album + dateSeed)
})

// Build the /folder URL from album path
const buildFolderUrl = (path) => {
    if (!path) return null
    
    const lastSlash = path.lastIndexOf('/')
    if (lastSlash !== -1) {
        const dir = path.substring(0, lastSlash)
        // Encode the directory path components
        const encodedDir = dir.split('/').map(encodeURIComponent).join('/')
        return `/folder/${encodedDir}/Folder.jpg`
    }
    return null
}

// Check if a URL returns 404 (run in background)
const checkFolderExists = (url) => {
    fetch(url, { method: 'HEAD' })
        .then(response => {
            folderUrlChecked.value = true
            if (response.status === 404) {
                console.log('[AlbumCard] Folder URL not found (404), switching to API')
                fallbackToApi()
            }
            // If 200+, folder URL works, keep it
        })
        .catch(() => {
            folderUrlChecked.value = true
            // Network error - fallback to API
            fallbackToApi()
        })
}

// Resolve image immediately from props (fast path - no API call needed)
const resolveImageFromProps = () => {
    if (props.coverUrl) {
        console.log('[AlbumCard] Using props.coverUrl immediately:', props.coverUrl)
        currentImageSrc.value = props.coverUrl
        usingFolderUrl.value = false
        folderUrlChecked.value = true
        return true
    }
    return false
}

// Resolve the best image source - try props first (fast), then fullDetails (slow)
const resolveImageSource = () => {
    // 1. Try using props.coverUrl first (fastest - no API needed)
    if (resolveImageFromProps()) {
        return
    }

    // 2. Fall back to folder URL from fullDetails (requires API call)
    const path = fullDetails.value?.tracks?.length > 0
        ? fullDetails.value.tracks[0].path
        : fullDetails.value?.album?.path

    if (!path) {
        // No path available yet, try API coverUrl as fallback
        const fallbackUrl = fullDetails.value?.album?.coverUrl || ''
        if (fallbackUrl) {
            console.log('[AlbumCard] Using fullDetails coverUrl as fallback:', fallbackUrl)
            currentImageSrc.value = fallbackUrl
            usingFolderUrl.value = false
            folderUrlChecked.value = true
        }
        return
    }

    const folderUrl = buildFolderUrl(path)
    if (!folderUrl) {
        fallbackToApi()
        return
    }

    // 3. Use /folder URL immediately (browser will cache it if it loads fast)
    console.log('[AlbumCard] Using folder URL:', folderUrl)
    currentImageSrc.value = folderUrl
    usingFolderUrl.value = true
    folderUrlChecked.value = false

    // 4. In background, check if it returns 404
    checkFolderExists(folderUrl)
}

const fallbackToApi = () => {
    // Only fallback if we're still using the folder URL
    if (usingFolderUrl.value) {
        usingFolderUrl.value = false
        const fallbackUrl = props.coverUrl || fullDetails.value?.album?.coverUrl || ''
        console.log('[AlbumCard] fallbackToApi:', fallbackUrl)
        currentImageSrc.value = fallbackUrl
    }
}

const handleImageLoad = () => {
  imageLoaded.value = true
}

const handleImageError = () => {
  // Only fall back if we were using /folder and haven't checked yet
  if (usingFolderUrl.value && !folderUrlChecked.value) {
      console.log('[AlbumCard] Image load error, checking if 404 or other error')
      // The image failed to load - could be 404 or other error
      // Since we already checked with fetch, this is likely a network error
      // Try the API fallback anyway
      fallbackToApi()
  } else {
      // API also failed or already checked, clear it to show default icon
      imageLoaded.value = true // Stop loading animation
      currentImageSrc.value = ''
  }
}

// Watch for prop changes (e.g. background enrichment finishes and parent passes new data)
watch(() => props.albumDetails, (newVal) => {
    if (newVal) {
        initData()
        // Reset state when new data arrives
        usingFolderUrl.value = false
        folderUrlChecked.value = false
        resolveImageSource()
    }
}, { deep: true, immediate: true })

// Re-resolve if config loads later
watch(() => mpdStore.config, () => {
    if (fullDetails.value) {
        resolveImageSource()
    }
}, { deep: true })

// Intersection Observer for lazy loading
let observer = null

const fetchDetails = async () => {
  // If album is missing or "undefined", we can't fetch details.
  // We allow artist to be empty as per user feedback ("normal" to have empty fields).
  if (!props.album || props.album === 'undefined' || props.album === '') {
      return
  }

  // If we already have enriched data (trackCount > 0), DO NOT FETCH.
  if (fullDetails.value?.album?.trackCount > 0) {
      return
  }
  
  if (loading.value) return
  
  loading.value = true
  isBusy.value = false
  try {
    // Use batch loader to coordinate requests and handle throttling
    const response = await albumBatchLoader.requestDetails(props.artist, props.album)
    if (response && response.data) {
      fullDetails.value = response.data
      resolveImageSource()
    }
  } catch (error) {
    if (error.response && error.response.status === 429) {
      isBusy.value = true
    } else {
      console.error('Failed to fetch album details:', error)
      // Even if fetch fails, try to show something if we have props
      fallbackToApi()
    }
  } finally {
    loading.value = false
  }
}

const playAlbum = () => {
  if (fullDetails.value?.tracks) {
    fullDetails.value.tracks.forEach(track => {
      mpdStore.addToPlaylist(track.path)
    })
  } else {
    // If not loaded, we could potentially have a backend endpoint for play-album-by-name
    // But for now, let's just trigger load and play
    fetchDetails().then(() => {
      if (fullDetails.value?.tracks) {
        fullDetails.value.tracks.forEach(track => {
            mpdStore.addToPlaylist(track.path)
        })
      }
    })
  }
}

const playTrack = (track) => {
  mpdStore.addToPlaylist(track.path)
}

const navigateToDetails = () => {
  router.push({ 
    name: 'album-detail', 
    params: { 
      artist: props.artist, 
      album: props.album 
    } 
  })
}

const openMetadataSearch = () => {
  // Emit event to open metadata search modal with album and artist pre-filled
  emit('open-metadata-search', {
    artist: props.artist,
    album: props.album
  })
}

const formatDuration = (seconds) => {
  if (!seconds) return '--:--'
  const m = Math.floor(seconds / 60)
  const s = Math.floor(seconds % 60)
  return `${m}:${s.toString().padStart(2, '0')}`
}

// Handle cache update events from background refresh
const handleCacheUpdate = (event) => {
  const { artist, album, data } = event.detail
  // Check if this update is for this album
  if (artist === props.artist && album === props.album) {
    console.log('[AlbumCard] Cache updated for:', artist, '-', album)
    fullDetails.value = data
    // Reset state when cache updates
    usingFolderUrl.value = false
    resolveImageSource()
  }
}

onMounted(() => {
  // IMPORTANT: Resolve image source immediately with props (fast path)
  // This shows cover art right away without waiting for fetchDetails API call
  resolveImageSource()

  observer = new IntersectionObserver((entries) => {
    if (entries[0].isIntersecting) {
      isVisible.value = true
      fetchDetails() // Only fetches trackCount/duration now, doesn't block image
      // Once visible and fetching started, we can stop observing if we don't care about visibility anymore
      // or keep it if we want to pause/resume. For lazy load, one-time is enough.
      observer.disconnect()
    }
  }, { threshold: 0.1 })

  if (cardRef.value) {
    observer.observe(cardRef.value)
  }

  // Listen for cache update events
  window.addEventListener('album-cache-updated', handleCacheUpdate)
})

onUnmounted(() => {
  if (observer) {
    observer.disconnect()
  }
  // Remove cache update listener
  window.removeEventListener('album-cache-updated', handleCacheUpdate)
})
</script>

<style scoped>
.text-primary-400 { color: #60a5fa; }
.text-primary-500 { color: #3b82f6; }
.bg-primary-500 { background-color: #3b82f6; }
.border-primary-500\/50 { border-color: rgba(59, 130, 246, 0.5); }
.ring-primary-500\/20 { --tw-ring-color: rgba(59, 130, 246, 0.2); }

.custom-scrollbar::-webkit-scrollbar {
  width: 4px;
}
.custom-scrollbar::-webkit-scrollbar-track {
  background: transparent;
}
.custom-scrollbar::-webkit-scrollbar-thumb {
  background: #52525b;
  border-radius: 10px;
}
.custom-scrollbar::-webkit-scrollbar-thumb:hover {
  background: #71717a;
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}
@keyframes slideInUp {
  from { transform: translateY(10px); opacity: 0; }
  to { transform: translateY(0); opacity: 1; }
}

.animate-in {
  animation-duration: 0.3s;
  animation-fill-mode: both;
}
.fade-in { animation-name: fadeIn; }
.slide-in-from-bottom-1 { animation-name: slideInUp; }
</style>
