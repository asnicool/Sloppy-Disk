import { ref, onMounted, onUnmounted } from 'vue'

/**
 * Composable for handling double-tap/double-click events
 * Works on both desktop (dblclick) and mobile touch devices
 * 
 * @param {Object} options
 * @param {number} options.delay - Maximum time between taps in ms (default: 300)
 * @param {number} options.distance - Maximum distance between taps in pixels (default: 30)
 * @returns {Object} - Methods to bind to elements
 */
export function useDoubleTap(options = {}) {
  const { delay = 300, distance = 30 } = options
  
  let lastTapTime = 0
  let lastTapX = 0
  let letTapY = 0
  let tapTimeout = null
  let touchStarted = false

  /**
   * Handle touch start - record position and time
   */
  const onTouchStart = (event) => {
    touchStarted = true
    const touch = event.touches[0]
    const currentTime = Date.now()
    const timeDiff = currentTime - lastTapTime
    
    // Calculate distance from last tap
    const xDiff = Math.abs(touch.clientX - lastTapX)
    const yDiff = Math.abs(touch.clientY - letTapY)
    const distanceDiff = Math.sqrt(xDiff * xDiff + yDiff * yDiff)
    
    // Check if this is a double tap
    if (timeDiff < delay && distanceDiff < distance && lastTapTime > 0) {
      // Clear any pending single tap timeout
      if (tapTimeout) {
        clearTimeout(tapTimeout)
        tapTimeout = null
      }
      
      // Reset state
      lastTapTime = 0
      lastTapX = 0
      letTapY = 0
      
      // Trigger double tap
      return true
    }
    
    // Record this tap for potential next double tap
    lastTapTime = currentTime
    lastTapX = touch.clientX
    letTapY = touch.clientY
    
    return false
  }

  /**
   * Handle touch end
   */
  const onTouchEnd = (handler) => {
    return (event) => {
      if (!touchStarted) return
      touchStarted = false
      
      const currentTime = Date.now()
      const timeDiff = currentTime - lastTapTime
      
      // Check if this completes a double tap
      if (timeDiff < delay && lastTapTime > 0) {
        // Prevent default to avoid zooming or other browser behaviors
        event.preventDefault()
        handler(event)
        
        // Reset after handling
        lastTapTime = 0
        if (tapTimeout) {
          clearTimeout(tapTimeout)
          tapTimeout = null
        }
      }
    }
  }

  /**
   * Handle double click (for desktop)
   */
  const onDblClick = (handler) => {
    return (event) => {
      handler(event)
    }
  }

  /**
   * Bind double tap/click handlers to an element
   * Usage: v-bind="bindDoubleTap(handler)"
   */
  const bindDoubleTap = (handler) => {
    return {
      onDblclick: onDblClick(handler),
      onTouchstart: onTouchStart,
      onTouchend: onTouchEnd(handler)
    }
  }

  /**
   * Create a directive-like object for use with v-on
   * Usage: v-on="doubleTapHandlers(handler)"
   */
  const doubleTapHandlers = (handler) => {
    return {
      dblclick: onDblClick(handler),
      touchstart: onTouchStart,
      touchend: onTouchEnd(handler)
    }
  }

  return {
    bindDoubleTap,
    doubleTapHandlers,
    onTouchStart,
    onTouchEnd,
    onDblClick
  }
}

/**
 * Simpler version using a tap counter approach
 * More reliable for detecting double taps on mobile
 */
export function useDoubleTapSimple(options = {}) {
  const { delay = 300 } = options
  
  // Use a Map to track tap counts per element to handle multiple elements
  const tapCounts = new Map()
  const tapTimers = new Map()

  const handleTap = (handler, elementKey = 'default') => {
    return (event) => {
      // Only handle left-click on desktop, ignore right-click
      if (event.type === 'click' && event.button !== 0) {
        return
      }
      
      const currentCount = tapCounts.get(elementKey) || 0
      const newCount = currentCount + 1
      tapCounts.set(elementKey, newCount)
      
      if (newCount === 1) {
        // Start timer for first tap
        const timer = setTimeout(() => {
          // Single tap - reset
          tapCounts.set(elementKey, 0)
          tapTimers.delete(elementKey)
        }, delay)
        tapTimers.set(elementKey, timer)
      } else if (newCount === 2) {
        // Double tap detected
        const timer = tapTimers.get(elementKey)
        if (timer) {
          clearTimeout(timer)
          tapTimers.delete(elementKey)
        }
        tapCounts.set(elementKey, 0)
        
        // Prevent default to avoid zooming on mobile
        if (event.preventDefault) {
          event.preventDefault()
        }
        
        // Stop propagation to avoid triggering other handlers
        if (event.stopPropagation) {
          event.stopPropagation()
        }
        
        handler(event)
      }
    }
  }

  /**
   * Bind handlers for both click (desktop) and touch (mobile)
   */
  const bind = (handler, elementKey = 'default') => {
    const tapHandler = handleTap(handler, elementKey)
    
    return {
      onClick: tapHandler,
      onTouchEnd: (e) => {
        // For touch devices, we need to handle touchend
        // Don't prevent default to allow scrolling, but do prevent on double tap
        tapHandler(e)
      }
    }
  }

  /**
   * Get event handlers object for v-on
   */
  const handlers = (handler) => {
    const tapHandler = handleTap(handler)
    
    return {
      click: tapHandler,
      dblclick: (e) => {
        // Native dblclick for desktop - just trigger handler directly
        e.preventDefault()
        e.stopPropagation()
        handler(e)
      }
    }
  }

  const reset = () => {
    for (const timer of tapTimers.values()) {
      clearTimeout(timer)
    }
    tapTimers.clear()
    tapCounts.clear()
  }

  return {
    bind,
    handlers,
    handleTap,
    reset
  }
}

export default useDoubleTapSimple
