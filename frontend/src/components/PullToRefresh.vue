<template>
  <div ref="containerRef" class="relative overflow-hidden">
    <!-- Pull Indicator -->
    <div 
      class="absolute left-0 right-0 flex items-center justify-center z-50 pointer-events-none"
      :style="indicatorStyle"
    >
      <div class="flex items-center gap-2 px-4 py-2 bg-neutral-800 rounded-full shadow-lg">
        <svg 
          v-if="isRefreshing"
          class="animate-spin h-5 w-5 text-blue-500"
          xmlns="http://www.w3.org/2000/svg"
          fill="none"
          viewBox="0 0 24 24"
        >
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle>
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
        </svg>
        <svg 
          v-else
          class="h-5 w-5 text-neutral-400 transition-transform duration-200"
          :style="{ transform: `rotate(${pullProgress * 180}deg)` }"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 14l-7 7m0 0l-7-7m7 7V3" />
        </svg>
        <span class="text-sm text-neutral-300">
          {{ isRefreshing ? 'Refreshing...' : (pullProgress >= 1 ? 'Release to refresh' : 'Pull to refresh') }}
        </span>
      </div>
    </div>
    
    <!-- Content -->
    <slot />
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { usePullToRefresh } from '@/composables/usePullToRefresh'

const props = defineProps({
  onRefresh: {
    type: Function,
    required: true
  },
  pullDistance: {
    type: Number,
    default: 80
  },
  maxPullDistance: {
    type: Number,
    default: 120
  }
})

const containerRef = ref(null)

const { 
  isPulling, 
  isRefreshing, 
  pullProgress, 
  indicatorStyle,
  setup, 
  cleanup 
} = usePullToRefresh({
  onRefresh: props.onRefresh,
  pullDistance: props.pullDistance,
  maxPullDistance: props.maxPullDistance
})

onMounted(() => {
  if (containerRef.value) {
    setup(containerRef.value)
  }
})

onUnmounted(() => {
  if (containerRef.value) {
    cleanup(containerRef.value)
  }
})
</script>