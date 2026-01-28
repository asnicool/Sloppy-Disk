import { ref, onMounted, onUnmounted } from 'vue'

/**
 * Composable for implementing focus trap in modals and dialogs
 * Ensures keyboard navigation stays within the modal for accessibility
 * 
 * @param {Object} options
 * @param {boolean} options.initialFocus - Whether to focus the first focusable element on mount
 * @param {boolean} options.returnFocus - Whether to return focus to the trigger element on unmount
 * @returns {Object} - Methods and state for focus trap
 */
export function useFocusTrap(options = {}) {
  const { 
    initialFocus = true, 
    returnFocus = true 
  } = options

  const containerRef = ref(null)
  const previousActiveElement = ref(null)
  const isActive = ref(false)

  // Selectors for focusable elements
  const FOCUSABLE_SELECTORS = [
    'button:not([disabled])',
    'a[href]',
    'input:not([disabled])',
    'select:not([disabled])',
    'textarea:not([disabled])',
    '[tabindex]:not([tabindex="-1"])',
    '[contenteditable]'
  ].join(', ')

  /**
   * Get all focusable elements within the container
   */
  const getFocusableElements = () => {
    if (!containerRef.value) return []
    
    return Array.from(
      containerRef.value.querySelectorAll(FOCUSABLE_SELECTORS)
    ).filter(el => {
      // Filter out hidden elements
      return el.offsetParent !== null && !el.disabled
    })
  }

  /**
   * Get the first focusable element
   */
  const getFirstFocusableElement = () => {
    const elements = getFocusableElements()
    return elements[0] || null
  }

  /**
   * Get the last focusable element
   */
  const getLastFocusableElement = () => {
    const elements = getFocusableElements()
    return elements[elements.length - 1] || null
  }

  /**
   * Handle tab key press to trap focus
   */
  const handleTabKey = (event) => {
    if (!isActive.value || !containerRef.value) return

    const focusableElements = getFocusableElements()
    if (focusableElements.length === 0) return

    const firstElement = focusableElements[0]
    const lastElement = focusableElements[focusableElements.length - 1]

    // Shift + Tab
    if (event.shiftKey) {
      if (document.activeElement === firstElement) {
        event.preventDefault()
        lastElement.focus()
      }
    } else {
      // Tab
      if (document.activeElement === lastElement) {
        event.preventDefault()
        firstElement.focus()
      }
    }
  }

  /**
   * Handle escape key to close modal
   */
  const handleEscapeKey = (event) => {
    if (!isActive.value) return
    
    if (event.key === 'Escape') {
      deactivate()
    }
  }

  /**
   * Activate the focus trap
   */
  const activate = () => {
    if (!containerRef.value) return

    // Store the currently focused element
    previousActiveElement.value = document.activeElement
    
    isActive.value = true

    // Focus the first focusable element
    if (initialFocus) {
      setTimeout(() => {
        const firstElement = getFirstFocusableElement()
        if (firstElement) {
          firstElement.focus()
        } else {
          containerRef.value?.focus()
        }
      }, 0)
    }

    // Add event listeners
    document.addEventListener('keydown', handleTabKey)
    document.addEventListener('keydown', handleEscapeKey)
  }

  /**
   * Deactivate the focus trap
   */
  const deactivate = () => {
    isActive.value = false

    // Remove event listeners
    document.removeEventListener('keydown', handleTabKey)
    document.removeEventListener('keydown', handleEscapeKey)

    // Return focus to the previous element
    if (returnFocus && previousActiveElement.value) {
      previousActiveElement.value.focus()
    }
  }

  /**
   * Focus the first focusable element
   */
  const focusFirst = () => {
    const firstElement = getFirstFocusableElement()
    if (firstElement) {
      firstElement.focus()
      return true
    }
    return false
  }

  /**
   * Focus the last focusable element
   */
  const focusLast = () => {
    const lastElement = getLastFocusableElement()
    if (lastElement) {
      lastElement.focus()
      return true
    }
    return false
  }

  // Cleanup on unmount
  onUnmounted(() => {
    deactivate()
  })

  return {
    containerRef,
    isActive,
    activate,
    deactivate,
    focusFirst,
    focusLast,
    getFocusableElements,
    getFirstFocusableElement,
    getLastFocusableElement
  }
}

export default useFocusTrap