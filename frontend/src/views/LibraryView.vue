<template>
  <div class="space-y-8">
    <h1 class="text-3xl font-bold text-white">Library Management</h1>

    <!-- Quick Stats -->
    <div class="bg-neutral-800 rounded-lg p-6 border border-neutral-700">
      <h2 class="text-xl font-semibold text-white mb-4">Quick Stats</h2>
      <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div class="text-center">
          <p class="text-2xl font-bold text-blue-400">{{ status?.playlist || 0 }}</p>
          <p class="text-neutral-400 text-sm">In Playlist</p>
        </div>
        <div class="text-center">
          <p class="text-2xl font-bold text-green-400">{{ status?.volume || 0 }}%</p>
          <p class="text-neutral-400 text-sm">Volume</p>
        </div>
        <div class="text-center">
          <p class="text-2xl font-bold text-purple-400">
            {{ status?.random ? 'On' : 'Off' }}
          </p>
          <p class="text-neutral-400 text-sm">Random</p>
        </div>
        <div class="text-center">
          <p class="text-2xl font-bold text-yellow-400">
            {{ status?.repeat ? 'On' : 'Off' }}
          </p>
          <p class="text-neutral-400 text-sm">Repeat</p>
        </div>
      </div>
    </div>

    <!-- Quick Links -->
    <section class="bg-neutral-800 rounded-lg p-6 flex items-center justify-between border border-neutral-700">
      <div>
        <h2 class="text-xl font-semibold text-white">Advanced Search</h2>
        <p class="text-neutral-400 text-sm mt-1">Search categories like artists, albums, songs, and genres with real-time results.</p>
      </div>
      <router-link 
        to="/search" 
        class="bg-blue-600 hover:bg-blue-500 text-white px-6 py-2 rounded-lg transition-colors flex items-center space-x-2"
      >
        <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
        </svg>
        <span>Open Search</span>
      </router-link>
    </section>

    <!-- Rsync Sync Section -->
    <section class="bg-neutral-800 rounded-lg p-6 border border-neutral-700">
      <h2 class="text-xl font-semibold text-white mb-4">Library Sync (Rsync)</h2>
      <div class="flex items-center justify-between mb-4">
        <div>
          <p class="text-white">Status: 
            <span :class="syncStatus?.isRunning ? 'text-blue-400' : 'text-neutral-400'">
              {{ syncStatus?.isRunning ? 'Running' : 'Idle' }}
            </span>
          </p>
          <p v-if="syncStatus?.lastRun" class="text-neutral-400 text-sm">
            Last Run: {{ new Date(syncStatus.lastRun).toLocaleString() }} 
            ({{ syncStatus.lastSuccess ? 'Success' : 'Failed' }})
          </p>
        </div>
        <button 
          @click="triggerSync" 
          :disabled="syncStatus?.isRunning"
          class="bg-blue-600 hover:bg-blue-500 disabled:bg-neutral-600 text-white px-6 py-2 rounded-lg transition-colors"
        >
          {{ syncStatus?.isRunning ? 'Syncing...' : 'Sync Now' }}
        </button>
      </div>
      <div v-if="syncStatus?.isRunning" class="w-full bg-neutral-700 rounded-full h-2.5">
        <div class="bg-blue-600 h-2.5 rounded-full transition-all duration-500" :style="{ width: syncStatus.progress + '%' }"></div>
      </div>
      <p v-if="syncStatus?.lastError" class="text-red-400 text-sm mt-2">{{ syncStatus.lastError }}</p>
    </section>

    <!-- Configuration Section -->
    <section class="bg-neutral-800 rounded-lg p-6 border border-neutral-700">
      <h2 class="text-xl font-semibold text-white mb-4">Configuration</h2>
      <form @submit.prevent="saveConfig" class="space-y-4">
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label class="block text-neutral-400 text-sm mb-1">MPD Host</label>
            <input v-model="config.mpdHost" type="text" class="w-full bg-neutral-700 text-white rounded px-3 py-2 border border-neutral-600 focus:border-blue-500 outline-none">
          </div>
          <div>
            <label class="block text-neutral-400 text-sm mb-1">MPD Port</label>
            <input v-model.number="config.mpdPort" type="number" class="w-full bg-neutral-700 text-white rounded px-3 py-2 border border-neutral-600 focus:border-blue-500 outline-none">
          </div>
          <div>
            <label class="block text-neutral-400 text-sm mb-1">Music Root (HDD)</label>
            <input v-model="config.musicRoot" type="text" class="w-full bg-neutral-700 text-white rounded px-3 py-2 border border-neutral-600 focus:border-blue-500 outline-none">
          </div>
          <div>
            <label class="block text-neutral-400 text-sm mb-1">Cover Art Root (SSD)</label>
            <input v-model="config.coverArtRoot" type="text" class="w-full bg-neutral-700 text-white rounded px-3 py-2 border border-neutral-600 focus:border-blue-500 outline-none">
          </div>
          <div>
            <label class="block text-neutral-400 text-sm mb-1">Discogs Token</label>
            <input v-model="config.discogsToken" type="password" class="w-full bg-neutral-700 text-white rounded px-3 py-2 border border-neutral-600 focus:border-blue-500 outline-none">
          </div>
          <div>
            <label class="block text-neutral-400 text-sm mb-1">Rsync Target</label>
            <input v-model="config.rsyncRemoteTarget" type="text" class="w-full bg-neutral-700 text-white rounded px-3 py-2 border border-neutral-600 focus:border-blue-500 outline-none" placeholder="user@host:/path">
          </div>
          <div class="flex items-center space-x-3 pt-6">
            <input v-model="config.enableActivityRefresh" type="checkbox" id="enableActivityRefresh" class="w-4 h-4 text-blue-600 bg-neutral-700 border-neutral-600 rounded focus:ring-blue-500 focus:ring-offset-neutral-800">
            <label for="enableActivityRefresh" class="text-neutral-400 text-sm font-medium">Enable Activity-Based Refresh (Mobile Batteries)</label>
          </div>
        </div>
        <div class="flex justify-end">
          <button type="submit" class="bg-green-600 hover:bg-green-500 text-white px-6 py-2 rounded-lg transition-colors">
            Save Configuration
          </button>
        </div>
      </form>
    </section>

    <!-- Keyboard Shortcuts Section -->
    <section class="bg-neutral-800 rounded-lg p-6 border border-neutral-700">
      <h2 class="text-xl font-semibold text-white mb-4">Keyboard Shortcuts</h2>
      <div class="space-y-6">
        <!-- Playback Controls -->
        <div>
          <h3 class="text-sm font-semibold text-neutral-400 uppercase tracking-wider mb-3">
            Playback
          </h3>
          <div class="space-y-2">
            <div class="flex items-center justify-between">
              <span class="text-neutral-300">Play / Pause</span>
              <kbd class="px-2 py-1 bg-neutral-700 rounded text-sm text-neutral-200 font-mono min-w-[60px] text-center">Space / K</kbd>
            </div>
            <div class="flex items-center justify-between">
              <span class="text-neutral-300">Previous track</span>
              <kbd class="px-2 py-1 bg-neutral-700 rounded text-sm text-neutral-200 font-mono min-w-[60px] text-center">← / J</kbd>
            </div>
            <div class="flex items-center justify-between">
              <span class="text-neutral-300">Next track</span>
              <kbd class="px-2 py-1 bg-neutral-700 rounded text-sm text-neutral-200 font-mono min-w-[60px] text-center">→ / L</kbd>
            </div>
            <div class="flex items-center justify-between">
              <span class="text-neutral-300">Volume up</span>
              <kbd class="px-2 py-1 bg-neutral-700 rounded text-sm text-neutral-200 font-mono min-w-[60px] text-center">↑</kbd>
            </div>
            <div class="flex items-center justify-between">
              <span class="text-neutral-300">Volume down</span>
              <kbd class="px-2 py-1 bg-neutral-700 rounded text-sm text-neutral-200 font-mono min-w-[60px] text-center">↓</kbd>
            </div>
            <div class="flex items-center justify-between">
              <span class="text-neutral-300">Mute / Unmute</span>
              <kbd class="px-2 py-1 bg-neutral-700 rounded text-sm text-neutral-200 font-mono min-w-[60px] text-center">M</kbd>
            </div>
          </div>
        </div>
        
        <!-- Navigation -->
        <div>
          <h3 class="text-sm font-semibold text-neutral-400 uppercase tracking-wider mb-3">
            Navigation
          </h3>
          <div class="space-y-2">
            <div class="flex items-center justify-between">
              <span class="text-neutral-300">Navigate to views</span>
              <kbd class="px-2 py-1 bg-neutral-700 rounded text-sm text-neutral-200 font-mono min-w-[60px] text-center">1-9</kbd>
            </div>
            <div class="flex items-center justify-between">
              <span class="text-neutral-300">Focus search</span>
              <kbd class="px-2 py-1 bg-neutral-700 rounded text-sm text-neutral-200 font-mono min-w-[60px] text-center">/ or S</kbd>
            </div>
            <div class="flex items-center justify-between">
              <span class="text-neutral-300">Close modals / Unfocus</span>
              <kbd class="px-2 py-1 bg-neutral-700 rounded text-sm text-neutral-200 font-mono min-w-[60px] text-center">Esc</kbd>
            </div>
          </div>
        </div>
        
        <!-- System Media Keys -->
        <div class="pt-4 border-t border-neutral-700">
          <p class="text-sm text-neutral-500">
            System media keys (play/pause, next, previous) are also supported.
          </p>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useMpdStore } from '@/stores/mpd'

const mpdStore = useMpdStore()

const status = computed(() => mpdStore.status)
const syncStatus = ref(null)
const config = ref({
  mpdHost: '',
  mpdPort: 6600,
  musicRoot: '',
  coverArtRoot: '',
  discogsToken: '',
  rsyncRemoteTarget: '',
  enableActivityRefresh: true
})

let syncInterval = null

const fetchSyncStatus = async () => {
  const response = await mpdStore.getSyncStatus()
  if (response.success) {
    syncStatus.value = response.data
  }
}

const triggerSync = async () => {
  await mpdStore.startSync()
  fetchSyncStatus()
}

const loadConfig = async () => {
  const response = await mpdStore.getConfig()
  if (response.success) {
    config.value = { ...response.data }
  }
}

const saveConfig = async () => {
  const response = await mpdStore.updateConfig(config.value)
  if (response.success) {
    alert('Configuration saved successfully')
  }
}

onMounted(() => {
  loadConfig()
  fetchSyncStatus()
  syncInterval = setInterval(fetchSyncStatus, 2000)
})

onUnmounted(() => {
  if (syncInterval) clearInterval(syncInterval)
})
</script>
