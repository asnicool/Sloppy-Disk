<template>
  <component
    :is="clickable ? 'button' : 'div'"
    :class="cardClasses"
    @click="$emit('click', $event)"
  >
    <!-- Image Section -->
    <div v-if="$slots.image || imageUrl" class="relative overflow-hidden" :class="imageContainerClass">
      <slot name="image">
        <img
          v-if="imageUrl"
          :src="imageUrl"
          :alt="imageAlt"
          class="w-full h-full object-cover transition-transform duration-500"
          :class="{ 'group-hover:scale-110': hoverable }"
          @error="$emit('imageError', $event)"
        />
      </slot>
      
      <!-- Overlay -->
      <div v-if="$slots.overlay" class="absolute inset-0">
        <slot name="overlay" />
      </div>
      
      <!-- Loading State -->
      <div v-if="loading" class="absolute inset-0 bg-neutral-800 animate-pulse" />
    </div>
    
    <!-- Default Placeholder when no image -->
    <div
      v-else-if="showPlaceholder"
      class="relative overflow-hidden bg-neutral-800 flex items-center justify-center"
      :class="imageContainerClass"
    >
      <slot name="placeholder">
        <svg class="w-12 h-12 text-neutral-600" fill="currentColor" viewBox="0 0 20 20">
          <path d="M18 3a1 1 0 00-1.196-.98l-10 2A1 1 0 006 5v9.114A4.369 4.369 0 005 14c-1.657 0-3 .895-3 2s1.343 2 3 2 3-.895 3-2V7.82l8-1.6v5.894A4.369 4.369 0 0015 12c-1.657 0-3 .895-3 2s1.343 2 3 2 3-.895 3-2V3z" />
        </svg>
      </slot>
    </div>
    
    <!-- Content Section -->
    <div v-if="$slots.default || title || subtitle" class="p-4 flex flex-col flex-1">
      <slot name="prepend" />
      
      <h3
        v-if="title"
        class="font-semibold truncate"
        :class="titleClass"
        :title="title"
      >
        {{ title }}
      </h3>
      
      <p
        v-if="subtitle"
        class="text-sm truncate mt-0.5"
        :class="subtitleClass"
      >
        {{ subtitle }}
      </p>
      
      <slot />
    </div>
    
    <!-- Actions Section -->
    <div v-if="$slots.actions" class="px-4 pb-4 mt-auto">
      <slot name="actions" />
    </div>
  </component>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  // Content
  title: {
    type: String,
    default: ''
  },
  subtitle: {
    type: String,
    default: ''
  },
  
  // Image
  imageUrl: {
    type: String,
    default: ''
  },
  imageAlt: {
    type: String,
    default: ''
  },
  imageRatio: {
    type: String,
    default: 'square',
    validator: (value) => ['square', 'video', 'portrait', 'auto'].includes(value)
  },
  
  // States
  loading: {
    type: Boolean,
    default: false
  },
  hoverable: {
    type: Boolean,
    default: true
  },
  clickable: {
    type: Boolean,
    default: false
  },
  showPlaceholder: {
    type: Boolean,
    default: true
  },
  
  // Variants
  variant: {
    type: String,
    default: 'default',
    validator: (value) => ['default', 'outlined', 'elevated', 'flat'].includes(value)
  },
  
  // Size
  size: {
    type: String,
    default: 'md',
    validator: (value) => ['sm', 'md', 'lg'].includes(value)
  }
})

defineEmits(['click', 'imageError'])

const cardClasses = computed(() => {
  const baseClasses = [
    'group',
    'flex',
    'flex-col',
    'h-full',
    'overflow-hidden',
    'transition-all',
    'duration-200'
  ]
  
  // Variant classes
  const variantClasses = {
    default: [
      'bg-neutral-800/40',
      'border',
      'border-neutral-700/50',
      'hover:border-primary-500/50',
      'rounded-xl'
    ],
    outlined: [
      'bg-transparent',
      'border',
      'border-neutral-700',
      'hover:border-neutral-500',
      'rounded-lg'
    ],
    elevated: [
      'bg-neutral-800',
      'shadow-lg',
      'hover:shadow-xl',
      'rounded-xl'
    ],
    flat: [
      'bg-neutral-800/20',
      'hover:bg-neutral-800/40',
      'rounded-lg'
    ]
  }
  
  // Hover effect
  const hoverClass = props.hoverable && !props.clickable ? 'hover:-translate-y-0.5' : ''
  
  // Clickable styles
  const clickableClass = props.clickable
    ? 'cursor-pointer active:scale-[0.98] focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 focus:ring-offset-neutral-900'
    : ''
  
  return [
    ...baseClasses,
    ...variantClasses[props.variant],
    hoverClass,
    clickableClass
  ].join(' ')
})

const imageContainerClass = computed(() => {
  const ratioClasses = {
    square: 'aspect-square',
    video: 'aspect-video',
    portrait: 'aspect-[3/4]',
    auto: ''
  }
  
  return ratioClasses[props.imageRatio]
})

const titleClass = computed(() => {
  const sizeClasses = {
    sm: 'text-sm',
    md: 'text-base',
    lg: 'text-lg'
  }
  
  return [
    sizeClasses[props.size],
    'text-neutral-100',
    props.hoverable && 'group-hover:text-primary-400'
  ].join(' ')
})

const subtitleClass = computed(() => {
  const sizeClasses = {
    sm: 'text-xs',
    md: 'text-sm',
    lg: 'text-base'
  }
  
  return [
    sizeClasses[props.size],
    'text-neutral-400',
    props.hoverable && 'group-hover:text-neutral-300'
  ].join(' ')
})
</script>