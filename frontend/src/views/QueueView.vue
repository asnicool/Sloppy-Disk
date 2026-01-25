<template>
  <div class="min-h-screen bg-neutral-900 text-white flex flex-col overflow-hidden">
    <div class="flex items-center justify-between p-4 px-6 border-b border-neutral-800 bg-neutral-950/50 backdrop-blur-md sticky top-0 z-10">
      <div class="flex items-center gap-4">
        <router-link to="/nowplaying" class="p-2 hover:bg-neutral-800 rounded-full transition-colors text-neutral-400 hover:text-white" title="Back to Now Playing">
          <svg class="w-6 h-6" fill="currentColor" viewBox="0 0 20 20">
            <path fill-rule="evenodd" d="M12.707 5.293a1 1 0 010 1.414L9.414 10l3.293 3.293a1 1 0 01-1.414 1.414l-4-4a1 1 0 010-1.414l4-4a1 1 0 011.414 0z" clip-rule="evenodd" />
          </svg>
        </router-link>
        <h1 class="text-xl font-bold">Play Queue</h1>
        <span class="text-sm text-neutral-500 font-mono">({{ mpdStore.playlist.length }} tracks)</span>
      </div>
      
      <div class="flex items-center gap-2">
         <!-- Any queue-wide actions could go here (e.g. clear, shuffle) -->
         <button @click="mpdStore.fetchPlaylist()" class="p-2 hover:bg-neutral-800 rounded-full transition-colors text-neutral-400 hover:text-white" title="Refresh Queue">
           <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" /></svg>
         </button>
      </div>
    </div>

    <div class="flex-1 overflow-hidden relative">
      <QueueAlbumStrip />
    </div>
  </div>
</template>

<script setup>
import { onMounted } from 'vue'
import { useMpdStore } from '@/stores/mpd'
import QueueAlbumStrip from '@/views/QueueAlbumStrip.vue'

const mpdStore = useMpdStore()

onMounted(async () => {
  await mpdStore.connect()
  await mpdStore.fetchPlaylist()
})
</script>

<style scoped>
/* Ensure the queue view fills the height appropriately */
</style>
