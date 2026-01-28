<template>
  <div v-if="hasError" class="error-boundary">
    <slot name="error" :error="error" :reset="reset">
      <EmptyState
        icon="error"
        :title="title"
        :description="description"
        variant="error"
      >
        <template #action>
          <BaseButton variant="primary" @click="reset">
            {{ buttonText }}
          </BaseButton>
          <BaseButton v-if="showHomeButton" variant="ghost" @click="goHome">
            Go Home
          </BaseButton>
        </template>
      </EmptyState>
    </slot>
  </div>
  <slot v-else />
</template>

<script setup>
import { ref, onErrorCaptured } from 'vue'
import { useRouter } from 'vue-router'
import EmptyState from './EmptyState.vue'
import BaseButton from './BaseButton.vue'

const props = defineProps({
  title: {
    type: String,
    default: 'Something went wrong'
  },
  description: {
    type: String,
    default: 'An unexpected error occurred. Please try again.'
  },
  buttonText: {
    type: String,
    default: 'Try Again'
  },
  showHomeButton: {
    type: Boolean,
    default: true
  },
  // Optional callback when error is caught
  onError: {
    type: Function,
    default: null
  },
  // Whether to stop error propagation
  stopPropagation: {
    type: Boolean,
    default: true
  }
})

const emit = defineEmits(['error', 'reset'])

const router = useRouter()
const hasError = ref(false)
const error = ref(null)
const errorInfo = ref('')

onErrorCaptured((err, instance, info) => {
  // Store error information
  hasError.value = true
  error.value = err
  errorInfo.value = info
  
  // Log error for debugging
  console.error('[ErrorBoundary] Caught error:', err)
  console.error('[ErrorBoundary] Component:', instance)
  console.error('[ErrorBoundary] Info:', info)
  
  // Call optional error handler
  if (props.onError) {
    props.onError(err, instance, info)
  }
  
  // Emit error event
  emit('error', { error: err, instance, info })
  
  // Return false to stop error propagation
  return !props.stopPropagation
})

const reset = () => {
  hasError.value = false
  error.value = null
  errorInfo.value = ''
  emit('reset')
}

const goHome = () => {
  reset()
  router.push('/')
}

// Expose reset method for parent components
defineExpose({
  reset,
  hasError,
  error
})
</script>