<template>
  <div class="h-full flex flex-col">
    <!-- Top Handle for iPhone Scrolling (visual indicator only, scrolling happens in draggable area) -->
    <div class="flex-none h-6 flex items-center justify-center">
      <div class="w-16 h-1 bg-neutral-700 rounded-full"></div>
    </div>

    <!-- Collapse/Expand Controls -->
    <div class="flex-none px-4 pb-2 flex items-center justify-between z-10">
      <button 
        @click="toggleCompact"
        class="flex items-center gap-2 px-3 py-1.5 bg-neutral-800 hover:bg-neutral-700 rounded-lg text-sm text-neutral-300 transition-colors"
      >
        <svg v-if="isCompact" class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 15l7-7 7 7" />
        </svg>
        <svg v-else class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
        </svg>
        {{ isCompact ? 'Expand All' : 'Collapse All' }}
      </button>
      
      <span class="text-xs text-neutral-500">{{ mpdStore.playlist.length }} tracks</span>
    </div>
          
    <draggable 
      v-model="groupedPlaylist" 
      item-key="id"
      group="albums"
      direction="both"
      handle=".album-handle"
      class="flex-1 flex overflow-x-auto overflow-y-hidden gap-4 px-4 pb-4 pt-8 scrollbar-thin scrollbar-thumb-neutral-700 scrollbar-track-transparent touch-pad"
      ghost-class="opacity-50"
      :class="{ 'flex-wrap content-start': isCompact, 'overflow-x-auto overflow-y-hidden': !isCompact }"
      @change="handleAlbumChange"
    >
      <template #item="{ element: group }">
        <div 
          class="flex-shrink-0 flex flex-col bg-neutral-900 rounded-xl overflow-hidden border border-neutral-800 shadow-xl group-card transition-all duration-300"
          :class="[
            isCompact ? 'w-32 h-auto' : 'w-40 h-full',
            isCompact ? 'mb-4' : ''
          ]"
        >
          <!-- Album Header (Draggable Handle) -->
          <div 
            class="album-handle relative w-full cursor-grab active:cursor-grabbing"
            :class="isCompact ? 'h-24' : 'h-40'"
          >
            <img 
              v-if="group.coverUrl" 
              :src="group.coverUrl" 
              class="w-full h-full object-cover"
              :class="{ 'grayscale opacity-50': group.isPlayed && !group.hasCurrentTrack }"
            />
            <div v-else class="w-full h-full bg-neutral-800 flex items-center justify-center">
              <svg class="w-8 h-8 text-neutral-700" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19V6l12-3v13M9 19c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zm12-3c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zM9 10l12-3" />
              </svg>
            </div>
            
            <!-- Overlay Info (only show when not compact) -->
            <div v-if="!isCompact" class="absolute bottom-0 inset-x-0 bg-gradient-to-t from-black/90 to-transparent p-4 pt-12">
              <h3 class="font-bold text-white truncate text-lg">{{ group.album || 'Unknown Album' }}</h3>
              <p class="text-sm text-neutral-300 truncate">{{ group.artist || 'Unknown Artist' }}</p>
              <div v-if="group.year" class="text-xs text-neutral-500 mt-1">{{ group.year }}</div>
            </div>
            
            <!-- Compact Header Info -->
            <div v-if="isCompact" class="absolute inset-0 flex items-center px-3 bg-gradient-to-r from-black/60 to-transparent">
              <div class="flex-1 min-w-0">
                <h3 class="font-bold text-white truncate text-sm">{{ group.album || 'Unknown Album' }}</h3>
                <p class="text-xs text-neutral-300 truncate">{{ group.artist || 'Unknown Artist' }}</p>
              </div>
            </div>
            
            <!-- Remove Button -->
            <button 
              @click.stop="removeAlbum(group)"
              class="absolute top-2 right-2 p-1.5 bg-black/50 hover:bg-red-500/80 rounded-full text-white opacity-0 group-hover:opacity-100 transition-all"
              title="Remove album"
            >
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
            
            <!-- Current Track Indicator -->
            <div 
              v-if="group.hasCurrentTrack"
              class="absolute top-2 left-2 px-2 py-0.5 bg-green-500/80 rounded text-xs font-medium text-white"
            >
              Playing
            </div>
          </div>

          <!-- Tracks List (hidden in compact mode) -->
          <div v-show="!isCompact" class="flex-1 overflow-y-auto bg-neutral-900 min-h-0 p-2">
            <QueueTrackList 
              :tracks="group.tracks" 
              :group-start-pos="group.startPos"
              :current-pos="mpdStore.playlistCurrentPos"
              @track-move="handleTrackMove"
              @track-remove="handleTrackRemove"
            />
          </div>
        </div>
      </template>
    </draggable>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import draggable from 'vuedraggable'
import { useMpdStore } from '@/stores/mpd'
import QueueTrackList from '@/components/QueueTrackList.vue'

const mpdStore = useMpdStore()
const isCompact = ref(false)

const toggleCompact = () => {
  isCompact.value = !isCompact.value
}

const groupedPlaylist = computed({
  get() {
    const playlist = mpdStore.playlist
    if (!playlist || playlist.length === 0) return []

    const groups = []
    let currentGroup = null
    const currentPos = mpdStore.playlistCurrentPos

    playlist.forEach((track, index) => {
      const key = `${track.album || ''}-${track.artist || ''}`
      const isCurrentTrack = index === currentPos
      
      if (!currentGroup || currentGroup.key !== key) {
        if (currentGroup) groups.push(currentGroup)
        
        let coverUrl = null
        const dir = track.path.substring(0, track.path.lastIndexOf('/'))
        const escapedDir = dir.split('/').map(encodeURIComponent).join('/')
        coverUrl = `/api/coverart/${escapedDir}`

        currentGroup = {
          id: `group-${index}`,
          key: key,
          album: track.album,
          artist: track.artist,
          year: track.date,
          coverUrl: coverUrl,
          startPos: index,
          tracks: [],
          isPlayed: index < currentPos,
          hasCurrentTrack: false
        }
      }
      
      if (isCurrentTrack) {
        currentGroup.hasCurrentTrack = true
      }
      
      currentGroup.tracks.push({
        ...track,
        isCurrentTrack,
        isPlayed: index < currentPos
      })
    })
    
    if (currentGroup) groups.push(currentGroup)
    return groups
  },
  set(newGroups) {
    // No-op setter, rely on @change event
  }
})

const handleAlbumChange = (event) => {
  if (event.moved) {
    const { element, newIndex, oldIndex } = event.moved
    const groups = groupedPlaylist.value
    const movedGroup = groups[oldIndex]
    
    const tempGroups = [...groups]
    const [removed] = tempGroups.splice(oldIndex, 1)
    tempGroups.splice(newIndex, 0, removed)
    
    let targetPos = 0
    for (let i = 0; i < newIndex; i++) {
      targetPos += tempGroups[i].tracks.length
    }
    
    const start = movedGroup.startPos
    mpdStore.moveAlbum(start, movedGroup.tracks.length, targetPos)
  }
}

const handleTrackMove = ({ from, to }) => {
  mpdStore.moveTrack(from, to)
}

const handleTrackRemove = (pos) => {
  mpdStore.removeFromPlaylist(pos)
}

const removeAlbum = (group) => {
  for (let i = group.tracks.length - 1; i >= 0; i--) {
    mpdStore.removeFromPlaylist(group.startPos + i)
  }
}
</script>

<style scoped>
.scrollbar-thin::-webkit-scrollbar {
  height: 8px;
}
.scrollbar-thin::-webkit-scrollbar-track {
  background: transparent;
}
.scrollbar-thin::-webkit-scrollbar-thumb {
  background-color: #404040;
  border-radius: 4px;
}

/* Enable touch scrolling */
.touch-pad {
  touch-action: pan-x pan-y;
  -webkit-overflow-scrolling: touch;
}

.group-card {
  transition: all 0.3s ease;
}
</style>
