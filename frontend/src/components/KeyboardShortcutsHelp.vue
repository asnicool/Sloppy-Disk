<template>
  <BaseModal
    v-model="isOpen"
    title="Keyboard Shortcuts"
    :closeable="true"
  >
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
  </BaseModal>
</template>

<script setup>
import { ref } from 'vue'
import BaseModal from './BaseModal.vue'

const isOpen = ref(false)

const open = () => {
  isOpen.value = true
}

const close = () => {
  isOpen.value = false
}

defineExpose({
  open,
  close,
  isOpen
})
</script>