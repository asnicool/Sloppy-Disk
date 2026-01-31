<template>
  <div class="space-y-6">
    <div class="flex justify-between items-center">
      <h1 class="text-3xl font-bold text-white">Release Dates</h1>
    </div>

    <div v-if="loading" class="flex justify-center items-center py-20">
      <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-500"></div>
    </div>

    <div v-else class="space-y-8">
      <div v-for="group in groups" :key="group.key" class="space-y-4">
        <h2 class="text-2xl font-semibold text-primary-400 border-b border-neutral-800 pb-2 cursor-pointer hover:text-primary-300 transition-colors"
            @click="navigateToDate(group.key)"
            :title="'Click to view all albums from ' + (group.key || 'Unknown Date')"
        >
          {{ group.key || 'Unknown Date' }}
        </h2>
        
        <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-4">
          <div 
            v-for="album in group.albums" 
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
    </div>

    <!-- Pagination -->
    <div v-if="!loading && totalPages > 1" class="flex justify-center items-center space-x-4 pt-8">
      <button 
        @click="changePage(currentPage - 1)"
        :disabled="currentPage === 1"
        class="px-4 py-2 bg-neutral-800 text-neutral-300 rounded-md hover:bg-neutral-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
      >
        Previous
      </button>
      <span class="text-neutral-400">Page {{ currentPage }} of {{ totalPages }}</span>
      <button 
        @click="changePage(currentPage + 1)"
        :disabled="currentPage === totalPages"
        class="px-4 py-2 bg-neutral-800 text-neutral-300 rounded-md hover:bg-neutral-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
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

const groups = ref([])
const loading = ref(true)
const currentPage = ref(1)
const totalPages = ref(1)
const limit = 20

const fetchGroups = async () => {
  loading.value = true
  try {
    const res = await mpdStore.fetchAlbumsByDate(currentPage.value, limit)
    if (res.success) {
      groups.value = res.data
      totalPages.value = Math.ceil((res.meta?.total || 1) / limit)
    }
  } catch (err) {
    console.error('Failed to fetch dates:', err)
  } finally {
    loading.value = false
  }
}

const changePage = (newPage) => {
  if (newPage >= 1 && newPage <= totalPages.value) {
    currentPage.value = newPage
    fetchGroups()
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }
}

const navigateToAlbum = (albumName) => {
  // We don't have the artist here easily with current grouped logic, 
  // but we can search for the album or use a generic search.
  // For now, let's just go to search with the album name.
  router.push({ name: 'search', query: { q: albumName } })
}

const navigateToDate = (date) => {
  router.push({ name: 'search', query: { q: date, type: 'date' } })
}

onMounted(fetchGroups)
</script>
