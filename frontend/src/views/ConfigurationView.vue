<template>
  <div class="space-y-8">
    <h1 class="text-3xl font-bold text-white">Configuration</h1>

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

    <!-- N50 HIFI Control Section -->
    <section class="bg-neutral-800 rounded-lg p-6 border border-neutral-700">
      <div class="flex items-center justify-between mb-4">
        <h2 class="text-xl font-semibold text-white">N50 HIFI Component Control</h2>
        <div class="flex items-center gap-2">
          <span class="text-sm text-neutral-400">Enabled</span>
          <label class="relative inline-flex items-center cursor-pointer">
            <input type="checkbox" v-model="config.n50Enabled" class="sr-only peer" @change="saveConfig">
            <div class="w-11 h-6 bg-neutral-600 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
          </label>
        </div>
      </div>
      
      <div v-if="config.n50Enabled">
        <!-- Status -->
        <div class="mb-4 p-4 bg-neutral-700/50 rounded-lg">
          <div class="flex items-center justify-between mb-2">
            <span class="text-neutral-400">Status:</span>
            <span :class="n50Status?.isConnected ? 'text-green-400' : 'text-red-400'">
              {{ n50Status?.isConnected ? 'Connected' : 'Disconnected' }}
            </span>
          </div>
          <div v-if="n50Status?.isConnected" class="flex items-center justify-between mb-2">
            <span class="text-neutral-400">Power:</span>
            <span class="text-white">{{ n50Status?.powerStatus || 'Unknown' }}</span>
          </div>
          <div v-if="n50Status?.isConnected" class="flex items-center justify-between">
            <span class="text-neutral-400">Current Input:</span>
            <span class="text-white">{{ n50Status?.currentInput || 'Unknown' }}</span>
          </div>
        </div>

        <!-- Power Controls -->
        <div class="flex gap-2 mb-4">
          <button 
            @click="n50PowerOn" 
            :disabled="!n50Status?.isConnected || n50Status?.powerStatus === 'Powered up'"
            class="bg-green-600 hover:bg-green-500 disabled:bg-neutral-600 text-white px-4 py-2 rounded-lg transition-colors text-sm"
          >
            Power On
          </button>
          <button 
            @click="n50PowerOff" 
            :disabled="!n50Status?.isConnected || n50Status?.powerStatus !== 'Powered up'"
            class="bg-red-600 hover:bg-red-500 disabled:bg-neutral-600 text-white px-4 py-2 rounded-lg transition-colors text-sm"
          >
            Standby
          </button>
          <button 
            @click="refreshN50Status" 
            :disabled="isLoadingN50"
            class="bg-neutral-600 hover:bg-neutral-500 disabled:bg-neutral-700 text-white px-4 py-2 rounded-lg transition-colors text-sm"
          >
            Refresh
          </button>
        </div>

        <!-- Input Selection -->
        <div v-if="n50Inputs.length > 0" class="mb-4">
          <label class="block text-neutral-400 text-sm mb-2">Input Selection</label>
          <div class="flex flex-wrap gap-2">
            <button
              v-for="input in n50Inputs"
              :key="input"
              @click="n50SetInput(input)"
              :class="[
                'px-3 py-1 rounded text-sm transition-colors',
                n50Status?.currentInput === getInputDisplayName(input) 
                  ? 'bg-blue-600 text-white' 
                  : 'bg-neutral-700 text-neutral-300 hover:bg-neutral-600'
              ]"
            >
              {{ getInputDisplayName(input) }}
            </button>
          </div>
        </div>

        <!-- Configuration -->
        <div class="grid grid-cols-1 md:grid-cols-3 gap-4 pt-4 border-t border-neutral-700">
          <div>
            <label class="block text-neutral-400 text-sm mb-1">N50 Host</label>
            <input v-model="config.n50Host" type="text" placeholder="192.168.1.70" class="w-full bg-neutral-700 text-white rounded px-3 py-2 border border-neutral-600 focus:border-blue-500 outline-none text-sm">
          </div>
          <div>
            <label class="block text-neutral-400 text-sm mb-1">N50 Port</label>
            <input v-model.number="config.n50Port" type="number" placeholder="8102" class="w-full bg-neutral-700 text-white rounded px-3 py-2 border border-neutral-600 focus:border-blue-500 outline-none text-sm">
          </div>
          <div>
            <label class="block text-neutral-400 text-sm mb-1">Default Input</label>
            <select v-model="config.n50Input" class="w-full bg-neutral-700 text-white rounded px-3 py-2 border border-neutral-600 focus:border-blue-500 outline-none text-sm">
              <option value="DigitalIn1">Digital In 1</option>
              <option value="DigitalIn2">Digital In 2</option>
              <option value="DigitalInUSB">Digital In USB</option>
              <option value="MusicServer">Music Server</option>
              <option value="InternetRadio">Internet Radio</option>
              <option value="USB">USB</option>
              <option value="BTAudio">BT Audio</option>
              <option value="AirJam">Air Jam</option>
              <option value="iPod">iPod</option>
            </select>
          </div>
        </div>
        <div class="flex gap-4 mt-4">
          <label class="flex items-center gap-2 cursor-pointer">
            <input v-model="config.n50AutoControl" type="checkbox" class="w-4 h-4 text-blue-600 bg-neutral-700 border-neutral-600 rounded focus:ring-blue-500">
            <span class="text-neutral-400 text-sm">Auto Control</span>
          </label>
          <label class="flex items-center gap-2 cursor-pointer">
            <input v-model="config.n50IgnoreOnStart" type="checkbox" class="w-4 h-4 text-blue-600 bg-neutral-700 border-neutral-600 rounded focus:ring-blue-500">
            <span class="text-neutral-400 text-sm">Ignore on Start</span>
          </label>
        </div>
      </div>
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

    <!-- Metadata Providers Section -->
    <section class="bg-neutral-800 rounded-lg p-6 border border-neutral-700">
      <h2 class="text-xl font-semibold text-white mb-4">Metadata Providers</h2>
      <p class="text-neutral-400 text-sm mb-4">Configure which metadata providers to use when searching for album information.</p>
      <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
        <div class="flex items-center justify-between bg-neutral-700/50 p-3 rounded-lg">
          <div class="flex items-center gap-3">
            <div class="w-8 h-8 bg-blue-600 rounded flex items-center justify-center text-xs font-bold">MB</div>
            <div>
              <p class="text-white font-medium">MusicBrainz</p>
              <p class="text-neutral-400 text-xs">Free, comprehensive database</p>
            </div>
          </div>
          <label class="relative inline-flex items-center cursor-pointer">
            <input type="checkbox" v-model="config.musicBrainzEnabled" class="sr-only peer">
            <div class="w-11 h-6 bg-neutral-600 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
          </label>
        </div>
        <div class="flex items-center justify-between bg-neutral-700/50 p-3 rounded-lg">
          <div class="flex items-center gap-3">
            <div class="w-8 h-8 bg-orange-600 rounded flex items-center justify-center text-xs font-bold">DC</div>
            <div>
              <p class="text-white font-medium">Discogs</p>
              <p class="text-neutral-400 text-xs">Requires API credentials</p>
            </div>
          </div>
          <label class="relative inline-flex items-center cursor-pointer">
            <input type="checkbox" v-model="config.discogsEnabled" class="sr-only peer">
            <div class="w-11 h-6 bg-neutral-600 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-orange-600"></div>
          </label>
        </div>
        <div class="flex items-center justify-between bg-neutral-700/50 p-3 rounded-lg">
          <div class="flex items-center gap-3">
            <div class="w-8 h-8 bg-green-600 rounded flex items-center justify-center text-xs font-bold">FD</div>
            <div>
              <p class="text-white font-medium">FreeDB / GNUDb</p>
              <p class="text-neutral-400 text-xs">Good for older CDs</p>
            </div>
          </div>
          <label class="relative inline-flex items-center cursor-pointer">
            <input type="checkbox" v-model="config.gnuDbEnabled" class="sr-only peer">
            <div class="w-11 h-6 bg-neutral-600 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-green-600"></div>
          </label>
        </div>
        <div class="flex items-center justify-between bg-neutral-700/50 p-3 rounded-lg">
          <div class="flex items-center gap-3">
            <div class="w-8 h-8 bg-purple-600 rounded flex items-center justify-center text-xs font-bold">AA</div>
            <div>
              <p class="text-white font-medium">AlbumArt</p>
              <p class="text-neutral-400 text-xs">Cover art focused</p>
            </div>
          </div>
          <label class="relative inline-flex items-center cursor-pointer">
            <input type="checkbox" v-model="config.albumArtEnabled" class="sr-only peer">
            <div class="w-11 h-6 bg-neutral-600 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-purple-600"></div>
          </label>
        </div>
      </div>
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
            <label class="block text-neutral-400 text-sm mb-1">Cover Art Base URL</label>
            <input v-model="config.coverArtBaseUrl" type="text" class="w-full bg-neutral-700 text-white rounded px-3 py-2 border border-neutral-600 focus:border-blue-500 outline-none">
          </div>
          <div>
            <label class="block text-neutral-400 text-sm mb-1">Random Album Count</label>
            <input v-model.number="config.randomAlbumCount" type="number" class="w-full bg-neutral-700 text-white rounded px-3 py-2 border border-neutral-600 focus:border-blue-500 outline-none">
          </div>
          <div>
            <label class="block text-neutral-400 text-sm mb-1">Discogs Key</label>
            <input v-model="config.discogsKey" type="password" class="w-full bg-neutral-700 text-white rounded px-3 py-2 border border-neutral-600 focus:border-blue-500 outline-none">
          </div>
          <div>
            <label class="block text-neutral-400 text-sm mb-1">Discogs Secret</label>
            <input v-model="config.discogsSecret" type="password" class="w-full bg-neutral-700 text-white rounded px-3 py-2 border border-neutral-600 focus:border-blue-500 outline-none">
          </div>
          <div>
            <label class="block text-neutral-400 text-sm mb-1">Rsync Target</label>
            <input v-model="config.rsyncRemoteTarget" type="text" class="w-full bg-neutral-700 text-white rounded px-3 py-2 border border-neutral-600 focus:border-blue-500 outline-none" placeholder="user@host:/path">
          </div>
          <div>
            <label class="block text-neutral-400 text-sm mb-1">Rsync Options</label>
            <input v-model="config.rsyncOptions" type="text" class="w-full bg-neutral-700 text-white rounded px-3 py-2 border border-neutral-600 focus:border-blue-500 outline-none" placeholder="--delete --progress">
          </div>
          <div class="flex items-center space-x-3">
            <input v-model="config.enableActivityRefresh" type="checkbox" id="enableActivityRefresh" class="w-4 h-4 text-blue-600 bg-neutral-700 border-neutral-600 rounded focus:ring-blue-500 focus:ring-offset-neutral-800">
            <label for="enableActivityRefresh" class="text-neutral-400 text-sm font-medium">Enable Activity-Based Refresh</label>
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
import { useMpdStore } from '@/stores/mpdStore'

const mpdStore = useMpdStore()

const status = computed(() => mpdStore.status)
const n50Status = computed(() => mpdStore.n50Status)
const n50Inputs = computed(() => mpdStore.n50Inputs)
const isLoadingN50 = computed(() => mpdStore.isLoadingN50)
const syncStatus = ref(null)
const config = ref({
  mpdHost: '',
  mpdPort: 6600,
  musicRoot: '',
  coverArtRoot: '',
  coverArtBaseUrl: '',
  discogsKey: '',
  discogsSecret: '',
  rsyncRemoteTarget: '',
  rsyncOptions: '',
  randomAlbumCount: 30,
  enableActivityRefresh: true,
  musicBrainzEnabled: true,
  discogsEnabled: true,
  gnuDbEnabled: true,
  albumArtEnabled: true,
  n50Enabled: false,
  n50Host: '',
  n50Port: 8102,
  n50Input: 'DigitalIn1',
  n50AutoControl: true,
  n50IgnoreOnStart: false
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
    config.value = { 
      ...config.value,
      ...response.data 
    }
  }
}

const saveConfig = async () => {
  const response = await mpdStore.updateConfig(config.value)
  if (response.success) {
    // Show a temporary success message instead of alert
    const btn = document.querySelector('button[type="submit"]')
    const originalText = btn.textContent
    btn.textContent = 'Saved!'
    btn.classList.add('bg-green-700')
    setTimeout(() => {
      btn.textContent = originalText
      btn.classList.remove('bg-green-700')
    }, 1500)
  }
}

const getInputDisplayName = (input) => {
  const names = {
    'DigitalIn1': 'Digital In 1',
    'DigitalIn2': 'Digital In 2',
    'DigitalInUSB': 'Digital In USB',
    'MusicServer': 'Music Server',
    'InternetRadio': 'Internet Radio',
    'USB': 'USB',
    'BTAudio': 'BT Audio',
    'AirJam': 'Air Jam',
    'iPod': 'iPod'
  }
  return names[input] || input
}

const refreshN50Status = async () => {
  await mpdStore.fetchN50Status()
}

const n50PowerOn = async () => {
  await mpdStore.n50PowerOn()
}

const n50PowerOff = async () => {
  await mpdStore.n50PowerOff()
}

const n50SetInput = async (input) => {
  await mpdStore.n50SetInput(input)
}

onMounted(() => {
  loadConfig()
  fetchSyncStatus()
  syncInterval = setInterval(fetchSyncStatus, 2000)
  // Fetch N50 status and inputs
  mpdStore.fetchN50Status()
  mpdStore.fetchN50Inputs()
})

onUnmounted(() => {
  if (syncInterval) clearInterval(syncInterval)
})
</script>
