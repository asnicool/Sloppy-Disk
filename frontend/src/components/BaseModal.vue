<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transition duration-200 ease-out"
      enter-from-class="opacity-0"
      enter-to-class="opacity-100"
      leave-active-class="transition duration-150 ease-in"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <div
        v-if="modelValue"
        class="fixed inset-0 z-50 flex items-center justify-center p-4"
        role="dialog"
        :aria-modal="true"
        :aria-labelledby="titleId"
        :aria-describedby="descriptionId"
        @click.self="handleBackdropClick"
      >
        <!-- Backdrop -->
        <div 
          class="absolute inset-0 bg-black/75 backdrop-blur-sm"
          aria-hidden="true"
        />
        
        <!-- Modal Content -->
        <Transition
          enter-active-class="transition duration-200 ease-out"
          enter-from-class="opacity-0 scale-95 translate-y-4"
          enter-to-class="opacity-100 scale-100 translate-y-0"
          leave-active-class="transition duration-150 ease-in"
          leave-from-class="opacity-100 scale-100 translate-y-0"
          leave-to-class="opacity-0 scale-95 translate-y-4"
        >
          <div
            v-if="modelValue"
            ref="containerRef"
            tabindex="-1"
            class="relative bg-neutral-800 rounded-xl shadow-2xl max-w-lg w-full max-h-[90vh] overflow-hidden flex flex-col"
            @keydown.esc="handleEscape"
          >
            <!-- Header -->
            <div 
              v-if="title || $slots.header" 
              class="flex items-center justify-between px-6 py-4 border-b border-neutral-700"
            >
              <slot name="header">
                <h2 :id="titleId" class="text-xl font-bold text-white">
                  {{ title }}
                </h2>
              </slot>
              
              <button
                v-if="closeable"
                type="button"
                class="p-2 text-neutral-400 hover:text-white hover:bg-neutral-700 rounded-lg transition-colors focus:outline-none focus:ring-2 focus:ring-primary-500"
                :aria-label="closeLabel"
                @click="close"
              >
                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
            
            <!-- Body -->
            <div 
              :id="descriptionId"
              class="flex-1 overflow-y-auto p-6"
            >
              <slot />
            </div>
            
            <!-- Footer -->
            <div 
              v-if="$slots.footer" 
              class="flex items-center justify-end gap-3 px-6 py-4 border-t border-neutral-700 bg-neutral-800/50"
            >
              <slot name="footer" :close="close" />
            </div>
          </div>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { watch, computed, onUnmounted } from 'vue'
import { useFocusTrap } from '@/composables/useFocusTrap'

const props = defineProps({
  modelValue: {
    type: Boolean,
    default: false
  },
  title: {
    type: String,
    default: ''
  },
  closeable: {
    type: Boolean,
    default: true
  },
  closeOnBackdrop: {
    type: Boolean,
    default: true
  },
  closeOnEscape: {
    type: Boolean,
    default: true
  },
  closeLabel: {
    type: String,
    default: 'Close modal'
  }
})

const emit = defineEmits(['update:modelValue', 'close'])

// Generate unique IDs for accessibility
const instanceId = Math.random().toString(36).substr(2, 9)
const titleId = computed(() => `modal-title-${instanceId}`)
const descriptionId = computed(() => `modal-description-${instanceId}`)

// Focus trap
const { containerRef, activate, deactivate } = useFocusTrap({
  initialFocus: true,
  returnFocus: true
})

// Watch for modal open/close
watch(() => props.modelValue, (isOpen) => {
  if (isOpen) {
    // Prevent body scroll when modal is open
    document.body.style.overflow = 'hidden'
    activate()
  } else {
    document.body.style.overflow = ''
    deactivate()
  }
})

// Cleanup on unmount
onUnmounted(() => {
  document.body.style.overflow = ''
})

const close = () => {
  emit('update:modelValue', false)
  emit('close')
}

const handleBackdropClick = () => {
  if (props.closeOnBackdrop && props.closeable) {
    close()
  }
}

const handleEscape = () => {
  if (props.closeOnEscape && props.closeable) {
    close()
  }
}

// Expose close method for slots
defineExpose({
  close
})
</script>