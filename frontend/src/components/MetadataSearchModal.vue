<template>
  <Teleport to="body">
    <div v-if="isOpen" class="fixed inset-0 z-50 flex items-center justify-center">
      <!-- Backdrop -->
      <div class="absolute inset-0 bg-black/50" @click="$emit('close')"></div>

      <!-- Modal -->
      <div class="relative bg-gray-900 rounded-lg shadow-xl w-full max-w-4xl max-h-[90vh] overflow-hidden">
        <!-- Header -->
        <div class="flex items-center justify-between px-4 py-3 border-b border-gray-700">
          <h3 class="text-lg font-semibold text-white">Find Metadata</h3>
          <button @click="$emit('close')" class="text-gray-400 hover:text-white">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
            </svg>
          </button>
        </div>

        <!-- Content -->
        <div class="overflow-y-auto max-h-[calc(90vh-140px)] p-4">
          <!-- Search Form -->
          <div class="mb-4 flex gap-2">
            <input
              v-model="displayArtist"
              placeholder="Artist"
              class="flex-1 px-3 py-2 bg-gray-800 border border-gray-700 rounded text-white placeholder-gray-400"
            />
            <input
              v-model="displayAlbum"
              placeholder="Album"
              class="flex-1 px-3 py-2 bg-gray-800 border border-gray-700 rounded text-white placeholder-gray-400"
            />
            <button
              @click="handleSearch"
              :disabled="loading || !displayArtist || !displayAlbum"
              class="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50"
            >
              {{ loading ? 'Searching...' : 'Search' }}
            </button>
          </div>

          <!-- Provider Filter -->
          <div class="mb-4 flex gap-2 flex-wrap">
            <button
              v-for="provider in providers"
              :key="provider"
              @click="toggleProvider(provider)"
              :class="[
                'px-3 py-1.5 text-sm font-medium rounded-full transition-all duration-200',
                isProviderSelected(provider)
                  ? 'bg-blue-600 text-white shadow-lg shadow-blue-600/30'
                  : 'bg-gray-800 text-gray-400 hover:bg-gray-700 hover:text-white'
              ]"
            >
              {{ provider }}
            </button>
          </div>

          <!-- Error -->
          <div v-if="error" class="mb-4 p-3 bg-red-900/50 border border-red-700 rounded text-red-200">
            {{ error }}
          </div>

          <!-- Results -->
          <div v-if="candidates.length > 0" class="space-y-2">
            <h4 class="text-sm font-medium text-gray-400">Search Results ({{ candidates.length }})</h4>
            
            <div
              v-for="(candidate, index) in candidates"
              :key="`${candidate.source}-${candidate.externalId}`"
              @click="selectCandidate(candidate)"
              :class="[
                'p-3 rounded cursor-pointer transition-colors',
                selectedCandidate?.externalId === candidate.externalId && selectedCandidate?.source === candidate.source
                  ? 'bg-blue-900/50 border border-blue-700'
                  : 'bg-gray-800 border border-gray-700 hover:bg-gray-750'
              ]"
            >
              <div class="flex justify-between items-start">
                <div>
                  <div class="font-medium text-white">{{ candidate.album }}</div>
                  <div class="text-sm text-gray-400">{{ candidate.artist }}</div>
                  <div class="text-xs text-gray-500 mt-1">
                    {{ candidate.year }} · {{ candidate.genre || 'No genre' }} · {{ candidate.source }}
                  </div>
                </div>
                <div class="text-right">
                  <div class="text-lg font-bold text-blue-400">{{ Math.round(candidate.confidence) }}%</div>
                  <div v-if="candidate.tracks" class="text-xs text-gray-500">
                    {{ candidate.tracks.length }} tracks
                  </div>
                </div>
              </div>

              <!-- Selected Details -->
              <div v-if="selectedCandidate?.externalId === candidate.externalId && selectedCandidate?.source === candidate.source" class="mt-3 pt-3 border-t border-gray-700">
                <div v-if="selectedCandidate.tracks && selectedCandidate.tracks.length > 0" class="mb-2">
                  <div class="text-xs text-gray-500 mb-1">Tracks:</div>
                  <div class="max-h-32 overflow-y-auto space-y-1">
                    <div v-for="track in selectedCandidate.tracks.slice(0, 10)" :key="track.track" class="text-sm text-gray-300 flex gap-2">
                      <span class="text-gray-500">{{ track.disc }}-{{ track.track }}</span>
                      <span class="flex-1 truncate">{{ track.title }}</span>
                    </div>
                    <div v-if="selectedCandidate.tracks.length > 10" class="text-xs text-gray-500">
                      ... and {{ selectedCandidate.tracks.length - 10 }} more
                    </div>
                  </div>
                </div>

                <!-- Cover Art -->
                <div v-if="coverArtOptions.length > 0" class="mb-2">
                  <div class="text-xs text-gray-500 mb-1">Cover Art:</div>
                  <div class="flex gap-2 flex-wrap">
                    <img
                      v-for="(art, idx) in coverArtOptions.slice(0, 4)"
                      :key="idx"
                      :src="art.thumbnail || art.url"
                      :class="[
                        'w-16 h-16 object-cover rounded cursor-pointer border-2',
                        selectedCoverArt?.url === art.url ? 'border-blue-500' : 'border-transparent'
                      ]"
                      @click.stop="selectedCoverArt = art"
                    />
                  </div>
                </div>

                <button
                  @click.stop="handleApply"
                  :disabled="applying"
                  class="w-full mt-2 px-4 py-2 bg-green-600 text-white rounded hover:bg-green-700 disabled:opacity-50"
                >
                  {{ applying ? 'Applying...' : 'Apply Metadata' }}
                </button>

                <div v-if="applyResult" class="mt-2 p-2 bg-green-900/30 border border-green-700 rounded text-sm text-green-200">
                  Applied to {{ applyResult.updatedFiles }} of {{ applyResult.totalFiles }} files
                  <span v-if="applyResult.errors && applyResult.errors.length > 0" class="text-yellow-400">
                    ({{ applyResult.errors.length }} errors)
                  </span>
                </div>
              </div>
            </div>
          </div>

          <!-- No Results -->
          <div v-else-if="searched && !loading" class="text-center py-8 text-gray-500">
            No metadata found. Try adjusting your search terms.
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { ref, watch, nextTick, computed } from 'vue'
import { useMetadata } from '../composables/useMetadata'

const props = defineProps({
  isOpen: Boolean,
  initialArtist: String,
  initialAlbum: String,
  albumPath: String
})

const emit = defineEmits(['close', 'applied'])

const {
  candidates,
  selectedCandidate,
  loading,
  error,
  applyResult,
  searchMetadata,
  getMetadataDetails,
  applyMetadata,
  searchCoverArt,
  clearSelection
} = useMetadata()

const searchArtist = ref('')
const searchAlbum = ref('')
const selectedProviders = ref(['MusicBrainz', 'Discogs', 'FreeDB'])
const providers = ['MusicBrainz', 'Discogs', 'FreeDB']
const coverArtOptions = ref([])
const selectedCoverArt = ref(null)
const applying = ref(false)
const searched = ref(false)

// Display value for inputs
const displayArtist = computed({
  get: () => searchArtist.value || props.initialArtist,
  set: (val) => { searchArtist.value = val }
})

const displayAlbum = computed({
  get: () => searchAlbum.value || props.initialAlbum,
  set: (val) => { searchAlbum.value = val }
})

const handleSearch = async () => {
  const artist = displayArtist.value
  const album = displayAlbum.value
  
  if (!artist || !album) return
  
  searched.value = true
  clearSelection()
  coverArtOptions.value = []
  selectedCoverArt.value = null
  
  await searchMetadata(artist, album, selectedProviders.value)
  
  // Also search for cover art
  coverArtOptions.value = await searchCoverArt(artist, album)
}

const selectCandidate = async (candidate) => {
  if (selectedCandidate.value?.externalId === candidate.externalId && selectedCandidate.value?.source === candidate.source) {
    // Deselect
    selectedCandidate.value = null
    return
  }
  
  selectedCandidate.value = candidate
  
  // Get detailed info
  await getMetadataDetails(candidate.source, candidate.externalId)
}

const handleApply = async () => {
  if (!selectedCandidate.value || !props.albumPath) return
  
  applying.value = true
  const result = await applyMetadata(
    props.albumPath,
    selectedCandidate.value,
    selectedCoverArt.value?.url || ''
  )
  applying.value = false
  
  if (result) {
    emit('applied', result)
  }
}

const toggleProvider = (provider) => {
  const index = selectedProviders.value.indexOf(provider)
  if (index === -1) {
    selectedProviders.value.push(provider)
  } else {
    selectedProviders.value.splice(index, 1)
  }
}

const isProviderSelected = (provider) => {
  return selectedProviders.value.includes(provider)
}

// Watch props to log changes
watch(() => props.initialArtist, (val) => {
  console.log('[MetadataSearchModal] initialArtist changed to:', val)
}, { immediate: true })

watch(() => props.initialAlbum, (val) => {
  console.log('[MetadataSearchModal] initialAlbum changed to:', val)
}, { immediate: true })

// Initialize with props - clear user edits when modal opens
watch(() => props.isOpen, (isOpen) => {
  if (isOpen) {
    console.log('[MetadataSearchModal] Opening with artist:', props.initialArtist, 'album:', props.initialAlbum)
    // Clear any previous user edits
    searchArtist.value = ''
    searchAlbum.value = ''
    searched.value = false
    clearSelection()
  }
})
</script>
