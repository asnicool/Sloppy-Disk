<template>
  <div id="app" class="min-h-screen bg-neutral-900 text-white flex flex-col">
    <!-- Navigation Header -->
    <nav class="bg-neutral-800 border-b border-neutral-700 sticky top-0 z-40">
      <div class="max-w-7xl mx-auto px-4">
        <div class="flex items-center justify-between h-14">
          <div class="flex items-center space-x-2 md:space-x-6">
            <router-link to="/albums" class="p-2 text-neutral-400 hover:text-white transition-colors" title="Albums">
              <svg class="w-6 h-6" fill="currentColor" viewBox="0 0 20 20">
                <path d="M18 3a1 1 0 00-1.196-.98l-10 2A1 1 0 006 5v9.114A4.369 4.369 0 005 14c-1.657 0-3 .895-3 2s1.343 2 3 2 3-.895 3-2V7.82l8-1.6v5.894A4.369 4.369 0 0015 12c-1.657 0-3 .895-3 2s1.343 2 3 2 3-.895 3-2V3z" />
              </svg>
            </router-link>
            <router-link to="/artists" class="p-2 text-neutral-400 hover:text-white transition-colors" title="Artists">
              <svg class="w-6 h-6" fill="currentColor" viewBox="0 0 20 20">
                <path fill-rule="evenodd" d="M10 9a3 3 0 100-6 3 3 0 000 6zm-7 9a7 7 0 1114 0H3z" clip-rule="evenodd" />
              </svg>
            </router-link>
            <router-link to="/dates" class="p-2 text-neutral-400 hover:text-white transition-colors" title="Dates">
              <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" />
              </svg>
            </router-link>
            <router-link to="/genres" class="p-2 text-neutral-400 hover:text-white transition-colors" title="Genres">
              <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19V6l12-3v13M9 19c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zm12-3c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zM9 10l12-3" />
              </svg>
            </router-link>
            <router-link to="/library" class="p-2 text-neutral-400 hover:text-white transition-colors" title="Library">
              <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4" />
              </svg>
            </router-link>
            <router-link to="/search" class="p-2 text-neutral-400 hover:text-white transition-colors" title="Search">
              <svg class="w-6 h-6" fill="currentColor" viewBox="0 0 20 20">
                <path fill-rule="evenodd" d="M8 4a4 4 0 100 8 4 4 0 000-8zM2 8a6 6 0 1110.89 3.476l4.817 4.817a1 1 0 01-1.414 1.414l-4.816-4.816A6 6 0 012 8z" clip-rule="evenodd" />
              </svg>
            </router-link>
            <router-link to="/queue" class="p-2 text-neutral-400 hover:text-white transition-colors" title="Playlist">
              <svg class="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16" />
              </svg>
            </router-link>
          </div>
          
          <div class="flex items-center">
            <!-- Connection Status -->
            <div class="flex items-center space-x-2" :title="isConnected ? 'Connected' : 'Disconnected'">
              <div
                :class="[
                  'w-2 h-2 rounded-full',
                  isConnected ? 'bg-green-500 shadow-[0_0_8px_rgba(34,197,94,0.6)]' : 'bg-red-500 shadow-[0_0_8px_rgba(239,68,68,0.6)]'
                ]"
              ></div>
            </div>
          </div>
        </div>
      </div>
    </nav>

    <!-- Main Content -->
    <main class="flex-1 max-w-7xl mx-auto w-full px-4 sm:px-6 lg:px-8 py-6 pb-32">
      <router-view v-slot="{ Component }">
        <transition name="page" mode="out-in">
          <component :is="Component" />
        </transition>
      </router-view>
    </main>

    <!-- Player Controls (Fixed Bottom) -->
    <PlayerControls v-if="showPlayerControls" class="fixed bottom-0 left-0 right-0 z-30" />
    
    
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMpdStore } from '@/stores/mpd'
import PlayerControls from '@/components/PlayerControls.vue'
import { useKeyboardShortcuts } from '@/composables/useKeyboardShortcuts'

const mpdStore = useMpdStore()
const route = useRoute()
const router = useRouter()
const isConnected = computed(() => mpdStore.isConnected)
const currentSong = computed(() => mpdStore.status?.currentSong)
const showPlayerControls = computed(() => {
  return currentSong.value && 
         route.name !== 'nowplaying' && 
         route.name !== 'queue'
})

// Keyboard shortcuts
useKeyboardShortcuts({
  onPlayPause: () => {
    if (mpdStore.isPlaying) {
      mpdStore.pause()
    } else {
      mpdStore.play()
    }
  },
  onNext: () => mpdStore.next(),
  onPrevious: () => mpdStore.previous(),
  onVolumeUp: () => {
    const newVolume = Math.min((mpdStore.volume || 0) + 5, 100)
    mpdStore.setVolume(newVolume)
  },
  onVolumeDown: () => {
    const newVolume = Math.max((mpdStore.volume || 0) - 5, 0)
    mpdStore.setVolume(newVolume)
  },
  onMute: () => {
    const currentVolume = mpdStore.volume || 0
    if (currentVolume > 0) {
      mpdStore.previousVolume = currentVolume
      mpdStore.setVolume(0)
    } else {
      mpdStore.setVolume(mpdStore.previousVolume || 50)
    }
  },
  onSearch: () => {
    router.push('/search')
    // Focus search input after navigation
    setTimeout(() => {
      const searchInput = document.querySelector('input[type="text"]')
      searchInput?.focus()
    }, 100)
  },
  onNavigate: (index) => {
    const routes = ['albums', 'artists', 'dates', 'genres', 'library', 'search', 'queue']
    if (routes[index]) {
      router.push(`/${routes[index]}`)
    }
  }
})

onMounted(() => {
  // Initialize MPD connection
  mpdStore.connect()
})
</script>

<style>
/* Global styles */
#app {
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
}

/* Custom scrollbar */
::-webkit-scrollbar {
  width: 6px;
}

::-webkit-scrollbar-track {
  background: #1f2937;
}

::-webkit-scrollbar-thumb {
  background: #4b5563;
  border-radius: 3px;
}

::-webkit-scrollbar-thumb:hover {
  background: #6b7280;
}

/* Touch-friendly sizing */
button, .clickable {
  min-height: 44px;
  min-width: 44px;
}

/* Loading animations */
@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

.animate-pulse {
  animation: pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite;
}

/* Page Transitions */
.page-enter-active,
.page-leave-active {
  transition: all 0.2s ease-out;
}

.page-enter-from {
  opacity: 0;
  transform: translateY(10px);
}

.page-leave-to {
  opacity: 0;
  transform: translateY(-10px);
}
</style>