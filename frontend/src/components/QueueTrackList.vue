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
        class="flex items-center gap-2 p-2 rounded cursor-grab active:cursor-grabbing group transition-colors select-none"
        :class="getTrackClass(element)"
        v-on="getTrackHandlers(element)"
      >
        <!-- Track Number / Playing Indicator -->
        <div class="w-6 flex items-center justify-center">
          <span v-if="element.isCurrentTrack" class="text-green-400">
            <svg class="w-4 h-4 animate-pulse" fill="currentColor" viewBox="0 0 20 20">
              <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM9.555 7.168A1 1 0 008 8v4a1 1 0 001.555.832l3-2a1 1 0 000-1.664l-3-2z" clip-rule="evenodd" />
            </svg>
          </span>
          <span v-else class="text-xs" :class="element.isPlayed ? 'text-neutral-600' : 'text-neutral-500'">
            {{ element.track || '-' }}
          </span>
        </div>
        
        <!-- Track Title -->
        <div class="flex-1 min-w-0">
          <div 
            class="text-sm truncate"
            :class="element.isCurrentTrack ? 'text-green-400 font-medium' : (element.isPlayed ? 'text-neutral-500' : 'text-neutral-200')"
          >
            {{ element.title || 'Unknown Title' }}
          </div>
        </div>
        
        <!-- Unplayed Indicator Dot -->
        <div 
          v-if="!element.isPlayed && !element.isCurrentTrack" 
          class="w-1.5 h-1.5 rounded-full bg-neutral-600"
          title="Not played yet"
        ></div>
      </div>
    </template>
  </draggable>
</template>

<script setup>
import { computed } from 'vue'
import draggable from 'vuedraggable'
import { useMpdStore } from '@/stores/mpd'
import { useDoubleTapSimple } from '@/composables/useDoubleTap'

const props = defineProps({
  tracks: {
    type: Array,
    required: true
  },
  groupStartPos: {
    type: Number,
    required: true
  },
  currentPos: {
    type: Number,
    default: -1
  }
})

const emit = defineEmits(['track-move', 'track-remove'])
const mpdStore = useMpdStore()
const { handlers: doubleTapHandlers } = useDoubleTapSimple({ delay: 300 })

const getTrackClass = (element) => {
  if (element.isCurrentTrack) {
    return 'bg-green-500/10 hover:bg-green-500/20 border border-green-500/30'
  }
  if (element.isPlayed) {
    return 'bg-neutral-800/30 hover:bg-neutral-800/50'
  }
  return 'bg-neutral-800/50 hover:bg-neutral-700/80'
}

const playTrack = (pos) => {
  mpdStore.playTrack(pos)
}

const getTrackHandlers = (element) => {
  return doubleTapHandlers(() => playTrack(element.pos))
}

const handleChange = (event) => {
  if (event.moved) {
    const { element, newIndex } = event.moved
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
