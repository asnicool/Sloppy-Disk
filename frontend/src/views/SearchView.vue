<template>
  <div class="space-y-6 pb-24">
    <div class="flex flex-col space-y-2">
      <h1 class="text-3xl font-bold text-white">Advanced Search</h1>
      <p class="text-neutral-400">Search results stream in real-time as they are found across your library.</p>
    </div>

    <!-- Search Input -->
    <div class="relative group">
      <div class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
        <svg class="h-5 w-5 text-neutral-500 group-focus-within:text-primary-500 transition-colors" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
        </svg>
      </div>
      <input 
        v-model="query" 
        @input="onInput"
        type="text" 
        placeholder="Type at least 3 characters to search artists, albums, songs, genres..." 
        class="block w-full pl-10 pr-3 py-4 bg-neutral-800 border border-neutral-700 rounded-xl leading-5 text-neutral-200 placeholder-neutral-500 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500 transition-all text-lg"
      />
      <div v-if="isSearching" class="absolute inset-y-0 right-0 pr-4 flex items-center">
        <div class="animate-spin h-5 w-5 border-2 border-primary-500 border-t-transparent rounded-full"></div>
      </div>
    </div>

    <!-- Category Filters -->
    <div class="flex flex-wrap gap-2 pb-2 overflow-x-auto scrollbar-hide">
      <button 
        v-for="cat in categories" 
        :key="cat.id"
        @click="selectCategory(cat.id)"
        class="px-4 py-1.5 rounded-full text-sm font-medium transition-all whitespace-nowrap border"
        :class="activeCategory === cat.id 
          ? 'bg-primary-600 text-white border-primary-500 shadow-lg shadow-primary-900/20' 
          : 'bg-neutral-800 text-neutral-400 border-neutral-700 hover:bg-neutral-700 hover:text-neutral-200'"
      >
        {{ cat.name }}
      </button>
    </div>

    <!-- Results Sections -->
    <div v-if="hasAnyResults" class="space-y-10">
      
      <!-- Artists -->
      <section v-if="results.artists.length" class="space-y-4">
        <h2 class="text-xl font-bold text-primary-400 flex items-center">
          <span class="mr-2">Artists</span>
          <span class="px-2 py-0.5 bg-neutral-800 text-xs rounded-full text-neutral-400">{{ results.artists.length }}</span>
        </h2>
        <div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-6 gap-4">
          <div 
            v-for="artist in results.artists" 
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
        </div>
      </section>

      <!-- Albums -->
      <section v-if="results.albums.length" class="space-y-4">
        <h2 class="text-xl font-bold text-primary-400 flex items-center">
          <span class="mr-2">Albums</span>
          <span class="px-2 py-0.5 bg-neutral-800 text-xs rounded-full text-neutral-400">{{ results.albums.length }}</span>
        </h2>
        <div class="grid grid-cols-2 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-6 gap-6">
          <AlbumCard 
            v-for="album in sortedAlbums" 
            :key="album.id || album.album"
            :album="album.album"
            :artist="album.artist"
            :cover-url="album.coverUrl"
            :date="album.date"
            :genre="album.genre"
          />
        </div>
      </section>

      <!-- Songs -->
      <section v-if="results.songs.length" class="space-y-4">
        <h2 class="text-xl font-bold text-primary-400 flex items-center">
          <span class="mr-2">Songs</span>
          <span class="px-2 py-0.5 bg-neutral-800 text-xs rounded-full text-neutral-400">{{ results.songs.length }}</span>
        </h2>
        <div class="bg-neutral-800/30 rounded-xl overflow-hidden border border-neutral-800">
          <div 
            v-for="(song, idx) in results.songs" 
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
          </div>
        </div>
      </section>

      <!-- Genres & Dates (Smaller grids) -->
       <div class="grid grid-cols-1 md:grid-cols-2 gap-8">
          <section v-if="results.genres.length" class="space-y-4">
            <h2 class="text-lg font-bold text-primary-300">Genres</h2>
            <div class="flex flex-wrap gap-2">
              <button 
                v-for="genre in results.genres" 
                :key="genre"
                @click="navigateToGenre(genre)"
                class="px-3 py-1.5 bg-neutral-800 hover:bg-neutral-700 text-sm text-neutral-300 rounded-full transition-colors border border-neutral-700"
              >
                {{ genre }}
              </button>
            </div>
          </section>

          <section v-if="results.dates.length" class="space-y-4">
            <h2 class="text-lg font-bold text-primary-300">Dates</h2>
            <div class="flex flex-wrap gap-2">
              <button 
                v-for="date in results.dates" 
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
    <div v-else-if="searched && query.length >= 3" class="flex flex-col items-center justify-center py-20 text-center">
        <svg class="w-16 h-16 text-neutral-700 mb-4" fill="none" stroke="currentColor" viewBox="0 0 14 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.172 9.172a4 4 0 015.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
        <p class="text-neutral-400 text-lg">No results found for "{{ query }}"</p>
        <p class="text-neutral-500 text-sm">Try searching for something else or check your spelling.</p>
    </div>
    
    <div v-else class="flex flex-col items-center justify-center py-20 text-center border-2 border-dashed border-neutral-800 rounded-3xl">
        <svg class="w-16 h-16 text-neutral-800 mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 12h.01M12 12h.01M16 12h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
        <p class="text-neutral-600 font-medium">Your search results will appear here as they are discovered</p>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onUnmounted, onMounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useMpdStore } from '@/stores/mpd'
import { debounce } from 'lodash-es'
import AlbumCard from '@/components/AlbumCard.vue'

const router = useRouter()
const route = useRoute()
const mpdStore = useMpdStore()

const query = ref('')
const searched = ref(false)
const activeCategory = ref('')

const categories = [
  { id: '', name: 'All' },
  { id: 'artists', name: 'Artists' },
  { id: 'albums', name: 'Albums' },
  { id: 'songs', name: 'Songs' },
  { id: 'genres', name: 'Genres' },
  { id: 'dates', name: 'Dates' }
]

const selectCategory = (catId) => {
  activeCategory.value = catId
  if (query.value.length >= 3) {
    performSearch(query.value, false, catId)
  }
}

const results = computed(() => {
  const base = mpdStore.searchResults || {}
  return {
    albums: Array.isArray(base.albums) ? base.albums : [],
    artists: Array.isArray(base.artists) ? base.artists : [],
    genres: Array.isArray(base.genres) ? base.genres : [],
    dates: Array.isArray(base.dates) ? base.dates : [],
    songs: Array.isArray(base.songs) ? base.songs : []
  }
})

// Sort albums by date (newest first)
const sortedAlbums = computed(() => {
  const albums = results.value.albums || []
  return [...albums].sort((a, b) => {
    // Handle dates - prefer newer first
    const dateA = a.date || ''
    const dateB = b.date || ''
    // Try to parse as year (4 digits) or full date
    const yearA = parseInt(dateA.substring(0, 4)) || 0
    const yearB = parseInt(dateB.substring(0, 4)) || 0
    return yearB - yearA
  })
})
const isSearching = computed(() => mpdStore.isSearching)
const hasAnyResults = computed(() => {
  return (results.value?.albums?.length || 0) > 0 || 
         (results.value?.artists?.length || 0) > 0 || 
         (results.value?.genres?.length || 0) > 0 || 
         (results.value?.dates?.length || 0) > 0 || 
         (results.value?.songs?.length || 0) > 0
})

const onInput = debounce(() => {
  if (query.value.length >= 3) {
    performSearch(query.value, false, activeCategory.value)
  } else {
    searched.value = false
  }
}, 300)

const performSearch = (searchTerm, exact = false, category = '') => {
  searched.value = true
  mpdStore.triggerStreamingSearch(searchTerm, exact, category)
}

const handleRouteQuery = () => {
  const q = route.query.q
  const type = route.query.type || ''
  if (q && q.length >= 3) {
    query.value = q
    performSearch(q, !!type, type)
  }
}

onMounted(() => {
  handleRouteQuery()
})

watch(() => route.query.q, () => {
  handleRouteQuery()
})

const playSong = (song) => {
  mpdStore.addToPlaylist(song.path)
}

const navigateToAlbum = (album) => {
  router.push({ name: 'search', query: { q: `${album.artist} ${album.album}` } })
}

const navigateToArtist = (artist) => {
  query.value = artist
  performSearch(artist, true)
}

const navigateToGenre = (genre) => {
  query.value = genre
  performSearch(genre, true)
}

const navigateToDate = (date) => {
  query.value = date
  performSearch(date, true)
}

onUnmounted(() => {
  // Hide loading spinner if we leave
  mpdStore.isSearching = false
})
</script>

<style scoped>
.text-primary-400 { color: #60a5fa; }
.text-primary-300 { color: #93c5fd; }
.bg-primary-500 { background-color: #3b82f6; }
.border-primary-500 { border-color: #3b82f6; }
.focus-ring-primary-500:focus { --tw-ring-color: #3b82f6; }
</style>
