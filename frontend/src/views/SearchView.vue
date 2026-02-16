<template>
  <div class="space-y-6 pb-24" @click="showHistory = false">
    <div class="flex flex-col space-y-2">
      <h1 class="text-3xl font-bold text-white">Advanced Search</h1>
      <p class="text-neutral-400">Search results stream in real-time. Use multiple criteria for better precision.</p>
    </div>

    <!-- Search Input Area -->
    <div class="space-y-4">
        <!-- Input Container -->
        <div class="relative group z-20">
            <div class="absolute top-3 left-3 flex items-start pointer-events-none pt-1">
                <svg class="h-5 w-5 text-neutral-500 group-focus-within:text-primary-500 transition-colors" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                </svg>
            </div>
            
            <div class="flex flex-col w-full bg-neutral-800 border border-neutral-700 rounded-xl focus-within:ring-2 focus-within:ring-primary-500 focus-within:border-primary-500 transition-all overflow-hidden min-h-[56px]">
                <!-- Search Chips (Top Line) -->
                <div v-if="searchChips.length > 0" class="flex flex-wrap gap-2 px-10 pt-3 pb-1">
                    <span 
                        v-for="(chip, index) in searchChips" 
                        :key="index" 
                        class="bg-primary-900/50 text-white text-sm px-2 py-1 rounded-md flex items-center border border-primary-700/50 animate-fadeIn"
                    >
                        {{ chip }}
                        <button @click.stop="removeChip(index)" class="ml-1.5 hover:text-primary-300 focus:outline-none">
                            <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path></svg>
                        </button>
                    </span>
                </div>

                <div class="flex items-center w-full">
                    <!-- Input Field -->
                    <input 
                        ref="searchInput"
                        v-model="query" 
                        @input="onInput"
                        @keydown.enter="addChipFromInput"
                        type="text" 
                        placeholder="Type to search..." 
                        class="flex-1 bg-transparent border-none focus:ring-0 text-neutral-200 placeholder-neutral-500 text-lg py-3 pl-10 pr-12 outline-none h-full"
                    />

                    <!-- Blue Circle Button -->
                    <div class="pr-3 flex items-center self-center absolute right-0 top-0 bottom-0 pointer-events-auto">
                        <button 
                            @click.stop="handleCircleClick"
                            @dblclick.stop="handleCircleDblClick"
                            @contextmenu.prevent="handleCircleLongPress"
                            v-long-press="handleCircleLongPress"
                            class="w-8 h-8 rounded-full bg-primary-600 hover:bg-primary-500 text-white flex items-center justify-center shadow-lg transition-all transform hover:scale-105 active:scale-95 focus:outline-none ring-2 ring-transparent focus:ring-primary-400"
                            title="Click: Add criteria | DblClick: Clear | Long Press: History"
                        >
                            <svg v-if="query.length > 0" class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"></path></svg>
                            <svg v-else class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"></path></svg>
                        </button>
                    </div>
                </div>
            </div>

            <!-- History Dropdown -->
            <div v-show="showHistory && searchHistory.length > 0" class="absolute top-full right-0 mt-2 w-64 bg-neutral-800 rounded-xl shadow-xl border border-neutral-700 overflow-hidden z-30">
                <div class="px-4 py-2 bg-neutral-900 border-b border-neutral-700 text-xs font-semibold text-neutral-400 uppercase tracking-wider">
                    Recent Searches
                </div>
                <ul class="max-h-60 overflow-y-auto">
                    <li v-for="(item, idx) in searchHistory" :key="idx">
                        <button 
                            @click.stop="applyHistoryItem(item)"
                            class="w-full text-left px-4 py-3 text-sm text-neutral-300 hover:bg-neutral-700/50 transition-colors flex items-center justify-between group"
                        >
                            <span class="truncate">{{ item.join(' + ') }}</span>
                            <span class="opacity-0 group-hover:opacity-100 text-primary-400">
                                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                                </svg>
                            </span>
                        </button>
                    </li>
                </ul>
            </div>
        </div>

        <!-- Controls: Fuzzy Toggle & Categories -->
        <div class="flex flex-col space-y-4">
             <!-- Toggle Switch -->
            <div class="flex items-center justify-end px-1">
                 <button 
                    @click="isStrict = !isStrict"
                    class="flex items-center space-x-3 group focus:outline-none"
                    role="switch"
                    :aria-checked="isStrict"
                 >
                    <span class="text-sm font-medium transition-colors" :class="!isStrict ? 'text-primary-400' : 'text-neutral-500'">Fuzzy</span>
                    <div class="relative inline-flex items-center h-6 rounded-full w-11 transition-colors focus:outline-none border-2 border-transparent"
                         :class="isStrict ? 'bg-primary-600' : 'bg-neutral-700'"
                    >
                        <span class="sr-only">Toggle Strict Search</span>
                        <span
                            class="translate-x-0 inline-block w-5 h-5 transform bg-white rounded-full transition-transform ease-in-out duration-200"
                            :class="isStrict ? 'translate-x-5' : 'translate-x-0'"
                        />
                    </div>
                    <span class="text-sm font-medium transition-colors" :class="isStrict ? 'text-primary-400' : 'text-neutral-500'">Strict</span>
                 </button>
            </div>

            <!-- Category Filters (Wrapped) -->
            <div class="flex flex-wrap gap-2 justify-center sm:justify-start">
            <button 
                v-for="cat in categories" 
                :key="cat.id"
                @click="selectCategory(cat.id)"
                :disabled="!hasActiveSearch"
                class="px-4 py-2 rounded-full text-sm font-medium transition-all whitespace-nowrap border flex-grow sm:flex-grow-0 text-center"
                :class="[
                    activeCategory === cat.id 
                    ? 'bg-primary-600 text-white border-primary-500 shadow-lg shadow-primary-900/20' 
                    : 'bg-neutral-800 text-neutral-400 border-neutral-700 hover:bg-neutral-700 hover:text-neutral-200',
                    !hasActiveSearch ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer'
                ]"
            >
                {{ cat.name }}
            </button>
            </div>
        </div>
    </div>

    <!-- Results Sections -->
    <div v-if="hasAnyResults" class="space-y-10 pt-4">
      
      <!-- Artists -->
      <section v-if="shouldShowSection('artists') && localArtists.length" class="space-y-4">
        <h2 class="text-xl font-bold text-primary-400 flex items-center border-b border-neutral-800 pb-2">
          <span class="mr-2">Artists</span>
          <span class="px-2 py-0.5 bg-neutral-800 text-xs rounded-full text-neutral-400">{{ localArtists.length }}</span>
        </h2>
        <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-4">
          <div 
            v-for="artist in localArtists.slice(0, displayLimitArtists)" 
            :key="artist"
            @click="navigateToArtist(artist)"
            class="bg-neutral-800/40 p-3 rounded-lg hover:bg-neutral-700/60 transition-colors cursor-pointer text-center group border border-neutral-800"
          >
            <div class="w-12 h-12 bg-neutral-700 rounded-full mx-auto mb-2 flex items-center justify-center text-neutral-400 group-hover:text-primary-300 transition-colors">
              <svg class="w-6 h-6" fill="currentColor" viewBox="0 0 20 20">
                <path fill-rule="evenodd" d="M10 9a3 3 0 100-6 3 3 0 000 6zm-7 9a7 7 0 1114 0H3z" clip-rule="evenodd" />
              </svg>
            </div>
            <p class="text-sm font-medium text-neutral-300 truncate">{{ artist }}</p>
          </div>
          <!-- Show More Button -->
          <button 
            v-if="localArtists.length > displayLimitArtists"
            @click="displayLimitArtists += 50"
            class="bg-neutral-800/20 p-3 rounded-lg hover:bg-neutral-700/40 transition-all cursor-pointer flex flex-col items-center justify-center group border border-dashed border-neutral-700/50 hover:border-primary-500/50"
          >
            <div class="w-10 h-10 rounded-full bg-neutral-800 flex items-center justify-center text-primary-400 group-hover:bg-primary-900/30 transition-all">
                <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"/></svg>
            </div>
            <span class="mt-2 text-xs text-neutral-500 group-hover:text-primary-400">Show More</span>
          </button>
        </div>
      </section>

      <!-- Albums -->
      <section v-if="shouldShowSection('albums') && localAlbums.length" class="space-y-4">
        <h2 class="text-xl font-bold text-primary-400 flex items-center border-b border-neutral-800 pb-2">
          <span class="mr-2">Albums</span>
          <span class="px-2 py-0.5 bg-neutral-800 text-xs rounded-full text-neutral-400">{{ localAlbums.length }}</span>
        </h2>
        <div class="grid grid-cols-2 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-6 gap-6">
          <AlbumCard 
            v-for="album in localAlbums.slice(0, displayLimitAlbums)" 
            :key="album.id || album.album"
            :album="album.album"
            :artist="album.artist"
            :cover-url="album.coverUrl"
            :date="album.date"
            :genre="album.genre"
          />
          <!-- Show More Button -->
          <button 
            v-if="localAlbums.length > displayLimitAlbums"
            @click="displayLimitAlbums += 50"
            class="aspect-[4/5] bg-neutral-800/20 rounded-xl hover:bg-neutral-700/40 transition-all cursor-pointer flex flex-col items-center justify-center group border border-dashed border-neutral-700/50 hover:border-primary-500/50"
          >
            <div class="w-12 h-12 rounded-full bg-neutral-800 flex items-center justify-center text-primary-400 group-hover:bg-primary-900/30 transition-all">
                <svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"/></svg>
            </div>
            <span class="mt-3 text-sm text-neutral-500 group-hover:text-primary-400 font-medium">Show More</span>
          </button>
        </div>
      </section>

      <!-- Songs (from MPD search) -->
      <section v-if="shouldShowSection('songs') && sortedSongs.length" class="space-y-4">
        <div class="flex items-center justify-between border-b border-neutral-800 pb-2">
            <h2 class="text-xl font-bold text-primary-400 flex items-center">
            <span class="mr-2">Songs</span>
            <span class="px-2 py-0.5 bg-neutral-800 text-xs rounded-full text-neutral-400">{{ sortedSongs.length }}</span>
            </h2>
             <div v-if="isSearching" class="animate-spin h-5 w-5 border-2 border-primary-500 border-t-transparent rounded-full"></div>
        </div>
        
        <div class="bg-neutral-800/30 rounded-xl overflow-hidden border border-neutral-800">
          <div 
            v-for="(song, idx) in sortedSongs.slice(0, displayLimitSongs)" 
            :key="song.path + idx"
            @click="playSong(song)"
            class="flex items-center px-4 py-3 hover:bg-neutral-700/40 cursor-pointer transition-colors border-b border-neutral-800/50 last:border-0 group"
          >
            <div class="w-8 text-neutral-500 text-xs text-center group-hover:hidden">{{ idx + 1 }}</div>
            <div class="w-8 hidden group-hover:flex items-center justify-center text-primary-400">
              <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
                <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM9.555 7.168A1 1 0 008 8v4a1 1 0 001.555.832l3-2a1 1 0 000-1.664l-3-2z" clip-rule="evenodd" />
              </svg>
            </div>
            <div class="flex-1 min-w-0 ml-2">
              <p class="text-sm font-medium text-neutral-200 truncate">{{ song.title || 'Unknown Title' }}</p>
              <p class="text-xs text-neutral-500 truncate">{{ song.artist }} • {{ song.album }}</p>
            </div>
            <div v-if="song._relevance && song._relevance < 1" class="text-xs text-neutral-600">
              {{ Math.round(song._relevance * 100) }}%
            </div>
          </div>
          <!-- Show More Button -->
          <button 
            v-if="sortedSongs.length > displayLimitSongs"
            @click.stop="displayLimitSongs += 100"
            class="w-full py-4 text-sm text-neutral-500 hover:text-primary-400 hover:bg-neutral-700/20 transition-all font-medium flex items-center justify-center space-x-2 border-t border-neutral-800"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6v6m0 0v6m0-6h6m-6 0H6"/></svg>
            <span>Show more songs</span>
          </button>
        </div>
      </section>

      <!-- Quick filters: Genres & Dates (from local search) -->
      <div v-if="shouldShowSection('genres') || shouldShowSection('dates')" class="grid grid-cols-1 md:grid-cols-2 gap-8">
        <section v-if="shouldShowSection('genres') && localGenres.length" class="space-y-4">
          <div class="flex items-center justify-between">
            <h2 class="text-lg font-bold text-primary-300 border-b border-neutral-800 pb-2">Genres</h2>
            <a href="#genre-matrix-link" class="text-xs text-primary-400 hover:text-primary-300" @click.prevent="router.push('/genreXdate')">
              View matrix →
            </a>
          </div>
          <div class="flex flex-wrap gap-2">
            <button
              v-for="genre in localGenres"
              :key="genre"
              @click="navigateToGenre(genre)"
              class="px-3 py-1.5 bg-neutral-800 hover:bg-neutral-700 text-sm text-neutral-300 rounded-full transition-colors border border-neutral-700"
            >
              {{ genre }}
            </button>
          </div>
        </section>

        <section v-if="shouldShowSection('dates') && localDates.length" class="space-y-4">
          <div class="flex items-center justify-between">
            <h2 class="text-lg font-bold text-primary-300 border-b border-neutral-800 pb-2">Dates</h2>
            <a href="#genre-matrix-link" class="text-xs text-primary-400 hover:text-primary-300" @click.prevent="router.push('/genreXdate')">
              View matrix →
            </a>
          </div>
          <div class="flex flex-wrap gap-2">
            <button
              v-for="date in localDates"
              :key="date"
              @click="navigateToDate(date)"
              class="px-3 py-1.5 bg-neutral-800 hover:bg-neutral-700 text-sm text-neutral-300 rounded-full transition-colors border border-neutral-700"
            >
              {{ date }}
            </button>
          </div>
        </section>
      </div>
    </div>

    <!-- Empty State -->
    <div v-else-if="hasActiveSearch" class="flex flex-col items-center justify-center py-20 text-center">
        <svg class="w-16 h-16 text-neutral-700 mb-4" fill="none" stroke="currentColor" viewBox="0 0 14 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.172 9.172a4 4 0 015.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
        <p class="text-neutral-400 text-lg">No results found for your criteria</p>
        <p class="text-neutral-500 text-sm">Try using different keywords or disable "Strict Search"</p>
    </div>
    
    <div v-else class="flex flex-col items-center justify-center py-20 text-center border-2 border-dashed border-neutral-800 rounded-3xl opacity-50">
        <svg class="w-16 h-16 text-neutral-800 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 12h.01M12 12h.01M16 12h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
        <p class="text-neutral-600 font-medium">Add criteria to start searching</p>
        <p class="text-neutral-700 text-sm mt-2">Start typing...</p>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onUnmounted, onMounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useMpdStore } from '@/stores/mpd'
import { albumList } from '@/services/albumList'
import { debounce } from 'lodash-es'
import AlbumCard from '@/components/AlbumCard.vue'
import { sortByRelevance, sortByDateDesc, filterByExactMatch } from '@/utils/fuzzyMatch'

// Simple directive for long press
const vLongPress = {
  mounted(el, binding) {
    let pressTimer = null
    const start = (e) => {
      if (e.type === 'click' && e.button !== 0) return
      if (pressTimer === null) {
        pressTimer = setTimeout(() => {
          binding.value(e)
          pressTimer = null
        }, 500) // 500ms for long press
      }
    }
    const cancel = () => {
      if (pressTimer !== null) {
        clearTimeout(pressTimer)
        pressTimer = null
      }
    }
    el.addEventListener('mousedown', start)
    el.addEventListener('touchstart', start)
    el.addEventListener('click', cancel)
    el.addEventListener('mouseout', cancel)
    el.addEventListener('touchend', cancel)
    el.addEventListener('touchcancel', cancel)
  }
}

const router = useRouter()
const route = useRoute()
const mpdStore = useMpdStore()
const searchInput = ref(null)

const query = ref('')
const searchChips = ref([])
const activeCategory = ref('')
const isStrict = ref(false)
const showHistory = ref(false)
const searchHistory = ref([]) // Array of arrays (chip sets)
const isFromUrlLink = ref(false) // Track if search came from URL parameters
const chipFieldTypes = ref({}) // Track which field each chip represents: { chipText: 'genre'|'date'|'artist'|'album' }

const localAlbums = ref([])
const localArtists = ref([])
const localGenres = ref([])
const localDates = ref([])

// Display Limits
const displayLimitArtists = ref(30)
const displayLimitAlbums = ref(30)
const displayLimitSongs = ref(30)

const categories = [
  { id: '', name: 'All' },
  { id: 'artists', name: 'Artists' },
  { id: 'albums', name: 'Albums' },
  { id: 'songs', name: 'Songs' },
  { id: 'genres', name: 'Genres' },
  { id: 'dates', name: 'Dates' }
]

// --- Initialization ---

onMounted(async () => {
  // Load history from local storage
  try {
    const saved = localStorage.getItem('search_history')
    if (saved) {
      searchHistory.value = JSON.parse(saved)
    }
  } catch (e) {
    console.error('Failed to load search history', e)
  }

  if (!albumList.isLoaded()) {
    await albumList.loadAlbums()
  }
  handleRouteQuery()
})

onUnmounted(() => {
  mpdStore.isSearching = false
})

// --- Interaction Handlers ---

const selectCategory = (catId) => {
  if (!hasActiveSearch.value) return // Disable if no search
  activeCategory.value = catId
  // Re-trigger search logic just in case backend needs it (logic in triggerSearch handles it)
  triggerSearch()
}

const handleCircleClick = () => {
  if (query.value.trim()) {
    addChipFromInput()
  } else {
    // If empty, focus input
    searchInput.value?.focus()
  }
}

const handleCircleDblClick = () => {
  query.value = ''
  searchChips.value = []
  chipFieldTypes.value = {} // Clear field types
  isFromUrlLink.value = false // Reset URL link flag on manual clear
  triggerSearch()
}

const handleCircleLongPress = () => {
  showHistory.value = !showHistory.value
}

const addChipFromInput = () => {
  const trimmed = query.value.trim()
  if (trimmed) {
    searchChips.value.push(trimmed)
    query.value = ''
    addToHistory(searchChips.value)
    isFromUrlLink.value = false // Reset URL link flag on manual entry
    // Manual entry: no field type for this chip, clear all field types
    chipFieldTypes.value = {}
    triggerSearch()
  }
}

const removeChip = (index) => {
  const removedChip = searchChips.value[index]
  searchChips.value.splice(index, 1)
  // Remove field type entry for the removed chip
  if (removedChip && chipFieldTypes.value[removedChip]) {
    delete chipFieldTypes.value[removedChip]
  }
  isFromUrlLink.value = false // Reset URL link flag when modifying chips
  triggerSearch()
}

const applyHistoryItem = (chips) => {
  searchChips.value = [...chips]
  query.value = ''
  showHistory.value = false
  isFromUrlLink.value = false // Reset URL link flag when applying history
  // History items don't have field type info, clear all
  chipFieldTypes.value = {}
  triggerSearch()
}

const addToHistory = (chips) => {
  // Deep clone and simple dedupe check
  const newEntry = [...chips]
  const existingIndex = searchHistory.value.findIndex(h => 
    h.length === newEntry.length && h.every((val, i) => val === newEntry[i])
  )
  
  if (existingIndex > -1) {
    // Move to top
    searchHistory.value.splice(existingIndex, 1)
  }
  
  searchHistory.value.unshift(newEntry)
  if (searchHistory.value.length > 10) {
    searchHistory.value.pop()
  }
  
  localStorage.setItem('search_history', JSON.stringify(searchHistory.value))
}

const onInput = debounce(() => {
  if (query.value.length >= 3 || searchChips.value.length > 0) {
     triggerSearch()
  } else if (query.value.length === 0 && searchChips.value.length === 0) {
     triggerSearch() // Clear results
  }
}, 500) // Rate limit timer 0.5 sec

watch(isStrict, () => {
  triggerSearch()
})

watch(() => route.query, () => {
    handleRouteQuery()
}, { deep: true })

// --- Search Logic ---

const hasActiveSearch = computed(() => {
  return query.value.trim().length > 0 || searchChips.value.length > 0
})

const getCombinedTerms = () => {
  const terms = [...searchChips.value]
  if (query.value.trim()) {
      terms.push(query.value.trim())
  }
  return terms
}

const triggerSearchRaw = () => {
  const terms = getCombinedTerms()
  
  if (terms.length === 0) {
    localAlbums.value = []
    localArtists.value = []
    localGenres.value = []
    localDates.value = []
    mpdStore.setSearchResults({ songs: [] }) // Clear MPD results
    return
  }

  // Reset display limits on new search
  displayLimitArtists.value = 30
  displayLimitAlbums.value = 30
  displayLimitSongs.value = 30
  
  // 1. Local Search (Albums, Artists, Genres, Dates)
  performLocalSearch(terms)
  
  // 2. MPD Search (Songs)
  performMpdSearch(terms.join(' '))
}

const triggerSearch = debounce(triggerSearchRaw, 300)

const performLocalSearch = async (terms) => {
  if (!albumList.isLoaded()) {
    console.log('[Search] albumList not loaded, attempting load...')
    await albumList.loadAlbums()
  }

  // Get all albums initially
  let results = albumList.getAlbums()
  console.log(`[Search] Local search for: "${terms.join(', ')}", analyzing ${results.length} albums`)

  if (isStrict.value) {
    // Check if we have field-specific filtering (from URL parameters or matrix clicks)
    const hasFieldSpecificFilters = terms.some(term => chipFieldTypes.value[term])

    if (hasFieldSpecificFilters) {
      // Field-specific filtering: each chip only searches in its designated field
      console.log('[Search] Using field-specific filtering:', chipFieldTypes.value)
      terms.forEach(term => {
        const fieldType = chipFieldTypes.value[term]
        if (fieldType) {
          // Filter only in the specified field
          const lowerTerm = term.toLowerCase()
          results = results.filter(album => {
            const fieldValue = String(album[fieldType] || '').toLowerCase()
            // Exact match OR starts-with in the specified field only
            return fieldValue === lowerTerm || fieldValue.startsWith(lowerTerm)
          })
        } else {
          // No field type specified for this chip, search all fields
          const lowerTerm = term.toLowerCase()
          results = results.filter(album => {
            const artist = String(album.artist || '').toLowerCase()
            const albumName = String(album.album || '').toLowerCase()
            const genre = String(album.genre || '').toLowerCase()
            const date = String(album.date || '').toLowerCase()

            return artist === lowerTerm || albumName === lowerTerm || genre === lowerTerm || date === lowerTerm ||
                   artist.startsWith(lowerTerm) || albumName.startsWith(lowerTerm) ||
                   genre.startsWith(lowerTerm) || date.startsWith(lowerTerm)
          })
        }
      })
    } else {
      // Standard strict filtering: each term can match in any field
      terms.forEach(term => {
        const lowerTerm = term.toLowerCase()
        results = results.filter(album => {
          const artist = String(album.artist || '').toLowerCase()
          const albumName = String(album.album || '').toLowerCase()
          const genre = String(album.genre || '').toLowerCase()
          const date = String(album.date || '').toLowerCase()

          // Exact match OR starts-with for all fields
          return artist === lowerTerm || albumName === lowerTerm || genre === lowerTerm || date === lowerTerm ||
                 artist.startsWith(lowerTerm) || albumName.startsWith(lowerTerm) ||
                 genre.startsWith(lowerTerm) || date.startsWith(lowerTerm)
        })
      })
    }
    // In strict mode, the manual filter above already did all the work
    // No need to run Fuse.js which would join terms and fail to match multi-field criteria

    // Sort by artist, then date, then genre when coming from URL link parameters (genre=, date=, etc.)
    if (isFromUrlLink.value) {
      results.sort((a, b) => {
        // Primary: Artist name (alphabetical)
        const artistA = String(a.artist || '').toLowerCase()
        const artistB = String(b.artist || '').toLowerCase()
        if (artistA !== artistB) {
          return artistA.localeCompare(artistB)
        }

        // Secondary: Date (newest first)
        const dateA = a.date || ''
        const dateB = b.date || ''
        if (dateA !== dateB) {
          // Extract year for comparison
          const yearA = parseInt(dateA.substring(0, 4)) || 0
          const yearB = parseInt(dateB.substring(0, 4)) || 0
          if (yearA !== yearB) {
            return yearB - yearA // Newest first
          }
          return dateB.localeCompare(dateA) // Full date comparison if same year
        }

        // Tertiary: Genre (alphabetical)
        const genreA = String(a.genre || '').toLowerCase()
        const genreB = String(b.genre || '').toLowerCase()
        return genreA.localeCompare(genreB)
      })
      console.log(`[Search] URL link strict mode: sorted ${results.length} albums by artist → date → genre`)
    }
  } else {
    // Fuzzy mode: Use Fuse.js for relevance scoring and ranking
    const weightedFields = [
      { name: 'album', weight: 1.0 },
      { name: 'artist', weight: 0.8 },
      { name: 'genre', weight: 0.4 },
      { name: 'date', weight: 0.2 }
    ]
    results = sortByRelevance(results, terms, weightedFields, isStrict.value)
    console.log(`[Search] Fuzzy match returned ${results.length} albums`)
  }

  localAlbums.value = results
  
  // Extract derived lists with relevance sorting for artists
  const artistSet = new Set()
  const genreSet = new Set()
  const dateSet = new Set()
  
  // Use a smaller subset for deriving filters to stay fast
  const filterSummaryData = results.slice(0, 500)
  filterSummaryData.forEach(album => {
    if (album.artist) artistSet.add(album.artist)
    if (album.genre) genreSet.add(album.genre)
    if (album.date) dateSet.add(album.date)
  })
  
  // For artists, we also want to sort by relevance to the query
  let artists = Array.from(artistSet).map(a => ({ name: a }))
  if (artists.length > 0) {
    artists = sortByRelevance(artists, terms, ['name'], isStrict.value)
    localArtists.value = artists.map(a => a.name)
  } else {
    localArtists.value = []
  }
  
  localGenres.value = Array.from(genreSet).sort()
  localDates.value = Array.from(dateSet).sort((a, b) => b.localeCompare(a))
}

const performMpdSearch = (searchString) => {
  let type = ''
  if (activeCategory.value === 'songs') type = 'title'
  if (activeCategory.value === 'artists') type = 'artist'
  if (activeCategory.value === 'albums') type = 'album'
  
  mpdStore.triggerStreamingSearch(searchString, isStrict.value, type)
}

// --- Results Display Logic ---

const mpdResults = computed(() => {
  const base = mpdStore.searchResults || {}
  return {
    songs: Array.isArray(base.songs) ? base.songs : []
  }
})

const sortedSongs = computed(() => {
  const songs = mpdResults.value.songs || []
  const terms = getCombinedTerms()
  
  return sortByRelevance(songs, terms, ['title', 'artist', 'album', 'date', 'genre'], isStrict.value)
})

const hasAnyResults = computed(() => {
  if (!hasActiveSearch.value) return false
  
  if (shouldShowSection('albums') && localAlbums.value.length > 0) return true
  if (shouldShowSection('artists') && localArtists.value.length > 0) return true
  if (shouldShowSection('genres') && localGenres.value.length > 0) return true
  if (shouldShowSection('dates') && localDates.value.length > 0) return true
  if (shouldShowSection('songs') && sortedSongs.value.length > 0) return true
  
  return false
})

const shouldShowSection = (sectionName) => {
  if (!activeCategory.value) return true // Show all if no category selected
  return activeCategory.value === sectionName
}

const isSearching = computed(() => mpdStore.isSearching)

const handleRouteQuery = () => {
    const q = route.query.q
    const type = route.query.type || ''
    const strictParam = route.query.strict

    // Handle direct field parameters (genre, date, artist, album)
    const genreParam = route.query.genre
    const dateParam = route.query.date
    const artistParam = route.query.artist
    const albumParam = route.query.album

    // Reset chip field types
    chipFieldTypes.value = {}

    if (q) {
        // Legacy q parameter handling
        query.value = q
        if (type && categories.some(c => c.id === type)) {
            activeCategory.value = type
        }
        // "from a clicked link, exact match by default"
        // If strict is specifically set in query, use it, else default to TRUE for links
        isStrict.value = strictParam !== undefined ? (strictParam === 'true' || strictParam === true) : true

        // Reset chips if we come from a direct link search
        searchChips.value = []
        chipFieldTypes.value = {}
        isFromUrlLink.value = false // Not a direct field parameter link

        triggerSearch()
    } else if (genreParam || dateParam || artistParam || albumParam) {
        // Handle genre/date/artist/album parameters with field tracking
        const paramChips = []
        if (genreParam) {
          paramChips.push(String(genreParam))
          chipFieldTypes.value[String(genreParam)] = 'genre'
        }
        if (dateParam) {
          paramChips.push(String(dateParam))
          chipFieldTypes.value[String(dateParam)] = 'date'
        }
        if (artistParam) {
          paramChips.push(String(artistParam))
          chipFieldTypes.value[String(artistParam)] = 'artist'
        }
        if (albumParam) {
          paramChips.push(String(albumParam))
          chipFieldTypes.value[String(albumParam)] = 'album'
        }

        searchChips.value = paramChips
        query.value = ''
        activeCategory.value = '' // Show all results (not limited to one category)

        // Direct field parameters = strict mode by default
        isStrict.value = strictParam !== undefined ? (strictParam === 'true' || strictParam === true) : true
        isFromUrlLink.value = true // Mark as URL link for date sorting

        triggerSearch()
    }
}

// --- Navigation ---

const playSong = (song) => {
  mpdStore.addToPlaylist(song.path)
}

const navigateToArtist = (artist) => {
  searchChips.value = [artist]
  chipFieldTypes.value = { [artist]: 'artist' }
  activeCategory.value = 'artists'
  query.value = ''
  isStrict.value = true // Link = Strict
  isFromUrlLink.value = true // From internal link = sort by date
  triggerSearch()
}

const navigateToGenre = (genre) => {
  searchChips.value = [genre]
  chipFieldTypes.value = { [genre]: 'genre' }
  activeCategory.value = 'genres'
  query.value = ''
  isStrict.value = true // Link = Strict
  isFromUrlLink.value = true // From internal link = sort by date
  triggerSearch()
}

const navigateToDate = (date) => {
  searchChips.value = [date]
  chipFieldTypes.value = { [date]: 'date' }
  activeCategory.value = 'dates'
  query.value = ''
  isStrict.value = true // Link = Strict
  isFromUrlLink.value = true // From internal link = sort by date
  triggerSearch()
}
</script>

<style scoped>
.text-primary-400 { color: #60a5fa; }
.text-primary-300 { color: #93c5fd; }
.bg-primary-900 { background-color: #1e3a8a; }
.bg-primary-600 { background-color: #2563eb; }
.bg-primary-500 { background-color: #3b82f6; }
.border-primary-500 { border-color: #3b82f6; }
.border-primary-700 { border-color: #1d4ed8; }
.focus-ring-primary-500:focus { --tw-ring-color: #3b82f6; }

/* Scrollbar hide utility */
.scrollbar-hide::-webkit-scrollbar {
    display: none;
}
.scrollbar-hide {
    -ms-overflow-style: none;
    scrollbar-width: none;
}

/* Animations */
@keyframes fadeIn {
    from { opacity: 0; transform: translateY(-5px); }
    to { opacity: 1; transform: translateY(0); }
}
.animate-fadeIn {
    animation: fadeIn 0.2s ease-out;
}
</style>
