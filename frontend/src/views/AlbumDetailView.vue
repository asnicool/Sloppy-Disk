<template>
  <div v-if="albumDetails" class="space-y-8">
    <!-- Album Header -->
    <div class="flex flex-col md:flex-row items-start md:items-end space-y-4 md:space-y-0 md:space-x-6">
      <div class="w-48 h-48 bg-gray-800 rounded-lg flex items-center justify-center relative group overflow-hidden shadow-2xl">
        <img 
          v-if="albumDetails?.coverUrl" 
          :src="albumDetails.coverUrl" 
          :alt="albumName"
          class="w-full h-full object-cover transition-transform duration-500 group-hover:scale-105"
        />
        <svg v-else class="w-20 h-20 text-gray-700" fill="currentColor" viewBox="0 0 20 20">
          <path d="M18 3a1 1 0 00-1.196-.98l-10 2A1 1 0 006 5v9.114A4.369 4.369 0 005 14c-1.657 0-3 .895-3 2s1.343 2 3 2 3-.895 3-2V7.82l8-1.6v5.894A4.369 4.369 0 0015 12c-1.657 0-3 .895-3 2s1.343 2 3 2 3-.895 3-2V3z" />
        </svg>
        <button 
          @click="showCoverPicker = true"
          class="absolute inset-0 bg-black bg-opacity-50 flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity"
        >
          <span class="text-white text-sm font-medium">Change Cover</span>
        </button>
      </div>
      <div class="flex-1">
        <h1 class="text-4xl font-bold text-white mb-2">{{ albumName }}</h1>
        <p class="text-xl text-gray-400 mb-2">{{ artistName }}</p>
        <div v-if="albumDetails" class="flex flex-wrap gap-3 mb-4 text-sm">
          <span v-if="albumDetails.date" class="text-gray-500">{{ albumDetails.date }}</span>
          <span v-if="albumDetails.genre" class="text-gray-500 px-2 border-l border-gray-700">{{ albumDetails.genre }}</span>
          <span v-if="albumDetails.trackCount" class="text-gray-500 px-2 border-l border-gray-700">{{ albumDetails.trackCount }} tracks</span>
        </div>
        <div class="flex space-x-4">
          <button @click="searchMetadata" class="bg-blue-600 hover:bg-blue-500 text-white px-4 py-2 rounded transition-colors">
            Search Metadata
          </button>
        </div>
      </div>
    </div>

    <!-- Tracks List -->
    <div class="bg-gray-800 rounded-lg overflow-hidden">
      <table class="w-full text-left">
        <thead class="bg-gray-700 text-gray-400 text-sm uppercase">
          <tr>
            <th class="px-6 py-3 font-medium">#</th>
            <th class="px-6 py-3 font-medium">Title</th>
            <th class="px-6 py-3 font-medium">Duration</th>
            <th class="px-6 py-3 font-medium"></th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-700">
          <tr v-for="(track, index) in tracks" :key="track.path" class="hover:bg-gray-700 transition-colors group">
            <td class="px-6 py-4 text-gray-400">{{ track.track || index + 1 }}</td>
            <td class="px-6 py-4 text-white">{{ track.title }}</td>
            <td class="px-6 py-4 text-gray-400">{{ formatDuration(track.duration) }}</td>
            <td class="px-6 py-4 text-right">
              <button @click="playTrack(track)" class="text-blue-400 hover:text-blue-300 opacity-0 group-hover:opacity-100 transition-opacity">
                Play
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Metadata Search Modal -->
    <div v-if="showMetadataModal" class="fixed inset-0 bg-black bg-opacity-75 flex items-center justify-center p-4 z-50">
      <div class="bg-gray-800 rounded-lg max-w-2xl w-full max-h-[80vh] overflow-hidden flex flex-col">
        <div class="p-6 border-b border-gray-700 flex justify-between items-center">
          <h2 class="text-xl font-bold text-white">Metadata Candidates</h2>
          <button @click="showMetadataModal = false" class="text-gray-400 hover:text-white">&times;</button>
        </div>
        <div class="p-6 overflow-y-auto flex-1 space-y-4">
          <div v-if="searchingMetadata" class="text-center py-8 text-gray-400">Searching Discogs...</div>
          <div v-else-if="metadataCandidates.length === 0" class="text-center py-8 text-gray-400">No candidates found.</div>
          <div 
            v-for="candidate in metadataCandidates" 
            :key="candidate.externalId"
            class="bg-gray-700 p-4 rounded-lg flex justify-between items-center hover:bg-gray-600 cursor-pointer transition-colors"
            @click="applyMetadata(candidate)"
          >
            <div>
              <h3 class="text-white font-medium">{{ candidate.album }}</h3>
              <p class="text-gray-400 text-sm">{{ candidate.artist }} ({{ candidate.year }})</p>
              <p class="text-blue-400 text-xs mt-1">Source: {{ candidate.source }}</p>
            </div>
            <button class="text-blue-400 font-medium">Apply</button>
          </div>
        </div>
      </div>
    </div>

    <!-- Cover Picker Modal -->
    <div v-if="showCoverPicker" class="fixed inset-0 bg-black bg-opacity-75 flex items-center justify-center p-4 z-50">
      <div class="bg-gray-800 rounded-lg max-w-4xl w-full max-h-[80vh] overflow-hidden flex flex-col">
        <div class="p-6 border-b border-gray-700 flex justify-between items-center">
          <h2 class="text-xl font-bold text-white">Pick Cover Art</h2>
          <button @click="showCoverPicker = false" class="text-gray-400 hover:text-white">&times;</button>
        </div>
        <div class="p-6 overflow-y-auto flex-1">
          <div v-if="fetchingCovers" class="text-center py-8 text-gray-400">Fetching covers...</div>
          <div v-else class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
            <div 
              v-for="cover in coverCandidates" 
              :key="cover.url"
              class="relative aspect-square bg-gray-700 rounded overflow-hidden group cursor-pointer"
              @click="selectCover(cover)"
            >
              <img :src="cover.url" class="w-full h-full object-cover" alt="Cover candidate">
              <div class="absolute inset-0 bg-black bg-opacity-50 flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity">
                <span class="text-white text-xs">{{ cover.size }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { useRoute } from 'vue-router'
import { useMpdStore } from '@/stores/mpd'

const route = useRoute()
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

const fetchAlbumDetails = async () => {
  loading.value = true
  try {
    const response = await mpdStore.fetchAlbumSongs(artistName.value, albumName.value)
    if (response.success) {
      albumDetails.value = response.data.album
      tracks.value = response.data.tracks
    }
  } finally {
    loading.value = false
  }
}

const searchMetadata = async () => {
  showMetadataModal.value = true
  searchingMetadata.value = true
  try {
    const response = await mpdStore.fetchMetadataCandidates(artistName.value, albumName.value)
    if (response.success) {
      metadataCandidates.value = response.data
    }
  } finally {
    searchingMetadata.value = false
  }
}

const applyMetadata = async (candidate) => {
  // In a real app, this would open an editor first
  if (confirm(`Apply metadata from ${candidate.source}?`)) {
    // Call backend to apply tags
    alert('Metadata applied (mock)')
    showMetadataModal.value = false
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
  if (confirm('Set this as album cover?')) {
    try {
      // We need the album path. For now using a placeholder or deriving from tracks
      const albumPath = tracks.value[0]?.path.split('/').slice(0, -1).join('/')
      await mpdStore.applyCoverArt(albumPath, cover.url)
      alert('Cover art updated')
      showCoverPicker.value = false
    } catch (error) {
      alert('Failed to update cover art')
    }
  }
}

const playTrack = (track) => {
  mpdStore.addToPlaylist(track.path)
}

const formatDuration = (seconds) => {
  if (!seconds) return '0:00'
  const minutes = Math.floor(seconds / 60)
  const remainingSeconds = Math.floor(seconds % 60)
  return `${minutes}:${remainingSeconds.toString().padStart(2, '0')}`
}

onMounted(() => {
  fetchAlbumDetails()
  fetchCovers()
})
</script>
