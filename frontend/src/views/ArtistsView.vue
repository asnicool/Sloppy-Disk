<template>
  <div class="space-y-6">
    <h1 class="text-3xl font-bold text-white">Artists</h1>
    <div v-if="loading" class="text-gray-400">Loading artists...</div>
    <div v-else class="space-y-8">
      <div v-for="artistGroup in artists" :key="artistGroup.artist" class="space-y-4">
        <h2 class="text-2xl font-semibold text-primary-400 border-b border-gray-800 pb-2 flex items-center">
          <svg class="w-6 h-6 mr-2 text-gray-400" fill="currentColor" viewBox="0 0 20 20">
            <path fill-rule="evenodd" d="M10 9a3 3 0 100-6 3 3 0 000 6zm-7 9a7 7 0 1114 0H3z" clip-rule="evenodd" />
          </svg>
          {{ artistGroup.artist || 'Unknown Artist' }}
        </h2>
        
        <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-4">
          <div 
            v-for="album in artistGroup.albums" 
            :key="album"
            @click="navigateToAlbum(artistGroup.artist, album)"
            class="group bg-gray-900/50 rounded-lg p-3 hover:bg-gray-800 transition-all cursor-pointer border border-gray-800 hover:border-primary-500/50"
          >
            <div class="aspect-square bg-gray-800 rounded-md mb-2 flex items-center justify-center overflow-hidden">
               <svg class="w-12 h-12 text-gray-700" fill="currentColor" viewBox="0 0 20 20">
                <path d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zM7 8H5v2h2V8zm2 0h2v2H9V8zm6 0h-2v2h2V8z" />
              </svg>
            </div>
            <p class="text-sm font-medium text-gray-200 truncate group-hover:text-primary-400">{{ album }}</p>
          </div>
        </div>
      </div>
    </div>

    <!-- Pagination Controls -->
    <div v-if="!loading && artists.length > 0" class="flex justify-center items-center space-x-4 mt-6">
      <button 
        @click="prevPage" 
        :disabled="currentPage <= 1"
        class="px-4 py-2 bg-gray-700 text-white rounded-lg hover:bg-gray-600 disabled:opacity-50 disabled:cursor-not-allowed"
      >
        Previous
      </button>
      <span class="text-white">Page {{ currentPage }} of {{ totalPages }}</span>
      <button 
        @click="nextPage" 
        :disabled="currentPage >= totalPages"
        class="px-4 py-2 bg-gray-700 text-white rounded-lg hover:bg-gray-600 disabled:opacity-50 disabled:cursor-not-allowed"
      >
        Next
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useMpdStore } from '@/stores/mpd'

const router = useRouter()
const mpdStore = useMpdStore()
const artists = ref([])
const loading = ref(true)
const currentPage = ref(1)
const totalPages = ref(1)
const totalArtists = ref(0)
const hasMore = ref(false)
const itemsPerPage = 50

const loadArtists = async () => {
  loading.value = true
  try {
    const response = await mpdStore.fetchArtists(currentPage.value, itemsPerPage)
    if (response.success) {
      artists.value = response.data
      if (response.meta) {
        totalArtists.value = response.meta.total || 0
        hasMore.value = response.meta.hasMore || false
        totalPages.value = Math.ceil(totalArtists.value / itemsPerPage)
      }
    }
  } finally {
    loading.value = false
  }
}

const nextPage = async () => {
  if (currentPage.value < totalPages.value) {
    currentPage.value++
    await loadArtists()
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }
}

const prevPage = async () => {
  if (currentPage.value > 1) {
    currentPage.value--
    await loadArtists()
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }
}

const navigateToAlbum = (artist, album) => {
  router.push({ name: 'search', query: { q: `${artist} ${album}` } })
}

onMounted(async () => {
 await loadArtists()
})
</script>
