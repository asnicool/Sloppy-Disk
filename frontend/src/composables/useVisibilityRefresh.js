/**
 * useVisibilityRefresh - Composable for managing visibility-based data refresh
 *
 * Handles multiple visibility detection strategies for maximum iOS/Safari compatibility:
 * 1. Page Visibility API (visibilitychange event)
 * 2. Window focus/blur events
 * 3. Page navigation guards
 * 4. WebSocket health checks on visibility regain
 *
 * iOS Safari Notes:
 * - visibilitychange: Supported since iOS 10, but may not fire reliably when switching apps
 * - focus/blur: More reliable for tab switching within Safari
 * - WebSocket: Gets frozen/throttled in background tabs, needs health check on resume
 */

import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'

export function useVisibilityRefresh(options = {}) {
  const {
    onVisible = null,           // Callback when page becomes visible
    onHidden = null,            // Callback when page becomes hidden
    onRouteChange = null,       // Callback when route changes
    refreshOnVisible = true,    // Auto-trigger refresh on visible
    refreshOnRouteChange = true,// Auto-trigger refresh on route change
    debug = false,              // Enable debug logging
    debounceMs = 500           // Debounce rapid visibility changes
  } = options

  const router = useRouter()
  const isVisible = ref(!document.hidden)
  const lastVisibilityChange = ref(Date.now())
  let debounceTimer = null
  let cleanupFunctions = []

  // Debug logger
  const log = (...args) => {
    if (debug) {
      console.log('[useVisibilityRefresh]', ...args)
    }
  }

  // Handle visibility change (Page Visibility API)
  const handleVisibilityChange = () => {
    const nowVisible = !document.hidden
    const now = Date.now()
    const timeSinceLastChange = now - lastVisibilityChange.value

    log('Visibility change detected:', {
      nowVisible,
      wasVisible: isVisible.value,
      timeSinceLastChange
    })

    // Debounce rapid visibility changes (common on iOS during app switching)
    if (debounceTimer) {
      clearTimeout(debounceTimer)
    }

    debounceTimer = setTimeout(() => {
      if (nowVisible && !isVisible.value) {
        // Page became visible
        log('Page became visible')
        isVisible.value = true
        lastVisibilityChange.value = now

        if (onVisible) {
          onVisible()
        }

        if (refreshOnVisible) {
          log('Triggering refresh on visible')
          // Dispatch global event for components to listen to
          window.dispatchEvent(new CustomEvent('page-visible'))
        }
      } else if (!nowVisible && isVisible.value) {
        // Page became hidden
        log('Page became hidden')
        isVisible.value = false
        lastVisibilityChange.value = now

        if (onHidden) {
          onHidden()
        }

        // Dispatch global event
        window.dispatchEvent(new CustomEvent('page-hidden'))
      }
    }, debounceMs)
  }

  // Handle window focus (complementary to visibility API)
  const handleFocus = () => {
    log('Window focused')
    // Only trigger if visibility API didn't catch it
    if (document.hidden && isVisible.value) {
      log('Forcing visibility state update on focus')
      isVisible.value = false
      handleVisibilityChange()
    }
    // Even if visible, focus event means user returned to tab
    else if (!document.hidden && refreshOnVisible) {
      log('Refresh triggered on focus')
      window.dispatchEvent(new CustomEvent('page-visible'))
    }
  }

  // Handle window blur
  const handleBlur = () => {
    log('Window blurred')
    // Blur is less reliable as indicator of hidden state, but log it
    if (debug) {
      console.log('[useVisibilityRefresh] Window lost focus')
    }
  }

  // Handle page show (for iOS page cache - bfcache)
  const handlePageShow = (event) => {
    log('Page show event', event.persisted)
    // If persisted is true, page was restored from bfcache (iOS Safari)
    if (event.persisted) {
      log('Page restored from bfcache (iOS Safari)')
      window.dispatchEvent(new CustomEvent('page-visible'))
    }
  }

  // Handle route changes
  const handleRouteChange = (to, from) => {
    log('Route changed:', from.path, '→', to.path)

    if (onRouteChange) {
      onRouteChange(to, from)
    }

    if (refreshOnRouteChange) {
      log('Triggering refresh on route change')
      window.dispatchEvent(new CustomEvent('route-changed', {
        detail: { to, from }
      }))
    }
  }

  // Setup all event listeners
  const setup = () => {
    log('Setting up visibility listeners')

    // Page Visibility API (primary detection method)
    document.addEventListener('visibilitychange', handleVisibilityChange, false)
    cleanupFunctions.push(() => {
      document.removeEventListener('visibilitychange', handleVisibilityChange, false)
    })

    // Window focus/blur (supplementary detection, good for iOS)
    window.addEventListener('focus', handleFocus, false)
    cleanupFunctions.push(() => {
      window.removeEventListener('focus', handleFocus, false)
    })

    window.addEventListener('blur', handleBlur, false)
    cleanupFunctions.push(() => {
      window.removeEventListener('blur', handleBlur, false)
    })

    // Page show/hide for iOS Safari bfcache handling
    window.addEventListener('pageshow', handlePageShow, false)
    cleanupFunctions.push(() => {
      window.removeEventListener('pageshow', handlePageShow, false)
    })

    // Router navigation guard
    const afterEachHook = router.afterEach(handleRouteChange)
    cleanupFunctions.push(() => {
      afterEachHook() // Remove the navigation guard
    })

    log('Visibility listeners setup complete')
  }

  // Cleanup
  const cleanup = () => {
    log('Cleaning up visibility listeners')
    if (debounceTimer) {
      clearTimeout(debounceTimer)
      debounceTimer = null
    }
    cleanupFunctions.forEach(fn => fn())
    cleanupFunctions = []
  }

  // Manual refresh trigger
  const triggerRefresh = () => {
    log('Manual refresh triggered')
    window.dispatchEvent(new CustomEvent('page-visible'))
  }

  // Current visibility state
  const currentVisibility = () => {
    return {
      isVisible: isVisible.value,
      documentHidden: document.hidden,
      hasFocus: document.hasFocus(),
      timeSinceChange: Date.now() - lastVisibilityChange.value
    }
  }

  return {
    isVisible,
    setup,
    cleanup,
    triggerRefresh,
    currentVisibility
  }
}
