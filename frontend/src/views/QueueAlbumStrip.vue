<template>
  <div class="h-full flex flex-col">
    <!-- Top Handle for iPhone Scrolling (visual indicator only, scrolling happens in draggable area) -->
    <div class="flex-none h-6 flex items-center justify-center">
      <div class="w-16 h-1 bg-neutral-700 rounded-full"></div>
    </div>

    <!-- Header Controls -->
    <div class="flex-none px-4 pb-2 flex items-center justify-between z-10">
      <div class="flex items-center gap-2">
        <!-- Collapse/Expand Toggle -->
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
          {{ isCompact ? 'Expand' : 'Collapse' }}
        </button>

        <!-- Remove Played Albums Button -->
        <button 
          v-if="playedAlbumsCount > 0"
          @click="removePlayedAlbums"
          class="flex items-center gap-2 px-3 py-1.5 bg-neutral-800 hover:bg-red-600/80 rounded-lg text-sm text-neutral-300 hover:text-white transition-colors"
          :title="`Remove ${playedAlbumsCount} played album${playedAlbumsCount > 1 ? 's' : ''}`"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
          </svg>
          Clear Played
          <span class="px-1.5 py-0.5 bg-red-500/20 rounded text-xs font-medium">
            {{ playedAlbumsCount }}
          </span>
        </button>
      </div>
      
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
      @start="onDragStart"
      @end="onDragEnd"
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
            class="album-handle relative w-full cursor-grab active:cursor-grabbing select-none"
            :class="isCompact ? 'h-24' : 'h-40'"
            v-on="getAlbumCoverHandlers(group)"
          >
            <img 
              v-if="group.coverUrl" 
              :src="group.coverUrl" 
              class="w-full h-full object-cover pointer-events-none"
              :class="{ 'grayscale opacity-50': group.isPlayed && !group.hasCurrentTrack }"
            />
            <div v-else class="w-full h-full bg-neutral-800 flex items-center justify-center pointer-events-none">
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
              @drag-start="onTrackDragStart"
              @drag-end="onTrackDragEnd"
              @dragging="onTrackDragging"
            />
          </div>
        </div>
      </template>
    </draggable>

    <!-- Trashcan Drop Zone (appears during drag) -->
    <div 
      v-if="isDragging"
      class="absolute inset-x-0 bottom-0 z-50 flex items-center justify-center pb-8 pointer-events-none"
    >
      <div 
        class="pointer-events-auto p-4 rounded-full transition-all cursor-pointer"
        :class="dragOverTrashcan ? 'bg-red-500 scale-110' : 'bg-neutral-800 hover:bg-red-500/80'"
        @dragover.prevent="onTrashcanDragOver"
        @dragleave="onTrashcanDragLeave"
        @drop="onTrashcanDrop"
      >
        <svg class="w-8 h-8" :class="dragOverTrashcan ? 'text-white' : 'text-neutral-400'" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
        </svg>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import draggable from 'vuedraggable'
import { useMpdStore } from '@/stores/mpd'
import { useDoubleTapSimple } from '@/composables/useDoubleTap'
import QueueTrackList from '@/components/QueueTrackList.vue'
import { debounce } from 'lodash-es'

const STORAGE_KEY = 'queue-view-compact'

const mpdStore = useMpdStore()
const isCompact = ref(true)
const isDragging = ref(false)
const dragOverTrashcan = ref(false)
let draggedItem = null
let draggedType = null
let draggedTrackPos = null
const isMutating = ref(false)
const lastDragEnd = ref(0)
const DRAG_PLAY_COOLDOWN = 800

const { handlers: doubleTapHandlers, reset: resetDoubleTap } = useDoubleTapSimple({ delay: 300 })

onMounted(() => {
  const saved = localStorage.getItem(STORAGE_KEY)
  if (saved !== null) {
    isCompact.value = saved === 'true'
  }
})

const toggleCompact = () => {
  isCompact.value = !isCompact.value
  localStorage.setItem(STORAGE_KEY, String(isCompact.value))
}

const playAlbumFromStart = (group) => {
  const timeSinceDrag = Date.now() - lastDragEnd.value
  if (timeSinceDrag < DRAG_PLAY_COOLDOWN) {
    console.log('[QueueAlbumStrip] Blocked album play — within drag cooldown (%dms)', timeSinceDrag)
    return
  }
  mpdStore.playTrack(group.startPos)
}

const getAlbumCoverHandlers = (group) => {
  if (isDragging.value) return {}
  return doubleTapHandlers(() => playAlbumFromStart(group))
}

const calculateDotSizes = debounce((tracks) => {
  if (!tracks || tracks.length === 0) return
  
  const durations = tracks.map(t => t.duration || 0).filter(d => d > 0)
  if (durations.length === 0) return
  
  const minDuration = Math.min(...durations)
  const maxDuration = Math.max(...durations)
  const durationRange = maxDuration - minDuration
  
  tracks.forEach(track => {
    if (track.duration && durationRange > 60) {
      const relativePosition = (track.duration - minDuration) / durationRange
      if (relativePosition > 0.66) {
        track.dotSize = 'large'
      } else if (relativePosition > 0.33) {
        track.dotSize = 'medium'
      } else {
        track.dotSize = 'small'
      }
    } else {
      const minutes = (track.duration || 0) / 60
      if (minutes >= 10) {
        track.dotSize = 'large'
      } else if (minutes >= 5) {
        track.dotSize = 'medium'
      } else {
        track.dotSize = 'small'
      }
    }
  })
}, 100)

const groupedPlaylist = computed({
  get() {
    const playlist = mpdStore.playlist
    if (!playlist || playlist.length === 0) return []

    const groups = []
    let currentGroup = null
    const currentPos = mpdStore.playlistCurrentPos
    const currentSongPath = mpdStore.currentSongPath

    playlist.forEach((track, index) => {
      const key = `${track.album || ''}-${track.artist || ''}`
      // FIX: Use song identity (path) instead of position index to determine current track
      // This prevents "jumping" when playlist order changes via moves
      const isCurrentTrack = currentSongPath ? track.path === currentSongPath : index === currentPos

      if (!currentGroup || currentGroup.key !== key) {
        if (currentGroup) {
          calculateDotSizes(currentGroup.tracks)
          currentGroup.isFullyPlayed = currentGroup.tracks.every(t => t.isPlayed)
          groups.push(currentGroup)
        }

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
          hasCurrentTrack: false,
          totalDuration: 0,
          isFullyPlayed: false
        }
      }

      if (isCurrentTrack) {
        currentGroup.hasCurrentTrack = true
      }

      const trackWithDuration = {
        ...track,
        isCurrentTrack,
        isPlayed: index < currentPos,
        duration: track.duration || 0,
        dotSize: 'small'
      }

      currentGroup.tracks.push(trackWithDuration)
      currentGroup.totalDuration += track.duration || 0
    })

    if (currentGroup) {
      calculateDotSizes(currentGroup.tracks)
      currentGroup.isFullyPlayed = currentGroup.tracks.every(t => t.isPlayed)
      groups.push(currentGroup)
    }
    return groups
  },
  set(newGroups) {
  }
})

const playedAlbumsCount = computed(() => {
  return groupedPlaylist.value.filter(g => g.isFullyPlayed && !g.hasCurrentTrack).length
})

const handleAlbumChange = (event) => {
  if (event.moved) {
    if (isMutating.value) return
    isMutating.value = true

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
    mpdStore.moveAlbum(start, movedGroup.tracks.length, targetPos).finally(() => {
      isMutating.value = false
    })
  }
}

const handleTrackMove = ({ from, to }) => {
  if (isMutating.value) return
  isMutating.value = true
  mpdStore.moveTrack(from, to).finally(() => {
    isMutating.value = false
  })
}

const handleTrackRemove = (pos) => {
  if (isMutating.value) return
  isMutating.value = true
  mpdStore.removeFromPlaylist(pos).finally(() => {
    isMutating.value = false
  })
}

const removeAlbum = async (group) => {
  const positions = []
  for (let i = 0; i < group.tracks.length; i++) {
    positions.push(group.startPos + i)
  }
  await mpdStore.removeMultipleFromPlaylist(positions)
}

const removePlayedAlbums = async () => {
  // Refresh playlist first to ensure we have current positions
  await mpdStore.fetchPlaylist()
  
  const groups = groupedPlaylist.value
  const toRemove = groups
    .filter(g => g.isFullyPlayed && !g.hasCurrentTrack)
    .sort((a, b) => b.startPos - a.startPos)
  
  console.log('[QueueAlbumStrip] removePlayedAlbums: found', toRemove.length, 'albums to remove')
  
  // Collect all positions to remove (in reverse order - highest first)
  const positionsToRemove = []
  for (const group of toRemove) {
    console.log('[QueueAlbumStrip] Removing album:', group.album, 'from pos', group.startPos, 'tracks:', group.tracks.length)
    for (let i = 0; i < group.tracks.length; i++) {
      positionsToRemove.push(group.startPos + i)
    }
  }
  
  console.log('[QueueAlbumStrip] Total positions to remove:', positionsToRemove)
  
  // Use batch removal
  await mpdStore.removeMultipleFromPlaylist(positionsToRemove)
}

const onDragStart = (event) => {
  isDragging.value = true
  resetDoubleTap()
  const groups = groupedPlaylist.value
  const oldIndex = event.oldIndex
  if (oldIndex !== undefined && groups[oldIndex]) {
    draggedItem = groups[oldIndex]
    draggedType = 'album'
  }
}

const onDragEnd = () => {
  isDragging.value = false
  dragOverTrashcan.value = false
  lastDragEnd.value = Date.now()
  resetDoubleTap()
  isMutating.value = false
  draggedItem = null
  draggedType = null
  draggedTrackPos = null
}

const onTrackDragStart = () => {
  isDragging.value = true
  draggedType = 'track'
}

const onTrackDragEnd = () => {
  isDragging.value = false
  dragOverTrashcan.value = false
  isMutating.value = false
  draggedItem = null
  draggedType = null
  draggedTrackPos = null
}

const onTrackDragging = (data) => {
  if (data && data.type === 'track') {
    draggedTrackPos = data.pos
  }
}

const onTrashcanDragOver = (event) => {
  event.dataTransfer.dropEffect = 'move'
  dragOverTrashcan.value = true
}

const onTrashcanDragLeave = () => {
  dragOverTrashcan.value = false
}

const onTrashcanDrop = async () => {
  dragOverTrashcan.value = false
  isDragging.value = false

  if (draggedType === 'album' && draggedItem) {
    await mpdStore.fetchPlaylist()
    const groups = groupedPlaylist.value
    const current = groups.find(g => g.id === draggedItem.id)
    if (current) {
      await removeAlbum(current)
    }
  } else if (draggedType === 'track' && draggedTrackPos !== null) {
    await mpdStore.fetchPlaylist()
    if (draggedTrackPos < mpdStore.playlist.length) {
      mpdStore.removeFromPlaylist(draggedTrackPos)
    }
  }
  
  draggedItem = null
  draggedType = null
  draggedTrackPos = null
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

.touch-pad {
  touch-action: pan-x pan-y;
  -webkit-overflow-scrolling: touch;
}

.group-card {
  transition: all 0.3s ease;
}
</style>