<template>
  <div class="flex flex-col items-center justify-center py-16 px-4 text-center">
    <div 
      class="w-20 h-20 rounded-full flex items-center justify-center mb-4"
      :class="iconBgClass"
    >
      <svg 
        v-if="icon === 'search'"
        class="w-10 h-10"
        :class="iconColorClass"
        fill="none" 
        stroke="currentColor" 
        viewBox="0 0 24 24"
      >
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
      </svg>
      <svg 
        v-else-if="icon === 'music'"
        class="w-10 h-10"
        :class="iconColorClass"
        fill="none" 
        stroke="currentColor" 
        viewBox="0 0 24 24"
      >
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19V6l12-3v13M9 19c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zm12-3c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zM9 10l12-3" />
      </svg>
      <svg 
        v-else-if="icon === 'playlist'"
        class="w-10 h-10"
        :class="iconColorClass"
        fill="none" 
        stroke="currentColor" 
        viewBox="0 0 24 24"
      >
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16" />
      </svg>
      <svg 
        v-else-if="icon === 'wifi'"
        class="w-10 h-10"
        :class="iconColorClass"
        fill="none" 
        stroke="currentColor" 
        viewBox="0 0 24 24"
      >
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8.111 16.404a5.5 5.5 0 017.778 0M12 20h.01m-7.08-7.071c3.904-3.905 10.236-3.905 14.141 0M1.394 9.393c5.857-5.857 15.355-5.857 21.213 0" />
      </svg>
      <svg 
        v-else-if="icon === 'error'"
        class="w-10 h-10"
        :class="iconColorClass"
        fill="none" 
        stroke="currentColor" 
        viewBox="0 0 24 24"
      >
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
      <svg 
        v-else
        class="w-10 h-10"
        :class="iconColorClass"
        fill="none" 
        stroke="currentColor" 
        viewBox="0 0 24 24"
      >
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
      </svg>
    </div>
    
    <h3 class="text-lg font-semibold text-neutral-100 mb-2">
      {{ title }}
    </h3>
    
    <p v-if="description" class="text-sm text-neutral-400 max-w-sm mb-6">
      {{ description }}
    </p>
    
    <div v-if="$slots.action" class="flex gap-3">
      <slot name="action" />
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  icon: {
    type: String,
    default: 'default',
    validator: (value) => ['search', 'music', 'playlist', 'wifi', 'error', 'default'].includes(value)
  },
  title: {
    type: String,
    required: true
  },
  description: {
    type: String,
    default: ''
  },
  variant: {
    type: String,
    default: 'default',
    validator: (value) => ['default', 'error', 'warning'].includes(value)
  }
})

const iconBgClass = computed(() => {
  const classes = {
    default: 'bg-neutral-800',
    error: 'bg-red-900/30',
    warning: 'bg-yellow-900/30'
  }
  return classes[props.variant]
})

const iconColorClass = computed(() => {
  const classes = {
    default: 'text-neutral-500',
    error: 'text-red-400',
    warning: 'text-yellow-400'
  }
  return classes[props.variant]
})
</script>