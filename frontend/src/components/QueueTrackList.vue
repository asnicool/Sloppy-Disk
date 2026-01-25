<template>
  <draggable 
    :list="tracks" 
    item-key="pos"
    group="tracks"
    ghost-class="opacity-50"
    drag-class="cursor-grabbing"
    @change="handleChange"
    class="flex flex-col gap-1 min-h-[20px]"
  >
    <template #item="{ element }">
      <div 
        class="flex items-center gap-2 p-2 bg-neutral-800/50 hover:bg-neutral-700/80 rounded cursor-grab active:cursor-grabbing group"
        @dblclick="playTrack(element.pos)"
      >
        <span class="text-xs text-neutral-500 w-6 text-center">{{ element.track || '-' }}</span>
        <div class="flex-1 min-w-0">
          <div class="text-sm text-neutral-200 truncate">{{ element.title || 'Unknown Title' }}</div>
          <!-- Optional: Show artist if it differs from album artist? For now keep simple -->
        </div>
        <div class="text-xs text-neutral-500 font-mono">{{ formatTime(element.duration) }}</div>
      </div>
    </template>
  </draggable>
</template>

<script setup>
import { computed } from 'vue'
import draggable from 'vuedraggable'
import { useMpdStore } from '@/stores/mpd'

const props = defineProps({
  tracks: {
    type: Array,
    required: true
  },
  groupStartPos: {
    type: Number,
    required: true
  }
})

const emit = defineEmits(['track-move', 'track-remove'])
const mpdStore = useMpdStore()

const formatTime = (seconds) => {
  if (!seconds || isNaN(seconds)) return '0:00'
  const mins = Math.floor(seconds / 60)
  const secs = Math.floor(seconds % 60)
  return `${mins}:${secs.toString().padStart(2, '0')}`
}

const playTrack = (pos) => {
  mpdStore.playTrack(pos)
}

const handleChange = (event) => {
  // We only care about user interactions that result in a move
  // 'moved': sorted within same list
  // 'added': dropped from another list
  
  if (event.moved) {
    const { element, newIndex } = event.moved
    // Calculate global target position
    // Since 'tracks' is just this group, the global index is groupStartPos + newIndex
    const globalTarget = props.groupStartPos + newIndex
    emit('track-move', { from: element.pos, to: globalTarget })
  } 
  
  if (event.added) {
    const { element, newIndex } = event.added
    const globalTarget = props.groupStartPos + newIndex
    emit('track-move', { from: element.pos, to: globalTarget })
  }
}
</script>
