<template>
  <Teleport to="body">
    <Transition
      enter-active-class="transform ease-out duration-300 transition"
      enter-from-class="translate-y-2 opacity-0 sm:translate-y-0 sm:translate-x-2"
      enter-to-class="translate-y-0 opacity-100 sm:translate-x-0"
      leave-active-class="transition ease-in duration-100"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <div v-if="show" class="fixed bottom-4 right-4 z-50 flex max-w-xs w-full">
        <div 
            class="bg-neutral-800 border border-neutral-700 rounded-lg shadow-lg p-4 flex items-center space-x-3 w-full"
            :class="{ 'border-green-500/50': type === 'success', 'border-red-500/50': type === 'error' }"
        >
          <div v-if="type === 'success'" class="flex-shrink-0 text-green-500">
            <svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
            </svg>
          </div>
          <div v-else-if="type === 'error'" class="flex-shrink-0 text-red-500">
             <svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
          </div>
          <div class="flex-1 text-sm font-medium text-white">
            {{ message }}
          </div>
          <button @click="show = false" class="text-neutral-400 hover:text-white">
            <svg class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
              <path fill-rule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clip-rule="evenodd" />
            </svg>
          </button>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { ref, watch } from 'vue'

const props = defineProps({
  modelValue: Boolean,
  message: String,
  type: {
      type: String,
      default: 'success'
  },
  duration: {
      type: Number,
      default: 3000
  }
})

const emit = defineEmits(['update:modelValue'])

const show = ref(props.modelValue)
let timer = null

watch(() => props.modelValue, (val) => {
    show.value = val
    if (val) {
        if (timer) clearTimeout(timer)
        timer = setTimeout(() => {
            show.value = false
            emit('update:modelValue', false)
        }, props.duration)
    }
})

watch(show, (val) => {
    if (!val) emit('update:modelValue', false)
})
</script>
