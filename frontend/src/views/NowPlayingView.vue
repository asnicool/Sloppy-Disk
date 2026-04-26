<template>
  <div class="min-h-screen bg-neutral-900 text-white flex flex-col lg:flex-row lg:items-center overflow-hidden">
    <div class="flex-1 flex items-center justify-center p-4 lg:p-8 relative">
      <div class="aspect-square w-full max-w-[min(80vh,600px)] lg:max-w-[min(60vh,700px)] relative">
        <img v-if="coverUrl" :src="coverUrl" :alt="mpdStore.currentSong?.album" class="w-full h-full object-cover shadow-2xl" />
        <div v-else class="w-full h-full bg-neutral-800 flex items-center justify-center">
          <svg class="w-24 h-24 text-neutral-600" fill="currentColor" viewBox="0 0 20 20">
            <path d="M18 3a1 1 0 00-1.196-.98l-10 2A1 1 0 006 5v9.114A4.369 4.369 0 005 14c-1.657 0-3 .895-3 2s1.343 2 3 2 3-.895 3-2V7.82l8-1.6v5.894A4.369 4.369 0 0015 12c-1.657 0-3 .895-3 2s1.343 2 3 2 3-.895 3-2V3z" />
          </svg>
        </div>
      </div>
    </div>

    <div class="lg:w-[480px] xl:w-[540px] flex flex-col justify-start lg:justify-center bg-neutral-800/50 lg:bg-transparent">
      <div class="p-6 lg:p-8 lg:pt-12 flex-shrink-0">
        <h1 class="text-2xl lg:text-3xl font-bold leading-tight mb-2 truncate">{{ mpdStore.currentSong?.title || 'Not Playing' }}</h1>
        <div class="flex flex-wrap items-baseline gap-x-3 gap-y-1">
          <button
            v-if="mpdStore.currentSong?.artist"
            @click="goToArtist"
            class="text-lg text-neutral-300 hover:text-white hover:underline truncate transition-colors"
          >{{ mpdStore.currentSong.artist }}</button>
          <span v-if="mpdStore.currentSong?.album" class="text-neutral-600">·</span>
          <button
            v-if="mpdStore.currentSong?.album"
            @click="goToAlbum"
            class="text-sm text-neutral-400 hover:text-white hover:underline truncate transition-colors"
          >{{ mpdStore.currentSong.album }}</button>
          <span v-if="mpdStore.currentSong?.date" class="text-neutral-600">·</span>
          <span v-if="mpdStore.currentSong?.date" class="text-sm text-neutral-500">{{ mpdStore.currentSong.date }}</span>
        </div>

        <div
          v-if="nextTrack"
          @click="next"
          class="mt-4 flex items-center gap-2 cursor-pointer group"
        >
          <svg class="w-3 h-3 text-neutral-600 group-hover:text-neutral-400 flex-shrink-0 transition-colors" fill="currentColor" viewBox="0 0 20 20">
            <path d="M4.555 5.168A1 1 0 003 6v8a1 1 0 001.555.832L10 11.202V14a1 1 0 001.555.832l6-4a1 1 0 000-1.664l-6-4A1 1 0 0010 6v2.798l-5.445-3.63z" />
          </svg>
          <span class="text-xs text-neutral-600 group-hover:text-neutral-400 truncate transition-colors">Next: {{ nextTrack.title || nextTrack.path }}</span>
          <span v-if="nextTrack.artist" class="text-xs text-neutral-700 group-hover:text-neutral-500 truncate transition-colors">— {{ nextTrack.artist }}</span>
        </div>
      </div>

      <div class="px-6 lg:px-8 flex-shrink-0">
        <div class="w-full h-1 bg-neutral-700 rounded-full cursor-pointer relative" @click="seekTo" ref="progressBar">
          <div class="absolute left-0 top-0 h-full bg-white rounded-full transition-all ease-linear"
               :style="{ width: `${progressPercentage}%`, transitionDuration: transitionDuration }" />
        </div>
        <div class="flex justify-between text-xs text-neutral-500 mt-2 font-mono">
          <span>{{ formatTime(displayTime) }}</span>
          <span>{{ formatTime(mpdStore.duration) }}</span>
        </div>
      </div>

      <div class="flex items-center justify-center gap-4 lg:gap-6 p-6 lg:p-8 flex-shrink-0">
        <button @click="previous" class="p-3 hover:bg-neutral-700 rounded-full transition-colors" :disabled="!mpdStore.isConnected">
          <svg class="w-7 h-7" fill="currentColor" viewBox="0 0 20 20"><path d="M8.445 14.832A1 1 0 0010 14v-2.798l5.445 3.63A1 1 0 0017 14V6a1 1 0 00-1.555-.832L10 8.798V6a1 1 0 00-1.555-.832l-6 4a1 1 0 000 1.664l6 4z" /></svg>
        </button>
        <button @click="mpdStore.isPlaying ? pause() : play()" class="p-5 text-white rounded-full hover:scale-105 transition-transform" :disabled="!mpdStore.isConnected">
          <svg v-if="mpdStore.isPlaying" class="w-8 h-8" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zM7 8a1 1 0 012 0v4a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v4a1 1 0 102 0V8a1 1 0 00-1-1z" clip-rule="evenodd" /></svg>
          <svg v-else class="w-8 h-8" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM9.555 7.168A1 1 0 008 8v4a1 1 0 001.555.832l3-2a1 1 0 000-1.664l-3-2z" clip-rule="evenodd" /></svg>
        </button>
        <button @click="next" class="p-3 hover:bg-neutral-700 rounded-full transition-colors" :disabled="!mpdStore.isConnected">
          <svg class="w-7 h-7" fill="currentColor" viewBox="0 0 20 20"><path d="M4.555 5.168A1 1 0 003 6v8a1 1 0 001.555.832L10 11.202V14a1 1 0 001.555.832l6-4a1 1 0 000-1.664l-6-4A1 1 0 0010 6v2.798l-5.445-3.63z" /></svg>
        </button>
      </div>

      <div class="px-6 lg:px-8 flex items-center gap-3 flex-shrink-0">
        <svg class="w-5 h-5 text-neutral-500 flex-shrink-0" fill="currentColor" viewBox="0 0 20 20"><path fill-rule="evenodd" d="M9.383 3.076A1 1 0 0110 4v12a1 1 0 01-1.617.804L4.586 13H2a1 1 0 01-1-1V8a1 1 0 011-1h2.586l3.797-3.804a1 1 0 011.617.804z" clip-rule="evenodd" /></svg>
        <input type="range" min="0" max="100" :value="mpdStore.volume" @input="setVolume($event.target.value)" class="flex-1 h-1 bg-neutral-700 rounded-lg appearance-none cursor-pointer" :disabled="!mpdStore.isConnected" />
        <span class="text-sm text-neutral-500 w-10 text-right font-mono">{{ mpdStore.volume }}</span>
      </div>

      <div class="flex-1"></div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useMpdStore } from '@/stores/mpd'

const mpdStore = useMpdStore()
const router = useRouter()
const progressBar = ref(null)

const progressPercentage = ref(0)
const transitionDuration = ref('0s')

const updateProgress = async () => {
  if (!mpdStore.duration) {
    progressPercentage.value = 0
    return
  }

  transitionDuration.value = '0s'
  progressPercentage.value = (mpdStore.currentTime / mpdStore.duration) * 100

  if (mpdStore.isPlaying) {
    // eslint-disable-next-line
    await new Promise(r => setTimeout(r, 0))

    if (document.body) document.body.offsetHeight

    transitionDuration.value = '10s'
    const targetTime = Math.min(mpdStore.currentTime + 10, mpdStore.duration)
    progressPercentage.value = (targetTime / mpdStore.duration) * 100
  }
}

watch(() => mpdStore.status, () => {
  updateProgress()
}, { deep: true })

watch(() => mpdStore.currentTime, () => updateProgress())

const coverUrl = computed(() => {
  if (mpdStore.currentSong?.path) {
    const path = mpdStore.currentSong.path
    const dir = path.substring(0, path.lastIndexOf('/'))
    const escapedDir = dir.split('/').map(encodeURIComponent).join('/')
    return `/api/coverart/${escapedDir}`
  }
  return null
})

const displayTime = computed(() => mpdStore.currentTime)

const nextTrack = computed(() => {
  const pos = mpdStore.playlistCurrentPos
  if (pos >= 0 && pos + 1 < mpdStore.playlist.length) {
    return mpdStore.playlist[pos + 1]
  }
  return null
})

const goToAlbum = () => {
  const song = mpdStore.currentSong
  if (song?.artist && song?.album) {
    router.push({ name: 'album-detail', params: { artist: song.artist, album: song.album } })
  }
}

const goToArtist = () => {
  const song = mpdStore.currentSong
  if (song?.artist) {
    router.push({ name: 'artist-detail', query: { artist: song.artist } })
  }
}

const play = () => mpdStore.play()
const pause = () => mpdStore.pause()
const next = () => mpdStore.next()
const previous = () => mpdStore.previous()
const setVolume = (newVolume) => mpdStore.setVolume(parseInt(newVolume))
const playTrack = (pos) => mpdStore.playTrack(pos)

const seekTo = (event) => {
  if (!progressBar.value || !mpdStore.duration) return
  const rect = progressBar.value.getBoundingClientRect()
  const percentage = (event.clientX - rect.left) / rect.width

  transitionDuration.value = '0s'
  progressPercentage.value = percentage * 100
}

const formatTime = (seconds) => {
  if (!seconds || isNaN(seconds)) return '0:00'
  const mins = Math.floor(seconds / 60)
  const secs = Math.floor(seconds % 60)
  return `${mins}:${secs.toString().padStart(2, '0')}`
}

onMounted(async () => {
  await mpdStore.connect()
  await mpdStore.fetchPlaylist()

  updateProgress()

  mpdStore.startPolling()
})

onUnmounted(() => {
})
</script>

<style scoped>
.slide-enter-active, .slide-leave-active { transition: transform 0.3s ease-out; }
.slide-enter-from, .slide-leave-to { transform: translateX(100%); }

.slide-up-enter-active, .slide-up-leave-active { transition: transform 0.3s ease-out; }
.slide-up-enter-from, .slide-up-leave-to { transform: translateY(100%); }

@media (min-width: 1024px) {
  .slide-enter-from, .slide-leave-to { transform: translateY(100%); }
}
</style>
