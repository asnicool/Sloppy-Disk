<template>
  <div v-if="loading" class="flex items-center justify-center h-64">
    <div class="text-neutral-400">Loading album details...</div>
  </div>
  <div v-else-if="albumDetails" class="space-y-8">
    <!-- Album Header -->
    <!-- Album Header -->
    <div class="flex flex-col md:flex-row items-start md:items-end space-y-4 md:space-y-0 md:space-x-6">
      <BaseToast v-model="showToast" :message="toastMessage" :type="toastType" />
      <div class="w-48 h-48 bg-neutral-800 rounded-lg flex items-center justify-center relative group overflow-hidden shadow-2xl flex-shrink-0">
        <img 
          v-if="albumDetails?.coverUrl" 
          :src="albumDetails.coverUrl + (coverBust ? '?t=' + coverBust : '')" 
          :alt="albumName"
          class="w-full h-full object-cover transition-transform duration-500 group-hover:scale-105"
        />
        <svg v-else class="w-20 h-20 text-neutral-700" fill="currentColor" viewBox="0 0 20 20">
          <path d="M18 3a1 1 0 00-1.196-.98l-10 2A1 1 0 006 5v9.114A4.369 4.369 0 005 14c-1.657 0-3 .895-3 2s1.343 2 3 2 3-.895 3-2V7.82l8-1.6v5.894A4.369 4.369 0 0015 12c-1.657 0-3 .895-3 2s1.343 2 3 2 3-.895 3-2V3z" />
        </svg>
        <button 
          @click="handleCoverClick"
          class="absolute inset-0 bg-black bg-opacity-50 flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity"
        >
          <div class="flex flex-col items-center space-y-2">
            <span class="text-white text-sm font-medium">Cover Options</span>
            <div class="flex items-center space-x-2 text-[10px] font-bold uppercase tracking-wider text-white/40">
                <span class="px-1.5 py-0.5 border border-white/20 rounded">Click</span>
                <span>Picker</span>
                <span class="text-white/20">•</span>
                <span class="px-1.5 py-0.5 border border-white/20 rounded">Double</span>
                <span>Zoom</span>
            </div>
          </div>
        </button>
      </div>
      
      <div class="flex-1 w-full min-w-0 flex flex-col justify-end h-48">
        <div class="flex-1">
            <h1 class="text-4xl font-bold text-white mb-2 truncate" :title="albumName">{{ albumName }}</h1>
            <p 
                class="text-xl text-neutral-400 mb-2 truncate cursor-pointer hover:text-primary-400 transition-colors"
                @click="navigateToArtist"
                :title="'Click to view all albums by ' + artistName"
            >
                {{ artistName }}
            </p>
            <div v-if="albumDetails" class="flex flex-wrap gap-3 mb-4 text-sm">
            <span 
                v-if="albumDetails.date" 
                class="text-neutral-500 cursor-pointer hover:text-primary-400 transition-colors"
                @click="navigateToDate"
                :title="'Click to view all albums from ' + albumDetails.date"
            >
                {{ albumDetails.date }}
            </span>
            <span 
                v-if="albumDetails.genre" 
                class="text-neutral-500 px-2 border-l border-neutral-700 cursor-pointer hover:text-primary-400 transition-colors"
                @click="navigateToGenre"
                :title="'Click to view all albums in ' + albumDetails.genre"
            >
                {{ albumDetails.genre }}
            </span>
            <span v-if="albumDetails.trackCount" class="text-neutral-500 px-2 border-l border-neutral-700">{{ albumDetails.trackCount }} tracks</span>
            </div>
        </div>

        <div class="flex flex-wrap gap-3 mt-auto">
             <button @click="handleAction('play')" class="flex items-center space-x-2 bg-blue-600 hover:bg-blue-500 text-white px-4 py-2 rounded transition-colors font-medium">
                <span v-html="QueueActionIcons.play"></span>
                <span>{{ hasSelection ? 'Play Selected' : 'Play Album' }}</span>
             </button>
             <button @click="handleAction('next')" class="flex items-center space-x-2 bg-neutral-700 hover:bg-neutral-600 text-white px-4 py-2 rounded transition-colors font-medium">
                <span v-html="QueueActionIcons.next"></span>
                <span>Play Next</span>
             </button>
             <button @click="handleAction('append')" class="flex items-center space-x-2 bg-neutral-700 hover:bg-neutral-600 text-white px-4 py-2 rounded transition-colors font-medium">
                <span v-html="QueueActionIcons.append"></span>
                <span>Add to Queue</span>
             </button>
             
             <div v-if="albumDetails?.isOverlayActive" class="flex items-center space-x-2 text-amber-500 bg-amber-500 bg-opacity-10 px-3 py-1 rounded border border-amber-500 border-opacity-20">
                <svg class="animate-spin h-4 w-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                  <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
                  <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                </svg>
                <span class="text-xs font-medium">Out of Sync</span>
                <button @click="reapplyMetadata" class="text-xs font-bold hover:underline ml-2">Reapply</button>
             </div>

             <button @click="searchMetadata" class="ml-auto text-sm text-neutral-400 hover:text-white px-3 py-2 rounded border border-neutral-700 hover:border-neutral-500 transition-colors">
                Metadata
            </button>
             <button v-if="hasSelection" @click="clearSelection" class="text-sm text-neutral-400 hover:text-white px-3 py-2">
                Clear Selection
             </button>
        </div>
      </div>
    </div>

    <!-- Tracks List -->
    <div class="bg-neutral-800 rounded-lg overflow-hidden">
      <table class="w-full text-left">
        <thead class="bg-neutral-700 text-neutral-400 text-xs uppercase">
          <tr>
            <th class="px-4 py-2 font-medium w-12">#</th>
            <th class="px-4 py-2 font-medium w-16 text-right">Time</th>
            <th class="px-4 py-2 font-medium">Title</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-neutral-700 select-none">
          <tr 
            v-for="(track, index) in tracks" 
            :key="track.path" 
            :data-track-path="track.path"
            @click="toggleSelection(track)"
            class="transition-colors group cursor-pointer"
            :class="[
                isTrackSelected(track) ? 'bg-blue-900 bg-opacity-30 hover:bg-blue-900 hover:bg-opacity-40' : 'hover:bg-neutral-700',
                hasSelection && !isTrackSelected(track) ? 'text-neutral-500' : 'text-neutral-300'
            ]"
          >
            <td class="px-4 py-2 relative group-hover:text-white">
                <div class="relative w-6 h-6 flex items-center justify-center">
                    <span 
                        class="text-sm transition-opacity duration-200"
                        :class="[
                            hasSelection && !isTrackSelected(track) ? 'opacity-50' : 'opacity-100',
                            isTrackSelected(track) ? 'opacity-0' : 'opacity-100'
                        ]"
                    >
                        {{ track.track || index + 1 }}
                    </span>
                    <!-- Selection Badge -->
                    <div 
                        v-if="isTrackSelected(track)" 
                        class="absolute inset-0 bg-blue-500 rounded-full flex items-center justify-center text-xs text-white font-bold shadow-sm transition-opacity duration-200"
                    >
                        {{ getSelectionOrder(track) }}
                    </div>
                </div>
            </td>
            <td class="px-4 py-2 text-right text-sm" :class="{ 'opacity-50': hasSelection && !isTrackSelected(track) }">{{ formatDuration(track.duration) }}</td>
            <td class="px-4 py-2 text-sm" :class="{ 'text-white': !hasSelection || isTrackSelected(track) }">
              <div class="flex items-center space-x-2 truncate" :title="track.title">
                <span class="truncate flex-1">{{ track.title }}</span>
                <span 
                  class="text-[10px] font-bold px-1.5 py-0.5 rounded uppercase tracking-wider flex-shrink-0"
                  :class="getFileExtensionClass(track.path)"
                >
                  {{ getFileExtension(track.path) }}
                </span>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

     <MetadataSearchModal
       v-if="showMetadataModal"
       :is-open="showMetadataModal"
       :initial-artist="artistName"
       :initial-album="albumName"
       :album-path="albumPath"
       :track-count="albumDetails?.trackCount"
       :duration="albumDetails?.duration"
       :library-tracks="tracks"
       @close="showMetadataModal = false"
       @applied="handleMetadataApplied"
       @cover-updated="handleCoverUpdated"
     />

    <!-- Cover Picker Modal -->
    <BaseModal
      v-model="showCoverPicker"
      title="Pick Cover Art"
    >
      <div class="space-y-6">
        <div v-if="fetchingCovers" class="text-center py-8 text-neutral-400">Fetching covers...</div>
        <div v-else class="grid grid-cols-2 md:grid-cols-3 gap-4">
              <div
                v-for="(cover, idx) in coverCandidates"
                :key="idx"
                :class="[
                  'aspect-square relative group cursor-pointer rounded-lg overflow-hidden bg-neutral-900 border transition-all duration-300',
                  applyingCover === cover.url ? 'border-green-500 ring-2 ring-green-500' : 'border-neutral-800 hover:border-blue-500'
                ]"
                @click="selectCover(cover)"
              >
                <img :src="cover.thumbnail || cover.url" class="w-full h-full object-cover transition-transform duration-500 group-hover:scale-110" />
                
                <!-- Dimension Legend (Always Visible) -->
                <div v-if="cover.width && cover.height" class="absolute bottom-0 left-0 right-0 px-2 py-1.5 bg-black/80 backdrop-blur-md border-t border-white/10 text-[10px] font-bold text-white flex justify-between items-center z-10">
                  <span class="opacity-50 uppercase tracking-tighter text-[8px]">Dimensions</span>
                  <span class="text-blue-400">{{ cover.width }}<span class="text-white/40 mx-0.5">x</span>{{ cover.height }}</span>
                </div>

                <!-- Loading State -->
                <div v-if="applyingCover === cover.url" class="absolute inset-0 bg-green-600/80 flex items-center justify-center backdrop-blur-[2px]">
                  <svg class="animate-spin h-8 w-8 text-white" viewBox="0 0 24 24">
                    <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
                    <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                  </svg>
                </div>

                <!-- Hover State (only show when not loading) -->
                <div v-else class="absolute inset-0 bg-blue-600/40 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center backdrop-blur-[1px]">
                  <div class="bg-blue-600 text-white px-3 py-1 rounded-full text-[10px] font-bold uppercase tracking-widest shadow-lg transform -translate-y-2 group-hover:translate-y-0 transition-transform">
                    Set Cover
                  </div>
                </div>
              </div>
        </div>
      </div>
    </BaseModal>

    <!-- Full Screen Cover Overlay -->
    <Transition
      enter-active-class="transition duration-300 ease-out"
      enter-from-class="opacity-0 scale-95"
      enter-to-class="opacity-100 scale-100"
      leave-active-class="transition duration-200 ease-in"
      leave-from-class="opacity-100 scale-100"
      leave-to-class="opacity-0 scale-95"
    >
      <div 
        v-if="showFullScreenCover" 
        class="fixed inset-0 z-[100] flex items-center justify-center bg-black/95 backdrop-blur-sm p-4 md:p-8"
        @click="showFullScreenCover = false"
      >
        <div class="relative max-w-5xl w-full h-full flex items-center justify-center">
          <img 
            :src="albumDetails?.coverUrl + (coverBust ? '?t=' + coverBust : '')" 
            :alt="albumName"
            class="max-w-full max-h-full object-contain shadow-2xl rounded-lg"
            @click.stop
          />
          
          <!-- Close Button -->
          <button 
            @click="showFullScreenCover = false"
            class="absolute top-0 right-0 m-4 p-2 bg-white/10 hover:bg-white/20 rounded-full text-white transition-colors"
          >
            <svg xmlns="http://www.w3.org/2000/svg" class="h-8 w-8" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>

          <!-- Info Overlay -->
          <div class="absolute bottom-0 left-0 right-0 p-8 bg-gradient-to-t from-black/80 to-transparent text-white rounded-b-lg">
            <h2 class="text-3xl font-bold">{{ albumName }}</h2>
            <p class="text-xl text-neutral-300">{{ artistName }}</p>
          </div>
        </div>
      </div>
    </Transition>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMpdStore } from '@/stores/mpd'
import { useQueueActions, QueueActionIcons } from '@/composables/useQueueActions'
import BaseToast from '@/components/BaseToast.vue'
import BaseModal from '@/components/BaseModal.vue'
import MetadataSearchModal from '@/components/MetadataSearchModal.vue'

const route = useRoute()
const router = useRouter()
const mpdStore = useMpdStore()

// Use shared queue actions composable for selection management
const {
  hasSelection,
  isTrackSelected,
  getSelectionOrder,
  toggleSelection,
  clearSelection,
  handleAction: handleQueueAction
} = useQueueActions()

const artistName = computed(() => route.params.artist)
const albumName = computed(() => route.params.album)

const albumDetails = ref(null)
const tracks = ref([])
const loading = ref(true)

const showMetadataModal = ref(false)
const searchingMetadata = ref(false)
const metadataCandidates = ref([])

const showCoverPicker = ref(false)
const fetchingCovers = ref(false)
const coverCandidates = ref([])
const coverBust = ref(0)
const showFullScreenCover = ref(false)
const applyingCover = ref(null) // Stores the URL of cover being applied

// Toast notification state
const showToast = ref(false)
const toastMessage = ref('')
const toastType = ref('success')

// Click handling for cover art
let clickCount = 0
let clickTimer = null

const handleCoverClick = () => {
    clickCount++
    if (clickCount === 1) {
        clickTimer = setTimeout(() => {
            if (clickCount === 1) {
                showCoverPicker.value = true
            }
            clickCount = 0
        }, 250) // 250ms is standard for double click detection
    } else if (clickCount === 2) {
        clearTimeout(clickTimer)
        showFullScreenCover.value = true
        clickCount = 0
    }
}

const showNotification = (message, type = 'success') => {
    toastMessage.value = message
    toastType.value = type
    showToast.value = true
}

const handleAction = async (mode) => {
  // If no explicit track selection, delegate to backend album endpoint
  // which can handle album-level adds more efficiently
  if (!hasSelection.value && albumDetails.value) {
    try {
      await mpdStore.addAlbumToPlaylist(artistName.value, albumName.value, mode)
      if (mode === 'append') showNotification('Added to queue')
      if (mode === 'next') showNotification('Playing next')
      if (mode === 'play') showNotification('Playing album')
    } catch (error) {
      showNotification('Action failed: ' + error.message, 'error')
    }
    return
  }

  // Use composable for track selection actions
  try {
    await handleQueueAction(mode)
    if (mode === 'append') showNotification('Added to queue')
    if (mode === 'next') showNotification('Playing next')
    if (mode === 'play') showNotification('Playing')
  } catch (error) {
    showNotification('Action failed: ' + error.message, 'error')
  }
}

const playSingleTrack = (track) => {
    mpdStore.addTracks([track.path], 'play')
}

const albumPath = ref('')

const fetchAlbumDetails = async () => {
  loading.value = true
  console.log('[AlbumDetailView] Fetching album details:', artistName.value, '-', albumName.value)
  try {
    const response = await mpdStore.fetchAlbumSongs(artistName.value, albumName.value)
    console.log('[AlbumDetailView] API response:', response)
    
    // Handle both cache (direct data) and API (wrapped) responses
    let albumData = null
    if (response.success) {
      // Fresh API response: { success: true, data: { album, tracks } }
      albumData = response.data.album
      tracks.value = response.data.tracks
    } else if (response.album && response.tracks) {
      // Cached response: { album, tracks }
      albumData = response.album
      tracks.value = response.tracks
    }
    
    if (albumData) {
      console.log('[AlbumDetailView] Setting albumDetails:', albumData)
      albumDetails.value = albumData
    } else {
      console.error('[AlbumDetailView] Could not parse response:', response)
    }

    // Calculate album path from first track
    if (tracks.value.length > 0) {
      const firstTrack = tracks.value[0]
      albumPath.value = firstTrack.path.split('/').slice(0, -1).join('/')

      // Check for pre-selection from query parameter
      handlePreSelection()
    }
  } catch (error) {
    console.error('[AlbumDetailView] Error fetching album details:', error)
  } finally {
    loading.value = false
    console.log('[AlbumDetailView] Loading complete, albumDetails:', albumDetails.value)
  }
}

const handlePreSelection = () => {
    const selectPath = route.query.selectPath
    if (selectPath && tracks.value.length > 0) {
        console.log('[AlbumDetailView] Handling pre-selection for:', selectPath)
        const track = tracks.value.find(t => t.path === selectPath)
        if (track) {
            // Already selected?
            if (!selectedTracks.value.has(track.path)) {
                toggleSelection(track)
            }
            // Scroll to track after a short delay for DOM update
            setTimeout(() => {
                const element = document.querySelector(`[data-track-path="${CSS.escape(track.path)}"]`)
                if (element) {
                    element.scrollIntoView({ behavior: 'smooth', block: 'center' })
                }
            }, 500)
        }
    }
}

watch(() => route.query.selectPath, () => {
    handlePreSelection()
})

const searchMetadata = async () => {
  showMetadataModal.value = true
}

const handleMetadataApplied = (result) => {
  showNotification(`Metadata applied to ${result.updatedFiles} files`, 'success')
  // Cache bust local cover if coverArtUrl was provided
  if (result.coverArtUrl) {
    coverBust.value = Date.now()
  }
  // Refresh album details to show updated metadata
  fetchAlbumDetails()
}

const handleCoverUpdated = () => {
  showNotification('Cover art updated successfully')
  coverBust.value = Date.now()
  fetchAlbumDetails()
}

const reapplyMetadata = async () => {
  const overlay = mpdStore.albumCache.getOverlay(artistName.value, albumName.value)
  if (overlay && overlay.originalMetadata) {
    try {
      searchingMetadata.value = true // Reusing searching state
      const result = await mpdStore.applyMetadata(albumPath.value, overlay.originalMetadata)
      handleMetadataApplied(result)
    } catch (error) {
      showNotification('Reapply failed: ' + error.message, 'error')
    } finally {
      searchingMetadata.value = false
    }
  }
}

const fetchCovers = async () => {
  fetchingCovers.value = true
  try {
    const response = await mpdStore.fetchCoverArtCandidates(artistName.value, albumName.value)
    if (response.success) {
      coverCandidates.value = response.data
    }
  } finally {
    fetchingCovers.value = false
  }
}

const selectCover = async (cover) => {
  // Prevent multiple simultaneous clicks
  if (applyingCover.value) return
  
  applyingCover.value = cover.url
  try {
    const albumPathValue = tracks.value[0]?.path.split('/').slice(0, -1).join('/')
    await mpdStore.applyCoverArt(albumPathValue, cover.url)
    showNotification('Cover art updated successfully')
    showCoverPicker.value = false
    
    // Cache bust local cover
    coverBust.value = Date.now()
    
    // Refresh album details
    await fetchAlbumDetails()
  } catch (error) {
    showNotification('Failed to update cover art: ' + error.message, 'error')
  } finally {
    applyingCover.value = null
  }
}

// Replaced by playSingleTrack and handleAction
// const playTrack = (track) => {
//   mpdStore.addToPlaylist(track.path)
// }

const formatDuration = (seconds) => {
  if (!seconds) return '0:00'
  const minutes = Math.floor(seconds / 60)
  const remainingSeconds = Math.floor(seconds % 60)
  return `${minutes}:${remainingSeconds.toString().padStart(2, '0')}`
}

const getFileExtension = (path) => {
  if (!path) return ''
  const ext = path.split('.').pop()?.toLowerCase() || ''
  return ext
}

const getFileExtensionClass = (path) => {
  const ext = getFileExtension(path)
  const classes = {
    'flac': 'bg-purple-600 bg-opacity-20 text-purple-400',
    'mp3': 'bg-blue-600 bg-opacity-20 text-blue-400',
    'ogg': 'bg-orange-600 bg-opacity-20 text-orange-400',
    'wav': 'bg-green-600 bg-opacity-20 text-green-400',
    'aac': 'bg-pink-600 bg-opacity-20 text-pink-400',
    'm4a': 'bg-pink-600 bg-opacity-20 text-pink-400',
    'alac': 'bg-purple-600 bg-opacity-20 text-purple-400',
    'aiff': 'bg-green-600 bg-opacity-20 text-green-400',
    'wma': 'bg-yellow-600 bg-opacity-20 text-yellow-400',
  }
  return classes[ext] || 'bg-neutral-600 bg-opacity-20 text-neutral-400'
}

const navigateToArtist = () => {
  router.push({ name: 'search', query: { q: artistName.value, type: 'artist' } })
}

const navigateToDate = () => {
  if (albumDetails.value?.date) {
    router.push({ name: 'search', query: { q: albumDetails.value.date, type: 'date' } })
  }
}

const navigateToGenre = () => {
  if (albumDetails.value?.genre) {
    router.push({ name: 'search', query: { q: albumDetails.value.genre, type: 'genre' } })
  }
}

// Handle cache update events from background refresh
const handleCacheUpdate = (event) => {
  const { artist, album, data } = event.detail
  // Check if this update is for this album
  if (artist === artistName.value && album === albumName.value) {
    console.log('[AlbumDetailView] Cache updated for:', artist, '-', album)
    albumDetails.value = data.album
    tracks.value = data.tracks
  }
}

onMounted(() => {
  fetchAlbumDetails()
  fetchCovers()
  // Listen for cache update events
  window.addEventListener('album-cache-updated', handleCacheUpdate)
})

onUnmounted(() => {
  // Remove cache update listener
  window.removeEventListener('album-cache-updated', handleCacheUpdate)
})
</script>
