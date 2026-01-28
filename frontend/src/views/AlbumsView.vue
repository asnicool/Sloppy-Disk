<template>
  <div class="space-y-6">
    <h1 class="text-3xl font-bold text-white">Albums</h1>
    <div class="flex justify-between items-center">
      <div class="flex space-x-2">
        <button 
          @click="sortMode = 'name'" 
          :class="{ 'bg-blue-600': sortMode === 'name', 'bg-neutral-700': sortMode !== 'name' }"
          class="px-4 py-2 rounded-lg text-white"
        >
          Sort by Name
        </button>
        <button 
          @click="sortMode = 'date'" 
          :class="{ 'bg-blue-600': sortMode === 'date', 'bg-neutral-700': sortMode !== 'date' }"
          class="px-4 py-2 rounded-lg text-white"
        >
          Sort by Date
        </button>
        <button 
          @click="sortMode = 'random'; if(sortMode === 'random') loadAlbums()" 
          :class="{ 'bg-blue-600': sortMode === 'random', 'bg-neutral-700': sortMode !== 'random' }"
          class="px-4 py-2 rounded-lg text-white"
        >
          Random
        </button>
      </div>
      <button 
        v-if="sortMode === 'random'"
        @click="loadAlbums"
        class="px-4 py-2 bg-neutral-700 text-white rounded-lg hover:bg-neutral-600 flex items-center gap-2"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
        </svg>
        Refresh
      </button>
    </div>
    <div v-if="loading" class="text-neutral-400">Loading albums...</div>
    <div v-else class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-6">
      <AlbumCard 
        v-for="album in albums" 
        :key="album.id || (album.album + album.artist)"
        :album="album.album"
        :artist="album.artist || 'Unknown'"
        :cover-url="album.coverUrl"
        :date="album.date"
        :genre="album.genre"
        :album-details="album"
      />
    </div>

    <!-- Pagination Controls -->
    <div v-if="!loading && albums.length > 0 && sortMode !== 'random'" class="flex justify-center items-center space-x-4 mt-6">
      <button 
        @click="prevPage" 
        :disabled="currentPage <= 1"
        class="px-4 py-2 bg-neutral-700 text-white rounded-lg hover:bg-neutral-600 disabled:opacity-50 disabled:cursor-not-allowed"
      >
        Previous
      </button>
      <span class="text-white">Page {{ currentPage }} of {{ totalPages }}</span>
      <button 
        @click="nextPage" 
        :disabled="currentPage >= totalPages"
        class="px-4 py-2 bg-neutral-700 text-white rounded-lg hover:bg-neutral-600 disabled:opacity-50 disabled:cursor-not-allowed"
      >
        Next
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { useMpdStore } from '@/stores/mpd'
import AlbumCard from '@/components/AlbumCard.vue'

const mpdStore = useMpdStore()
const albums = ref([])
const loading = ref(true)
const currentPage = ref(1)
const totalPages = ref(1)
const totalAlbums = ref(0)
const hasMore = ref(false)
const itemsPerPage = 36
const sortMode = ref('random') // 'name', 'date', 'random'

const loadAlbums = async () => {
  loading.value = true
  try {
    if (sortMode.value === 'date' || sortMode.value === 'name') {
      const response = await mpdStore.fetchAlbums(currentPage.value, itemsPerPage, '', sortMode.value)
      if (response.success) {
        albums.value = response.data
        if (response.meta) {
          totalAlbums.value = response.meta.total || 0
          hasMore.value = response.meta.hasMore || false
          totalPages.value = Math.ceil(totalAlbums.value / itemsPerPage)
        } else {
          totalAlbums.value = albums.value.length
          totalPages.value = 1
        }
      }
    } else if (sortMode.value === 'random') {
      const response = await mpdStore.fetchRandomAlbums(itemsPerPage, true)
      if (response.success) {
        albums.value = response.data
        totalAlbums.value = albums.value.length
        totalPages.value = 1
        hasMore.value = false
      }
    }
  } finally {
    loading.value = false
  }
}

const nextPage = async () => {
  if (currentPage.value < totalPages.value) {
    currentPage.value++
    await loadAlbums()
  }
}

const prevPage = async () => {
  if (currentPage.value > 1) {
    currentPage.value--
    await loadAlbums()
  }
}

// Watch for changes in sortMode and reset to page 1
watch(sortMode, async () => {
  currentPage.value = 1
  await loadAlbums()
})


onMounted(async () => {
  await loadAlbums()
})
</script>
