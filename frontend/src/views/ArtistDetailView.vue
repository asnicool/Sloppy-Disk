<template>
  <div class="space-y-6">
    <div class="flex items-center space-x-4">
      <button @click="goBack" class="p-2 hover:bg-neutral-800 rounded-lg transition-colors">
        <svg class="w-6 h-6 text-neutral-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
        </svg>
      </button>
      <h1 class="text-3xl font-bold text-white">Artist: {{ artistName }}</h1>
    </div>

    <!-- Artist Image Section -->
    <div class="flex flex-col md:flex-row gap-8 items-start">
      <!-- Artist Image -->
      <div class="w-64 h-64 rounded-xl overflow-hidden bg-neutral-800 flex-shrink-0">
        <img 
          v-if="artistImageUrl" 
          :src="artistImageUrl" 
          class="w-full h-full object-cover"
          :alt="artistName"
        />
        <div v-else class="w-full h-full flex items-center justify-center">
          <svg class="w-24 h-24 text-neutral-600" fill="currentColor" viewBox="0 0 20 20">
            <path fill-rule="evenodd" d="M10 9a3 3 0 100-6 3 3 0 000 6zm-7 9a7 7 0 1114 0H3z" clip-rule="evenodd" />
          </svg>
        </div>
      </div>

      <!-- Artist Info -->
      <div class="flex-1 space-y-4">
        <div class="text-neutral-400">
          <p>{{ albumCount }} albums</p>
        </div>
        
        <!-- Update Button -->
        <button 
          @click="showCandidates = true"
          class="px-4 py-2 bg-primary-600 hover:bg-primary-500 text-white rounded-lg transition-colors flex items-center space-x-2"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
          </svg>
          <span>Update Image</span>
        </button>
      </div>
    </div>

    <!-- Albums by Artist -->
    <div class="space-y-4">
      <h2 class="text-xl font-semibold text-primary-400 border-b border-neutral-800 pb-2">Albums</h2>
      <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-4">
        <div 
          v-for="album in albums" 
          :key="album"
          @click="navigateToAlbum(album)"
          class="group bg-neutral-900/50 rounded-lg p-3 hover:bg-neutral-800 transition-all cursor-pointer border border-neutral-800 hover:border-primary-500/50"
        >
          <div class="aspect-square bg-neutral-800 rounded-md mb-2 flex items-center justify-center overflow-hidden">
             <svg class="w-12 h-12 text-neutral-700" fill="currentColor" viewBox="0 0 20 20">
              <path d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zM7 8H5v2h2V8zm2 0h2v2H9V8zm6 0h-2v2h2V8z" />
            </svg>
          </div>
          <p class="text-sm font-medium text-neutral-200 truncate group-hover:text-primary-400">{{ album }}</p>
        </div>
      </div>
    </div>

    <!-- Candidate Modal -->
    <div v-if="showCandidates" class="fixed inset-0 bg-black/70 flex items-center justify-center z-50" @click.self="showCandidates = false">
      <div class="bg-neutral-900 rounded-xl p-6 max-w-4xl w-full max-h-[80vh] overflow-y-auto">
        <div class="flex justify-between items-center mb-4">
          <h3 class="text-xl font-semibold text-white">Select Artist Image</h3>
          <button @click="showCandidates = false" class="text-neutral-400 hover:text-white">
            <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <div v-if="loadingCandidates" class="text-center py-8 text-neutral-400">
          Loading candidates...
        </div>

        <div v-else-if="candidates.length === 0" class="text-center py-8 text-neutral-400">
          No images found. Try again later.
        </div>

        <div v-else class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-4">
          <div 
            v-for="(candidate, idx) in candidates" 
            :key="idx"
            @click="selectCandidate(candidate)"
            class="cursor-pointer rounded-lg overflow-hidden border-2 border-transparent hover:border-primary-500 transition-colors"
          >
            <img 
              :src="candidate.thumbnail || candidate.url" 
              class="w-full aspect-square object-cover"
              :alt="candidate.source"
            />
            <div class="p-2 bg-neutral-800 text-xs text-neutral-400">
              {{ candidate.source }}
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, computed, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useMpdStore } from '@/stores/mpd'

const router = useRouter()
const route = useRoute()
const mpdStore = useMpdStore()

const artistName = computed(() => route.query.artist || '')
const albums = ref([])
const artistImageUrl = ref('')
const showCandidates = ref(false)
const candidates = ref([])
const loadingCandidates = ref(false)

const getArtistImageUrl = (name) => {
  const encodedArtist = encodeURIComponent(name).replace(/%2F/g, '/')
  return `/folder/${encodedArtist}/Artist.jpg`
}

const albumCount = computed(() => albums.value.length)

const loadArtistData = async () => {
  if (!artistName.value) return

  // Check for existing image
  const url = getArtistImageUrl(artistName.value)
  const img = new Image()
  img.onload = () => {
    artistImageUrl.value = url
  }
  img.onerror = () => {
    // No image, will trigger auto-fetch
    fetchAndCacheArtistImage()
  }
  img.src = url

  // Load albums for this artist
  try {
    const response = await fetch(`/api/albums?artist=${encodeURIComponent(artistName.value)}`)
    const data = await response.json()
    if (data.success && data.data) {
      albums.value = data.data.map(a => a.album)
    }
  } catch (e) {
    console.error('Failed to load albums:', e)
  }
}

const fetchAndCacheArtistImage = async () => {
  try {
    const response = await fetch(`/api/artistart/candidates?artist=${encodeURIComponent(artistName.value)}`)
    const data = await response.json()
    if (data.success && data.data && data.data.length > 0) {
      const firstCandidate = data.data[0]
      await fetch('/api/artistart/apply', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          artist: artistName.value,
          imageUrl: firstCandidate.url
        })
      })
      artistImageUrl.value = getArtistImageUrl(artistName.value)
    }
  } catch (e) {
    console.error('Failed to fetch artist image:', e)
  }
}

const fetchCandidates = async () => {
  loadingCandidates.value = true
  try {
    const response = await fetch(`/api/artistart/candidates?artist=${encodeURIComponent(artistName.value)}`)
    const data = await response.json()
    if (data.success) {
      candidates.value = data.data || []
    }
  } catch (e) {
    console.error('Failed to fetch candidates:', e)
  } finally {
    loadingCandidates.value = false
  }
}

const selectCandidate = async (candidate) => {
  try {
    await fetch('/api/artistart/apply', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        artist: artistName.value,
        imageUrl: candidate.url
      })
    })
    artistImageUrl.value = candidate.url
    showCandidates.value = false
  } catch (e) {
    console.error('Failed to apply candidate:', e)
  }
}

const goBack = () => {
  router.back()
}

const navigateToAlbum = (album) => {
  router.push({ name: 'album-detail', params: { artist: artistName.value, album } })
}

// Watch for modal open
watch(() => showCandidates.value, (newVal) => {
  if (newVal && candidates.value.length === 0) {
    fetchCandidates()
  }
})

onMounted(() => {
  loadArtistData()
})
</script>