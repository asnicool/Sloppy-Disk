import { ref } from 'vue'

/**
 * Composable for handling API errors gracefully
 * Provides consistent error handling, retry logic, and user feedback
 */
export function useApiError(options = {}) {
  const {
    maxRetries = 3,
    retryDelay = 1000,
    onError = null,
    onRetry = null
  } = options

  const error = ref(null)
  const isError = ref(false)
  const isRetrying = ref(false)
  const retryCount = ref(0)
  const lastError = ref(null)

  /**
   * Clear the current error state
   */
  const clearError = () => {
    error.value = null
    isError.value = false
    lastError.value = null
  }

  /**
   * Set error state with user-friendly message
   */
  const setError = (err, context = '') => {
    const errorMessage = getErrorMessage(err, context)
    
    error.value = {
      message: errorMessage,
      original: err,
      context,
      timestamp: Date.now()
    }
    
    isError.value = true
    lastError.value = err
    
    if (onError) {
      onError(error.value)
    }
    
    console.error(`[API Error] ${context}:`, err)
    
    return error.value
  }

  /**
   * Get user-friendly error message
   */
  const getErrorMessage = (err, context = '') => {
    // Network errors
    if (!navigator.onLine) {
      return 'No internet connection. Please check your network and try again.'
    }
    
    if (err.code === 'ECONNABORTED' || err.code === 'ETIMEDOUT') {
      return 'Request timed out. Please try again.'
    }
    
    if (err.message?.includes('Network Error')) {
      return 'Network error. Please check your connection.'
    }
    
    // HTTP status errors
    if (err.response) {
      const status = err.response.status
      const message = err.response.data?.message || err.response.data?.error
      
      switch (status) {
        case 400:
          return message || 'Invalid request. Please check your input.'
        case 401:
          return 'Session expired. Please log in again.'
        case 403:
          return 'You don\'t have permission to perform this action.'
        case 404:
          return context 
            ? `${context} not found.`
            : 'The requested resource was not found.'
        case 409:
          return message || 'Conflict detected. Please try again.'
        case 422:
          return message || 'Validation failed. Please check your input.'
        case 429:
          return 'Too many requests. Please wait a moment and try again.'
        case 500:
        case 502:
        case 503:
        case 504:
          return 'Server error. Please try again later.'
        default:
          return message || `An error occurred (${status}). Please try again.`
      }
    }
    
    // Default error message
    return err.message || 'An unexpected error occurred. Please try again.'
  }

  /**
   * Execute an API call with automatic retry logic
   */
  const executeWithRetry = async (apiCall, context = '') => {
    clearError()
    retryCount.value = 0
    
    while (retryCount.value <= maxRetries) {
      try {
        isRetrying.value = retryCount.value > 0
        const result = await apiCall()
        clearError()
        isRetrying.value = false
        return { success: true, data: result, error: null }
      } catch (err) {
        retryCount.value++
        
        // Don't retry certain errors
        if (shouldNotRetry(err)) {
          const errorResult = setError(err, context)
          isRetrying.value = false
          return { success: false, data: null, error: errorResult }
        }
        
        // If we've exhausted retries, set error
        if (retryCount.value > maxRetries) {
          const errorResult = setError(err, context)
          isRetrying.value = false
          return { success: false, data: null, error: errorResult }
        }
        
        // Call retry callback
        if (onRetry) {
          onRetry(retryCount.value, maxRetries)
        }
        
        // Wait before retrying with exponential backoff
        const delay = retryDelay * Math.pow(2, retryCount.value - 1)
        await new Promise(resolve => setTimeout(resolve, delay))
      }
    }
    
    isRetrying.value = false
    return { success: false, data: null, error: error.value }
  }

  /**
   * Determine if an error should not be retried
   */
  const shouldNotRetry = (err) => {
    if (!err.response) return false
    
    const status = err.response.status
    
    // Don't retry client errors (4xx) except timeouts
    if (status >= 400 && status < 500) {
      return status !== 408 && status !== 429
    }
    
    return false
  }

  /**
   * Wrap an async function with error handling
   */
  const withErrorHandling = (fn, context = '') => {
    return async (...args) => {
      try {
        clearError()
        const result = await fn(...args)
        return { success: true, data: result, error: null }
      } catch (err) {
        const errorResult = setError(err, context)
        return { success: false, data: null, error: errorResult }
      }
    }
  }

  return {
    // State
    error,
    isError,
    isRetrying,
    retryCount,
    lastError,
    
    // Methods
    clearError,
    setError,
    getErrorMessage,
    executeWithRetry,
    withErrorHandling
  }
}

export default useApiError