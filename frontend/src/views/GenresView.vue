<template>
  <div class="space-y-6">
    <div class="flex flex-col space-y-2">
      <h1 class="text-3xl font-bold text-white">Genre × Date Matrix</h1>
      <p class="text-neutral-400">
        Interactive matrix showing album counts across genres and years.
        <span class="text-primary-400">Click</span> row/col headers to toggle •
        <span class="text-primary-400">Double-click</span> for highest ranked cell
      </p>
      <div class="text-sm text-neutral-500" v-if="toggledGenres.length > 0 || toggledDates.length > 0">
        Active filters:
        <span v-if="toggledGenres.length > 0" class="text-primary-400">
          Genres: {{ toggledGenres.join(', ') }}
        </span>
        <span v-if="toggledDates.length > 0" class="text-primary-400 ml-2">
          Dates: {{ toggledDates.join(', ') }}
        </span>
        <button
          @click="clearAllToggles"
          class="ml-3 text-xs text-neutral-400 hover:text-white underline"
        >
          Clear all
        </button>
      </div>
    </div>

    <div v-if="loading" class="flex justify-center items-center py-20">
      <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-500"></div>
    </div>

    <div v-else-if="error" class="bg-red-900/50 border border-red-700 rounded-lg p-4 text-red-200">
      {{ error }}
    </div>

    <div v-else-if="matrixData.data.length > 0" class="space-y-4">
      <div class="overflow-auto max-h-[70vh]">
        <table class="w-full text-sm border-collapse" ref="matrixTable">
          <thead class="sticky top-0 z-50">
            <tr class="bg-neutral-800">
              <th class="px-3 py-2 text-left text-neutral-400 font-semibold border border-neutral-700 sticky left-0 bg-neutral-800 z-50 min-w-[120px]">Genre</th>
              <th
                v-for="date in visibleDates"
                :key="'date-col-' + date"
                :ref="(el) => setCellRef('date-col-' + date, el)"
                @click="toggleDate(date)"
                @dblclick="navigateToHighestInColumn(date)"
                class="px-3 py-2 text-center font-semibold border cursor-pointer transition-colors min-w-[80px]"
                :class="{
                  'text-neutral-300 hover:bg-primary-900/30': !isDateToggled(date),
                  'text-primary-400 bg-primary-900/60 hover:bg-primary-900/80': isDateToggled(date)
                }"
                :title="isDateToggled(date) ? 'Click to show all' : 'Click: Toggle this column • Double-click: Highest ranked'"
              >
                {{ date }}
                <span v-if="isDateToggled(date)" class="ml-1 text-xs">✓</span>
              </th>
              <th class="px-3 py-2 text-center text-primary-400 font-bold border border-neutral-700 bg-neutral-800 min-w-[80px]">Total</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="row in visibleRows"
              :key="'genre-row-' + row.genre"
              class="hover:bg-neutral-700/20 transition-colors"
            >
              <td
                :ref="(el) => setCellRef('genre-row-' + row.genre, el)"
                @click="toggleGenre(row.genre)"
                @dblclick="navigateToHighestInRow(row.genre)"
                class="px-3 py-2 font-medium border cursor-pointer hover:bg-primary-900/20 transition-colors sticky left-0 bg-neutral-800 z-50 relative"
                :class="{
                  'text-neutral-300': !isGenreToggled(row.genre),
                  'text-primary-400 bg-primary-900/80': isGenreToggled(row.genre)
                }"
                :title="isGenreToggled(row.genre) ? 'Click to show all' : 'Click: Toggle this row • Double-click: Highest ranked'"
              >
                {{ row.genre }}
                <span v-if="isGenreToggled(row.genre)" class="ml-1 text-xs">✓</span>
              </td>
              <td
                v-for="date in visibleDates"
                :key="'cell-' + row.genre + '-' + date"
                :ref="(el) => setCellRef('cell-' + row.genre + '-' + date, el)"
                @click="handleCellClick(row.genre, date, row.totals[date])"
                class="px-3 py-2 text-center border transition-colors cursor-pointer"
                :class="{
                  'bg-primary-900/40 text-primary-300 font-semibold': row.totals[date] > 0,
                  'text-neutral-600': row.totals[date] === 0
                }"
                :title="row.totals[date] > 0 ? `${row.totals[date]} albums • Click to view` : 'No albums'"
              >
                {{ row.totals[date] || '' }}
              </td>
              <td class="px-3 py-2 text-center text-primary-400 font-bold border bg-neutral-800/50">
                {{ row.rowTotal }}
              </td>
            </tr>
          </tbody>
          <tfoot>
            <tr class="bg-neutral-800">
              <td class="px-3 py-2 text-primary-400 font-bold border sticky left-0 bg-neutral-800 z-50">Total</td>
              <td
                v-for="date in visibleDates"
                :key="'col-total-' + date"
                class="px-3 py-2 text-center text-primary-400 font-bold border"
              >
                {{ matrixData.columnTotals[date] || 0 }}
              </td>
              <td class="px-3 py-2 text-center text-primary-300 font-bold border bg-primary-900/50">
                {{ grandTotalVisible }}
              </td>
            </tr>
          </tfoot>
        </table>
      </div>

      <!-- Legend -->
      <div class="flex flex-wrap items-center gap-6 text-sm text-neutral-400">
        <div class="flex items-center gap-2">
          <div class="w-4 h-4 bg-primary-900/40 rounded"></div>
          <span>Has albums (click to view)</span>
        </div>
        <div class="flex items-center gap-2">
          <div class="w-4 h-4 bg-neutral-800 rounded border border-neutral-700"></div>
          <span>No albums</span>
        </div>
        <div class="flex items-center gap-2">
          <div class="w-4 h-4 bg-primary-900/60 border border-primary-500 rounded"></div>
          <span>Toggled filter</span>
        </div>
        <div class="flex items-center gap-2">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 15l-2 5L9 9l11 4-5 2zm0 0l5 5M7.188 2.239l.777 2.897M5.136 7.965l-2.898-.777M13.95 4.05l-2.122 2.122m-5.657 5.656l-2.12 2.122" />
          </svg>
          <span>Click row/col to toggle</span>
        </div>
        <div class="flex items-center gap-2">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
          </svg>
          <span>Double-click for highest ranked</span>
        </div>
      </div>
    </div>

    <div v-else class="flex flex-col items-center justify-center py-20 text-center border-2 border-dashed border-neutral-800 rounded-3xl">
      <svg class="w-16 h-16 text-neutral-800 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19V6l12-3v13M9 19c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zm12-3c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zM9 10l12-3" />
      </svg>
      <p class="text-neutral-600 font-medium">No genre/date data available</p>
      <p class="text-neutral-700 text-sm mt-2">Load some albums first</p>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useMpdStore } from '@/stores/mpd'
import { albumList } from '@/services/albumList'

const router = useRouter()
const mpdStore = useMpdStore()

const loading = ref(true)
const error = ref(null)
const matrixTable = ref(null)
const cellRefs = ref({})

// Toggle state
const toggledGenres = ref([])
const toggledDates = ref([])

// Toggle functions
const isGenreToggled = (genre) => {
  return toggledGenres.value.includes(genre)
}

const isDateToggled = (date) => {
  return toggledDates.value.includes(date)
}

const toggleGenre = (genre) => {
  const index = toggledGenres.value.indexOf(genre)
  if (index > -1) {
    toggledGenres.value.splice(index, 1)
  } else {
    toggledGenres.value.push(genre)
  }
}

const toggleDate = (date) => {
  const index = toggledDates.value.indexOf(date)
  if (index > -1) {
    toggledDates.value.splice(index, 1)
  } else {
    toggledDates.value.push(date)
  }
}

const clearAllToggles = () => {
  toggledGenres.value = []
  toggledDates.value = []
}

// Compute visible rows and columns based on toggles
const visibleRows = computed(() => {
  const matrix = matrixData.value
  if (!matrix.data.length) return []

  // If no filters active, show all rows
  if (toggledDates.value.length === 0) {
    return matrix.data
  }

  // Show rows that have at least one non-empty cell in visible columns
  return matrix.data.filter(row => {
    return toggledDates.value.some(date => row.totals[date] > 0)
  })
})

const visibleDates = computed(() => {
  const matrix = matrixData.value
  if (!matrix.dates.length) return []

  // If no filters active, show all columns
  if (toggledGenres.value.length === 0) {
    return matrix.dates
  }

  // Show columns that have at least one non-empty cell in visible rows
  return matrix.dates.filter(date => {
    return matrix.data.some(row => {
      return toggledGenres.value.includes(row.genre) && row.totals[date] > 0
    })
  })
})

const grandTotalVisible = computed(() => {
  let total = 0
  visibleRows.value.forEach(row => {
    visibleDates.value.forEach(date => {
      total += row.totals[date] || 0
    })
  })
  return total
})

// Build genre × date matrix from albumList
const matrixData = computed(() => {
  // Try to get albums from mpdStore first, fall back to albumList
  let albums = mpdStore.allAlbums || []

  if (albums.length === 0 && albumList.isLoaded()) {
    albums = albumList.getAlbums()
  }

  if (!albums || albums.length === 0) {
    return { data: [], dates: [], columnTotals: {}, grandTotal: 0 }
  }

  // Build genre x date matrix (case-insensitive for genres)
  const matrix = {} // { genre (lowercase): { originalGenre: string, totals: { date: count } } }
  const allDates = new Set()

  albums.forEach(album => {
    // Skip albums without date - they shouldn't appear in the matrix
    if (!album.date || album.date.trim() === '') {
      return
    }

    const genre = String(album.genre || 'Unknown').trim()
    const genreLower = genre.toLowerCase()
    const date = album.date.trim()

    if (!matrix[genreLower]) {
      matrix[genreLower] = {
        originalGenre: genre,
        totals: {}
      }
    }

    if (!matrix[genreLower].totals[date]) {
      matrix[genreLower].totals[date] = 0
    }
    matrix[genreLower].totals[date]++
    allDates.add(date)
  })

  // Sort dates descending (newest first)
  const sortedDates = Array.from(allDates).sort((a, b) => b.localeCompare(a))

  // Calculate column totals and build data array
  const columnTotals = {}
  const data = []
  let grandTotal = 0

  // Initialize column totals
  sortedDates.forEach(date => {
    columnTotals[date] = 0
  })

  // Build row data
  Object.keys(matrix).sort().forEach(genreLower => {
    const rowData = matrix[genreLower]
    const row = {
      genre: rowData.originalGenre,
      totals: {},
      rowTotal: 0
    }

    sortedDates.forEach(date => {
      const count = rowData.totals[date] || 0
      row.totals[date] = count
      row.rowTotal += count
      columnTotals[date] += count
      grandTotal += count
    })

    data.push(row)
  })

  return {
    data,
    dates: sortedDates,
    columnTotals,
    grandTotal
  }
})

// Helper function to set cell refs
const setCellRef = (key, el) => {
  if (el) {
    cellRefs.value[key] = el
  }
}

// Cell click handler - navigate to search with filters
const handleCellClick = (genre, date, count) => {
  if (count > 0) {
    router.push({
      name: 'search',
      query: {
        genre: genre,
        date: date
      }
    })
  }
}

// Scroll to first non-empty cell in a genre row
const scrollToFirstInRow = (genre) => {
  const matrix = matrixData.value
  const row = matrix.data.find(r => r.genre.toLowerCase() === genre.toLowerCase())

  if (!row) return

  for (const date of visibleDates.value) {
    if (row.totals[date] > 0) {
      const cellRefName = 'cell-' + row.genre + '-' + date
      scrollToCell(cellRefName)
      return
    }
  }
}

// Scroll to first non-empty cell in a date column
const scrollToFirstInColumn = (date) => {
  const matrix = matrixData.value

  for (const row of visibleRows.value) {
    if (row.totals[date] > 0) {
      const cellRefName = 'cell-' + row.genre + '-' + date
      scrollToCell(cellRefName)
      return
    }
  }
}

// Navigate to highest ranked cell in a genre row (most albums)
const navigateToHighestInRow = (genre) => {
  const matrix = matrixData.value
  const row = matrix.data.find(r => r.genre.toLowerCase() === genre.toLowerCase())

  if (!row) return

  let maxCount = 0
  let bestDate = null

  for (const date of matrix.dates) {
    if (row.totals[date] > maxCount) {
      maxCount = row.totals[date]
      bestDate = date
    }
  }

  if (bestDate && maxCount > 0) {
    router.push({
      name: 'search',
      query: {
        genre: row.genre,
        date: bestDate
      }
    })
  }
}

// Navigate to highest ranked cell in a date column (most albums)
const navigateToHighestInColumn = (date) => {
  const matrix = matrixData.value

  let maxCount = 0
  let bestGenre = null

  for (const row of matrix.data) {
    if (row.totals[date] > maxCount) {
      maxCount = row.totals[date]
      bestGenre = row.genre
    }
  }

  if (bestGenre && maxCount > 0) {
    router.push({
      name: 'search',
      query: {
        genre: bestGenre,
        date: date
      }
    })
  }
}

// Smooth scroll to cell with visual feedback
const scrollToCell = (refName) => {
  const element = cellRefs.value[refName]
  if (element) {
    element.scrollIntoView({ behavior: 'smooth', block: 'center', inline: 'center' })
    // Add highlight effect
    element.classList.add('ring-2', 'ring-primary-500', 'z-20')
    setTimeout(() => {
      element.classList.remove('ring-2', 'ring-primary-500', 'z-20')
    }, 1000)
  }
}

// Load albums on mount
onMounted(async () => {
  try {
    // Check if mpdStore has albums
    if (mpdStore.allAlbums && mpdStore.allAlbums.length > 0) {
      loading.value = false
      return
    }

    // Try loading from albumList
    if (!albumList.isLoaded()) {
      await albumList.loadAlbums()
    }

    if (albumList.isLoaded() && albumList.getAlbums().length > 0) {
      loading.value = false
    } else {
      error.value = 'No albums found. Please add some music to your library.'
    }
  } catch (err) {
    console.error('[GenresView] Failed to load albums:', err)
    error.value = 'Failed to load album data: ' + err.message
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.text-primary-400 { color: #60a5fa; }
.bg-primary-900 { background-color: #1e3a8a; }
.bg-primary-500 { background-color: #3b82f6; }
.border-primary-500 { border-color: #3b82f6; }
.border-primary-700 { border-color: #1d4ed8; }
.ring-primary-500 { --tw-ring-color: #3b82f6; }

/* Table styling */
table {
  border-collapse: separate;
  border-spacing: 0;
  min-width: 100%;
}

/* Genre column - responsive width */
thead th:first-child,
tbody td:first-child,
tfoot td:first-child {
  min-width: 120px;
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Mobile: limit genre column width */
@media (max-width: 640px) {
  thead th:first-child,
  tbody td:first-child,
 tfoot td:first-child {
    min-width: 100px;
    max-width: 100px;
    font-size: 0.75rem;
    padding: 8px;
  }

  thead th:not(:first-child),
  tbody td:not(:first-child),
  tfoot td:not(:first-child) {
    min-width: 60px;
    font-size: 0.7rem;
    padding: 6px 4px;
  }
}

/* Table border colors */
.border {
  border-color: #374151;
}

/* Scrollbar styling for overflow-auto container */
.overflow-auto::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

.overflow-auto::-webkit-scrollbar-track {
  background: #1f2937;
  border-radius: 4px;
}

.overflow-auto::-webkit-scrollbar-thumb {
  background: #4b5563;
  border-radius: 4px;
}

.overflow-auto::-webkit-scrollbar-thumb:hover {
  background: #6b7280;
}

/* Smooth transitions */
* {
  transition-property: color, background-color, border-color;
  transition-timing-function: cubic-bezier(0.4, 0, 0.2, 1);
  transition-duration: 150ms;
}
</style>
