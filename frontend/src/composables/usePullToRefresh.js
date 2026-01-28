import { ref, onMounted, onUnmounted } from 'vue'

/**
 * Composable for implementing pull-to-refresh functionality
 * Optimized for mobile native feel
 * 
 * @param {Object} options
 * @param {Function} options.onRefresh - Callback function to call when refresh is triggered
 * @param {number} options.pullDistance - Minimum pull distance to trigger refresh (default: 80)
 * @param {number} options.maxPullDistance - Maximum pull distance for visual feedback (default: 120)
 * @returns {Object} - Reactive state and handlers
 */
export function usePullToRefresh(options = {}) {
  const {
    onRefresh,
    pullDistance = 80,
    maxPullDistance = 120
  } = options

  const isPulling = ref(false)
  const isRefreshing = ref(false)
  const pullProgress = ref(0)
  const startY = ref(0)
  const currentY = ref(0)

  // Touch event handlers
  const handleTouchStart = (e) => {
    // Only enable pull-to-refresh when at top of page
    if (window.scrollY > 0) return
    
    isPulling.value = true
    startY.value = e.touches[0].clientY
    currentY.value = startY.value
  }

  const handleTouchMove = (e) => {
    if (!isPulling.value || isRefreshing.value) return
    
    // Only allow pulling down (positive delta)
    const touchY = e.touches[0].clientY
    const deltaY = touchY - startY.value
    
    // Only allow pulling down, not up
    if (deltaY < 0) {
      isPulling.value = false
      pullProgress.value = 0
      return
    }
    
    // Prevent default scrolling when pulling
    if (deltaY > 0 && window.scrollY === 0) {
      e.preventDefault()
    }
    
    currentY.value = touchY
    
    // Calculate progress with resistance curve for natural feel
    const rawProgress = Math.min(deltaY / pullDistance, 1.5)
    pullProgress.value = Math.min(rawProgress, maxPullDistance / pullDistance)
  }

  const handleTouchEnd = async () => {
    if (!isPulling.value) return
    
    const deltaY = currentY.value - startY.value
    
    // Check if pulled far enough to trigger refresh
    if (deltaY >= pullDistance && onRefresh && !isRefreshing.value) {
      isRefreshing.value = true
      
      try {
        await onRefresh()
      } finally {
        // Keep the refreshing state visible briefly for visual feedback
        setTimeout(() => {
          isRefreshing.value = false
          pullProgress.value = 0
        }, 500)
      }
    } else {
      // Reset if not pulled far enough
      pullProgress.value = 0
    }
    
    isPulling.value = false
    startY.value = 0
    currentY.value = 0
  }

  // Setup and teardown
  const setup = (element) => {
    const el = element || document.body
    
    el.addEventListener('touchstart', handleTouchStart, { passive: false })
    el.addEventListener('touchmove', handleTouchMove, { passive: false })
    el.addEventListener('touchend', handleTouchEnd, { passive: true })
    el.addEventListener('touchcancel', handleTouchEnd, { passive: true })
  }

  const cleanup = (element) => {
    const el = element || document.body
    
    el.removeEventListener('touchstart', handleTouchStart)
    el.removeEventListener('touchmove', handleTouchMove)
    el.removeEventListener('touchend', handleTouchEnd)
    el.removeEventListener('touchcancel', handleTouchEnd)
  }

  // Computed style for the pull indicator
  const indicatorStyle = computed(() => {
    const translateY = Math.min(
      (pullProgress.value * pullDistance) - pullDistance,
      maxPullDistance - pullDistance
    )
    
    return {
      transform: `translateY(${translateY}px)`,
      opacity: Math.min(pullProgress.value, 1),
      transition: isPulling.value ? 'none' : 'transform 0.3s ease-out, opacity 0.3s ease-out'
    }
  })

  return {
    isPulling,
    isRefreshing,
    pullProgress,
    indicatorStyle,
    setup,
    cleanup
  }
}

// Helper for computed (need to import)
import { computed } from 'vue'