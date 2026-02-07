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
          @click="showCoverPicker = true"
          class="absolute inset-0 bg-black bg-opacity-50 flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity"
        >
          <span class="text-white text-sm font-medium">Change Cover</span>
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
                <svg v-if="hasSelection" xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                <svg v-else xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                  <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM9.555 7.168A1 1 0 008 8v4a1 1 0 001.555.832l3-1.664a1 1 0 000-1.664l-3-1.664z" clip-rule="evenodd" />
                </svg>
                <span>{{ hasSelection ? 'Play Selected' : 'Play Album' }}</span>
             </button>
             <button @click="handleAction('next')" class="flex items-center space-x-2 bg-neutral-700 hover:bg-neutral-600 text-white px-4 py-2 rounded transition-colors font-medium">
                <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                   <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 5l7 7-7 7M5 5l7 7-7 7" />
                 </svg>
                <span>Play Next</span>
             </button>
             <button @click="handleAction('append')" class="flex items-center space-x-2 bg-neutral-700 hover:bg-neutral-600 text-white px-4 py-2 rounded transition-colors font-medium">
                <svg class="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v3m0 0v3m0-3h3m-3 0H9m12 0a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
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
              <div class="truncate" :title="track.title">{{ track.title }}</div>
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
                class="aspect-square relative group cursor-pointer rounded-lg overflow-hidden bg-neutral-900 border border-neutral-800 hover:border-blue-500 transition-colors"
                @click="selectCover(cover)"
              >
                <img :src="cover.thumbnail || cover.url" class="w-full h-full object-cover" />
                
                <!-- Dimension Badge -->
                <div v-if="cover.width && cover.height" class="absolute bottom-1 right-1 px-1.5 py-0.5 bg-black/70 backdrop-blur-md rounded text-[9px] font-bold text-white opacity-0 group-hover:opacity-100 transition-opacity">
                  {{ cover.width }}x{{ cover.height }}
                </div>

                <div class="absolute inset-0 bg-black/40 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center">
                  <span class="text-xs font-bold text-white uppercase tracking-wider">Select</span>
                </div>
              </div>
        </div>
      </div>
    </BaseModal>

    <!-- Confirmation Modal -->
    <BaseConfirmModal
      v-model="showConfirmModal"
      title="Update Cover Art"
      message="Set this as album cover art? This will update the folder image on disk."
      confirm-label="Set Cover"
      :loading="applyingCover"
      @confirm="handleConfirmCover"
      @cancel="showConfirmModal = false"
    >
      <div v-if="pendingCover" class="aspect-square w-32 mx-auto rounded-lg overflow-hidden border border-neutral-700 shadow-xl">
        <img :src="pendingCover.url" class="w-full h-full object-cover" />
      </div>
    </BaseConfirmModal>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMpdStore } from '@/stores/mpd'
import BaseToast from '@/components/BaseToast.vue'
import BaseModal from '@/components/BaseModal.vue'
import BaseConfirmModal from '@/components/BaseConfirmModal.vue'
import MetadataSearchModal from '@/components/MetadataSearchModal.vue'

const route = useRoute()
const router = useRouter()
const mpdStore = useMpdStore()

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

// Confirmation Modal
const showConfirmModal = ref(false)
const pendingCover = ref(null)
const applyingCover = ref(false)

// Toast Notification
const showToast = ref(false)
const toastMessage = ref('')
const toastType = ref('success')

const showNotification = (message, type = 'success') => {
    toastMessage.value = message
    toastType.value = type
    showToast.value = true
}

// Selection State
const selectedTracks = ref(new Set())
const selectionOrder = ref([])

const hasSelection = computed(() => selectedTracks.value.size > 0)

const isTrackSelected = (track) => selectedTracks.value.has(track.path)

const getSelectionOrder = (track) => {
    return selectionOrder.value.indexOf(track.path) + 1
}

const toggleSelection = (track) => {
    if (selectedTracks.value.has(track.path)) {
        selectedTracks.value.delete(track.path)
        selectionOrder.value = selectionOrder.value.filter(path => path !== track.path)
    } else {
        selectedTracks.value.add(track.path)
        selectionOrder.value.push(track.path)
    }
}

const clearSelection = () => {
    selectedTracks.value.clear()
    selectionOrder.value = []
}

const getTargetTracks = () => {
    if (hasSelection.value) {
        return selectionOrder.value
    }
    return tracks.value.map(t => t.path)
}

const handleAction = async (mode) => {
    const tracksToAdd = getTargetTracks()
    if (tracksToAdd.length === 0) return
    
    try {
        await mpdStore.addTracks(tracksToAdd, mode)
        // Feedback
        if (mode === 'append') showNotification('Added to queue')
        if (mode === 'next') showNotification('Playing next')
        if (mode === 'play') {
             // Already playing
        }
        clearSelection()
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
    }
  } catch (error) {
    console.error('[AlbumDetailView] Error fetching album details:', error)
  } finally {
    loading.value = false
    console.log('[AlbumDetailView] Loading complete, albumDetails:', albumDetails.value)
  }
}

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
      selectionMetadata.value = true // Reusing searching state
      const result = await mpdStore.applyMetadata(albumPath.value, overlay.originalMetadata)
      handleMetadataApplied(result)
    } catch (error) {
      showNotification('Reapply failed: ' + error.message, 'error')
    } finally {
      selectionMetadata.value = false
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
  pendingCover.value = cover
  showConfirmModal.value = true
}

const handleConfirmCover = async () => {
  if (!pendingCover.value) return
  
  applyingCover.value = true
  try {
    const albumPath = tracks.value[0]?.path.split('/').slice(0, -1).join('/')
    await mpdStore.applyCoverArt(albumPath, pendingCover.value.url)
    showNotification('Cover art updated successfully')
    showCoverPicker.value = false
    showConfirmModal.value = false
    
    // Cache bust local cover
    coverBust.value = Date.now()
    
    // Refresh album details
    await fetchAlbumDetails()
  } catch (error) {
    showNotification('Failed to update cover art: ' + error.message, 'error')
  } finally {
    applyingCover.value = false
    pendingCover.value = null
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
