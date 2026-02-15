<template>
  <div class="space-y-6">
    <div class="flex justify-between items-center">
      <h1 class="text-3xl font-bold text-white">Genres & Dates Matrix</h1>
      <div class="text-neutral-400 text-sm">
        Click any cell to view matching albums
        <span v-if="isLoadingAlbums" class="ml-2 text-blue-400">(Loading...)</span>
      </div>
    </div>

    <div v-if="loading" class="flex justify-center items-center py-20">
      <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-500"></div>
    </div>

    <div v-else-if="error" class="bg-red-900/50 border border-red-700 rounded-lg p-4 text-red-200">
      {{ error }}
    </div>

    <div v-else class="bg-neutral-800 rounded-lg border border-neutral-700 overflow-hidden">
      <!-- Matrix Container with scroll -->
      <div class="matrix-container">
        <table class="matrix-table">
          <thead>
            <tr>
              <th class="sticky-header sticky-col bg-neutral-900">Genre / Year</th>
              <th 
                v-for="date in matrixData.dates" 
                :key="date"
                class="sticky-header bg-neutral-900 text-neutral-400 font-medium"
              >
                {{ date }}
              </th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="genre in matrixData.genres" :key="genre">
              <td class="sticky-col bg-neutral-800 text-neutral-300 font-medium">
                {{ genre || 'Unknown' }}
              </td>
              <td 
                v-for="date in matrixData.dates" 
                :key="`${genre}-${date}`"
                class="matrix-cell"
                :class="{ 
                  'has-albums': getCount(genre, date) > 0,
                  'empty': !getCount(genre, date)
                }"
                @click="handleCellClick(genre, date)"
              >
                <span v-if="getCount(genre, date) > 0" class="count">
                  {{ getCount(genre, date) }}
                </span>
                <span v-else class="empty-indicator">-</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Legend -->
    <div class="flex items-center gap-6 text-sm text-neutral-400">
      <div class="flex items-center gap-2">
        <div class="w-4 h-4 bg-blue-600/30 rounded"></div>
        <span>Has albums (click to view)</span>
      </div>
      <div class="flex items-center gap-2">
        <div class="w-4 h-4 bg-neutral-800 rounded"></div>
        <span>No albums</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useMpdStore } from '@/stores/mpdStore'

const router = useRouter()
const mpdStore = useMpdStore()

const loading = ref(true)
const error = ref(null)

// Use computed matrix from store (Phase 1)
const matrixData = computed(() => mpdStore.genreDateMatrix)
const isLoadingAlbums = computed(() => mpdStore.isLoadingAlbums)

const getCount = (genre, date) => {
  return matrixData.value.matrix[genre]?.[date] || 0
}

const handleCellClick = (genre, date) => {
  const count = getCount(genre, date)
  if (count > 0) {
    // Navigate to search with genre and date filters
    router.push({ 
      name: 'search', 
      query: { 
        genre: genre,
        date: date
      } 
    })
  }
}

// Wait for albums to load if not already loaded
onMounted(() => {
  // If albums are already loaded, we're ready
  if (mpdStore.allAlbums.length > 0) {
    loading.value = false
    return
  }
  
  // Otherwise, wait for them to load
  const checkInterval = setInterval(() => {
    if (!mpdStore.isLoadingAlbums && mpdStore.allAlbums.length > 0) {
      loading.value = false
      clearInterval(checkInterval)
    } else if (!mpdStore.isLoadingAlbums && mpdStore.allAlbums.length === 0) {
      // Try loading albums
      mpdStore.loadAllAlbums()
    }
  }, 100)
  
  // Timeout after 30 seconds
  setTimeout(() => {
    clearInterval(checkInterval)
    if (loading.value) {
      loading.value = false
      error.value = 'Failed to load album data. Please refresh the page.'
    }
  }, 30000)
})
</script>

<style scoped>
.matrix-container {
  max-height: 70vh;
  overflow: auto;
  position: relative;
}

.matrix-table {
  border-collapse: separate;
  border-spacing: 0;
  width: auto;
  min-width: 100%;
}

/* Sticky headers */
.sticky-header {
  position: sticky;
  top: 0;
  z-index: 10;
  padding: 12px 16px;
  text-align: center;
  font-size: 0.875rem;
  border-bottom: 1px solid #374151;
  white-space: nowrap;
}

.sticky-col {
  position: sticky;
  left: 0;
  z-index: 20;
  padding: 12px 16px;
  text-align: left;
  font-size: 0.875rem;
  border-right: 1px solid #374151;
  white-space: nowrap;
  min-width: 150px;
}

/* Corner cell (first th) needs higher z-index */
.sticky-header.sticky-col {
  z-index: 30;
}

/* Table cells */
.matrix-cell {
  padding: 12px 16px;
  text-align: center;
  font-size: 0.875rem;
  border-bottom: 1px solid #374151;
  border-right: 1px solid #374151;
  cursor: pointer;
  transition: all 0.15s ease;
  min-width: 70px;
}

.matrix-cell.has-albums {
  background-color: rgba(37, 99, 235, 0.2);
  color: #60a5fa;
}

.matrix-cell.has-albums:hover {
  background-color: rgba(37, 99, 235, 0.4);
}

.matrix-cell.empty {
  background-color: #1f2937;
  color: #4b5563;
  cursor: default;
}

.matrix-cell .count {
  font-weight: 600;
}

.matrix-cell .empty-indicator {
  opacity: 0.3;
}

/* Body rows */
tbody tr:hover .sticky-col {
  background-color: #262626;
}

/* Scrollbar styling */
.matrix-container::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

.matrix-container::-webkit-scrollbar-track {
  background: #1f2937;
}

.matrix-container::-webkit-scrollbar-thumb {
  background: #4b5563;
  border-radius: 4px;
}

.matrix-container::-webkit-scrollbar-thumb:hover {
  background: #6b7280;
}
</style>
