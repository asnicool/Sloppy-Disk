<template>
  <div class="bg-gray-800 border-t border-gray-700 p-3 shadow-2xl z-50">
    <div class="max-w-7xl mx-auto">
      <div class="flex items-center gap-4">
        <!-- Song Info & CoverArt -->
        <div class="flex items-center space-x-3 overflow-hidden min-w-0 flex-grow-[2]">
          <router-link 
            to="/nowplaying"
            class="w-12 h-12 bg-neutral-700 rounded flex-shrink-0 flex items-center justify-center overflow-hidden border border-neutral-600 hover:border-blue-500 transition-colors group relative"
            title="Now Playing"
          >
            <img v-if="coverUrl" :src="coverUrl" class="w-full h-full object-cover group-hover:opacity-75 transition-opacity" />
            <svg v-else class="w-6 h-6 text-neutral-500" fill="currentColor" viewBox="0 0 20 20">
              <path d="M18 3a1 1 0 00-1.196-.98l-10 2A1 1 0 006 5v9.114A4.369 4.369 0 005 14c-1.657 0-3 .895-3 2s1.343 2 3 2 3-.895 3-2V7.82l8-1.6v5.894A4.369 4.369 0 0015 12c-1.657 0-3 .895-3 2s1.343 2 3 2 3-.895 3-2V3z" />
            </svg>
            <div class="absolute inset-0 flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity bg-black/20">
              <svg class="w-4 h-4 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
              </svg>
            </div>
          </router-link>
          
          <div class="min-w-0 flex-1 leading-tight">
            <h3 class="text-sm font-bold text-white truncate cursor-default">
              <span v-if="currentSong?.track" class="text-gray-400 mr-1.5 font-mono text-xs">{{ formatTrack(currentSong.track) }}</span>
              {{ currentSong?.title || 'No song playing' }}
            </h3>
            <div class="flex items-center text-xs text-gray-400 space-x-1 whitespace-nowrap overflow-hidden">
              <span 
                @click="searchBy(currentSong?.artist)"
                class="hover:text-blue-400 cursor-pointer transition-colors max-w-[120px] truncate"
              >{{ currentSong?.artist || 'Unknown Artist' }}</span>
              <span class="text-gray-600 px-0.5">•</span>
              <span 
                @click="searchBy(currentSong?.date)"
                v-if="currentSong?.date"
                class="hover:text-blue-400 cursor-pointer transition-colors"
                title="Search by Date"
              >{{ currentSong.date }}</span>
              <span v-if="currentSong?.date" class="mx-0.5">-</span>
              <span 
                @click="searchBy(currentSong?.album)"
                v-if="currentSong?.album"
                class="hover:text-blue-400 cursor-pointer transition-colors truncate"
              >{{ currentSong.album }}</span>
            </div>
          </div>
        </div>

        <!-- Progress (Inlined for compactness) -->
        <div class="hidden md:flex flex-col flex-grow-[3] min-w-[200px] px-2">
          <div class="flex items-center justify-between text-[10px] text-gray-500 mb-0.5 px-0.5">
            <span>{{ formatTime(currentTime) }}</span>
            <span>{{ formatTime(duration) }}</span>
          </div>
          <div class="w-full bg-gray-700/50 rounded-full h-1 relative group cursor-pointer">
            <div 
              class="bg-blue-500 h-full rounded-full transition-all ease-linear relative overflow-visible"
              :style="{ width: `${progressPercentage}%`, transitionDuration: transitionDuration }"
            >
              <div class="absolute right-0 top-1/2 -translate-y-1/2 w-2 h-2 bg-white rounded-full shadow-lg scale-0 group-hover:scale-100 transition-transform"></div>
            </div>
          </div>
        </div>

        <!-- Controls -->
        <div class="flex items-center space-x-1 shrink-0">
          <button 
            @click="previous" 
            class="p-2 rounded-full hover:bg-gray-700/50 text-gray-300 transition-colors"
            title="Previous"
          >
            <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
              <path d="M8.445 14.832A1 1 0 0010 14v-2.798l5.445 3.63A1 1 0 0017 14V6a1 1 0 00-1.555-.832L10 8.798V6a1 1 0 00-1.555-.832l-6 4a1 1 0 000 1.664l6 4z" />
            </svg>
          </button>

          <button 
            @click="isPlaying ? pause() : play()" 
            class="p-2.5 rounded-full bg-blue-600 hover:bg-blue-700 text-white shadow-lg transition-all active:scale-95"
          >
            <svg v-if="isPlaying" class="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
              <path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zM7 8a1 1 0 012 0v4a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v4a1 1 0 102 0V8a1 1 0 00-1-1z" clip-rule="evenodd" />
            </svg>
            <svg v-else class="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
              <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM9.555 7.168A1 1 0 008 8v4a1 1 0 001.555.832l3-2a1 1 0 000-1.664l-3-2z" clip-rule="evenodd" />
            </svg>
          </button>

          <button 
            @click="next" 
            class="p-2 rounded-full hover:bg-gray-700/50 text-gray-300 transition-colors"
            title="Next"
          >
            <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 20 20">
              <path d="M4.555 5.168A1 1 0 003 6v8a1 1 0 001.555.832L10 11.202V14a1 1 0 001.555.832l6-4a1 1 0 000-1.664l-6-4A1 1 0 0010 6v2.798l-5.445-3.63z" />
            </svg>
          </button>
        </div>

        <!-- Volume Toggle -->
        <div v-if="volume !== -1" class="relative group shrink-0 ml-1">
          <button 
            @click="showVolumeSlider = !showVolumeSlider"
            class="p-2 rounded-full hover:bg-gray-700/50 transition-colors"
            :class="showVolumeSlider ? 'text-blue-500' : 'text-gray-400 hover:text-white'"
            title="Volume"
          >
            <svg class="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
              <path v-if="volume === 0" fill-rule="evenodd" d="M9.383 3.076A1 1 0 0110 4v12a1 1 0 01-1.617.804L4.586 13H2a1 1 0 01-1-1V8a1 1 0 011-1h2.586l3.797-3.804a1 1 0 011.617.804z" clip-rule="evenodd" />
              <path v-else-if="volume < 50" fill-rule="evenodd" d="M9.383 3.076A1 1 0 0110 4v12a1 1 0 01-1.617.804L4.586 13H2a1 1 0 01-1-1V8a1 1 0 011-1h2.586l3.797-3.804a1 1 0 011.617.804zM14.657 2.929a1 1 0 011.414 0A9.972 9.972 0 0119 10a9.972 9.972 0 01-2.929 7.071 1 1 0 01-1.414-1.414A7.971 7.971 0 0017 10c0-2.21-.894-4.208-2.343-5.657a1 1 0 010-1.414z" clip-rule="evenodd" />
              <path v-else fill-rule="evenodd" d="M9.383 3.076A1 1 0 0110 4v12a1 1 0 01-1.617.804L4.586 13H2a1 1 0 01-1-1V8a1 1 0 011-1h2.586l3.797-3.804a1 1 0 011.617.804zM14.657 2.929a1 1 0 011.414 0A9.972 9.972 0 0119 10a9.972 9.972 0 01-2.929 7.071 1 1 0 01-1.414-1.414A7.971 7.971 0 0017 10c0-2.21-.894-4.208-2.343-5.657a1 1 0 010-1.414zm-2.829 2.828a1 1 0 011.415 0A5.983 5.983 0 0115 10a5.984 5.984 0 01-1.757 4.243 1 1 0 01-1.415-1.415A3.984 3.984 0 0013 10a3.983 3.983 0 00-1.172-2.828 1 1 0 010-1.415z" clip-rule="evenodd" />
            </svg>
          </button>
          
          <div 
            v-if="showVolumeSlider"
            class="absolute bottom-full right-0 mb-4 bg-gray-800 border border-gray-700 rounded-xl p-3 shadow-2xl w-10 flex flex-col items-center h-48 animate-in fade-in slide-in-from-bottom-4 duration-200"
          >
            <div class="flex-1 w-full flex flex-col items-center py-1">
              <input 
                type="range" 
                min="0" 
                max="100" 
                :value="volume"
                @input="setVolume($event.target.value)"
                class="vertical-slider h-full cursor-pointer appearance-none bg-gray-700 rounded-full w-2"
              >
            </div>
            <span class="text-[10px] text-gray-400 mt-2 font-mono">{{ volume }}</span>
          </div>
        </div>
      </div>

      <!-- Playlist Visualization (Condensed) -->
      <div v-if="playlist.length > 0" class="mt-2.5 flex items-center justify-center gap-1 overflow-x-auto py-1 px-1 scrollbar-hide">
        <div 
          v-for="(item, index) in playlist" 
          :key="`${item.path}-${index}`"
          class="rounded-full flex-shrink-0 transition-all duration-500"
          :class="{
            'scale-125 border border-white z-10': index === playlistCurrentPos,
            'opacity-40 scale-75': Math.abs(index - playlistCurrentPos) > 10
          }"
          :style="{
            width: getDotSize(item.duration),
            height: getDotSize(item.duration),
            backgroundColor: index === playlistCurrentPos ? '#ffffff' : (index < playlistCurrentPos ? '#4b5563' : getAlbumColor(item.album, item.artist)),
          }"
          :title="`${item.artist} - ${item.title} (${item.album})`"
        ></div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { useMpdStore } from '@/stores/mpd'

const router = useRouter()
const mpdStore = useMpdStore()

const showVolumeSlider = ref(false)

// Computed properties
const currentSong = computed(() => mpdStore.currentSong)
const isPlaying = computed(() => mpdStore.isPlaying)
const isConnected = computed(() => mpdStore.isConnected)
const currentTime = computed(() => mpdStore.currentTime)
const duration = computed(() => mpdStore.duration)
const volume = computed(() => mpdStore.volume)
const playlist = computed(() => mpdStore.playlist)
const playlistCurrentPos = computed(() => mpdStore.playlistCurrentPos)

const coverUrl = computed(() => {
  if (currentSong.value?.path) {
    const path = currentSong.value.path
    const dir = path.substring(0, path.lastIndexOf('/'))
    const escapedDir = dir.split('/').map(encodeURIComponent).join('/')
    return `/api/coverart/${escapedDir}`
  }
  return null
})

// Progress Bar Logic
const progressPercentage = ref(0)
const transitionDuration = ref('0s')

const updateProgress = async () => {
  if (!duration.value) {
    progressPercentage.value = 0
    return
  }
  
  transitionDuration.value = '0s'
  progressPercentage.value = (currentTime.value / duration.value) * 100
  
  if (isPlaying.value) {
    await nextTick()
    // Trigger reflow
    // eslint-disable-next-line no-unused-vars
    const _ = document.body.offsetHeight

    transitionDuration.value = '10s'
    const targetTime = Math.min(currentTime.value + 10, duration.value)
    progressPercentage.value = (targetTime / duration.value) * 100
  }
}

// Actions
const play = () => mpdStore.play()
const pause = () => mpdStore.pause()
const next = () => mpdStore.next()
const previous = () => mpdStore.previous()
const setVolume = (v) => mpdStore.setVolume(parseInt(v))

const searchBy = (q) => {
  if (!q) return
  router.push({ name: 'search', query: { q } })
}

const formatTrack = (track) => {
  if (!track) return ''
  return track.split('/')[0].padStart(2, '0')
}

// Watchers
watch([currentTime, isPlaying, duration], () => {
  updateProgress()
})

// Lifecycle
onMounted(() => {
  mpdStore.startPolling()
  mpdStore.fetchPlaylist()
  updateProgress()
})

onUnmounted(() => {
  mpdStore.stopPolling()
})

// Utility functions
const formatTime = (seconds) => {
  if (!seconds) return '0:00'
  const minutes = Math.floor(seconds / 60)
  const remainingSeconds = Math.floor(seconds % 60)
  return `${minutes}:${remainingSeconds.toString().padStart(2, '0')}`
}

const getDotSize = (seconds) => {
  if (!seconds) return '5px'
  const size = 3 + Math.log(seconds + 1) * 1.2
  return `${size}px`
}

const getAlbumColor = (album, artist) => {
  const str = `${artist || ''} - ${album || ''}`
  if (!str.trim()) return '#3b82f6'
  let hash = 0
  for (let i = 0; i < str.length; i++) {
    hash = str.charCodeAt(i) + ((hash << 5) - hash)
  }
  const h = Math.abs(hash % 360)
  return `hsl(${h}, 60%, 55%)`
}
</script>

<style scoped>
.scrollbar-hide::-webkit-scrollbar {
  display: none;
}
.scrollbar-hide {
  -ms-overflow-style: none;
  scrollbar-width: none;
}

.vertical-slider {
  writing-mode: bt-64;
  -webkit-appearance: slider-vertical;
  width: 8px;
  height: 100%;
}

input[type=range]::-webkit-slider-thumb {
  -webkit-appearance: none;
  height: 12px;
  width: 12px;
  border-radius: 50%;
  background: white;
  cursor: pointer;
  border: 2px solid #3b82f6;
  margin-top: 0;
}

.animate-in {
  animation-duration: 0.2s;
  animation-fill-mode: both;
}

@keyframes fade-in {
  from { opacity: 0; }
  to { opacity: 1; }
}

@keyframes slide-in-from-bottom-4 {
  from { transform: translateY(1rem); }
  to { transform: translateY(0); }
}

.fade-in { animation-name: fade-in; }
.slide-in-from-bottom-4 { animation-name: slide-in-from-bottom-4; }
</style>