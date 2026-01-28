import { onMounted, onUnmounted } from 'vue'

/**
 * Composable for handling global keyboard shortcuts
 * Provides media player controls and navigation shortcuts
 * 
 * @param {Object} options - Shortcut handlers
 */
export function useKeyboardShortcuts(options = {}) {
  const {
    onPlayPause,
    onNext,
    onPrevious,
    onVolumeUp,
    onVolumeDown,
    onMute,
    onSearch,
    onNavigate,
    enabled = true
  } = options

  // Track if user is typing in an input field
  const isTyping = () => {
    const activeElement = document.activeElement
    const tagName = activeElement?.tagName?.toLowerCase()
    const isEditable = activeElement?.isContentEditable
    
    return isEditable || 
           tagName === 'input' || 
           tagName === 'textarea' || 
           tagName === 'select'
  }

  const handleKeyDown = (event) => {
    if (!enabled) return
    
    // Don't trigger shortcuts when typing in form elements
    if (isTyping()) {
      // Allow Escape key even when typing
      if (event.key !== 'Escape') {
        return
      }
    }

    const key = event.key.toLowerCase()
    const ctrl = event.ctrlKey
    const shift = event.shiftKey
    const alt = event.altKey
    const meta = event.metaKey

    // Media Controls
    switch (key) {
      case ' ':
      case 'k':
        // Space or K: Play/Pause
        if (!isTyping() && onPlayPause) {
          event.preventDefault()
          onPlayPause()
        }
        break
        
      case 'arrowright':
      case 'l':
        // Right arrow or L: Next track
        if (onNext && !isTyping()) {
          event.preventDefault()
          onNext()
        }
        break
        
      case 'arrowleft':
      case 'j':
        // Left arrow or J: Previous track
        if (onPrevious && !isTyping()) {
          event.preventDefault()
          onPrevious()
        }
        break
        
      case 'arrowup':
        // Up arrow: Volume up
        if (onVolumeUp && !isTyping()) {
          event.preventDefault()
          onVolumeUp()
        }
        break
        
      case 'arrowdown':
        // Down arrow: Volume down
        if (onVolumeDown && !isTyping()) {
          event.preventDefault()
          onVolumeDown()
        }
        break
        
      case 'm':
        // M: Mute toggle
        if (!isTyping() && onMute) {
          event.preventDefault()
          onMute()
        }
        break
        
      case 'f':
        // F: Fullscreen (if implemented)
        if (!isTyping() && !ctrl && !alt && !meta) {
          // Could trigger fullscreen mode
        }
        break
        
      case '/':
      case 's':
        // / or S: Focus search
        if (onSearch && !ctrl && !alt && !meta && !shift) {
          event.preventDefault()
          onSearch()
        }
        break
        
      case '1':
      case '2':
      case '3':
      case '4':
      case '5':
      case '6':
      case '7':
      case '8':
      case '9':
        // Number keys: Navigate to different views
        if (onNavigate && !ctrl && !alt && !meta && !isTyping()) {
          const index = parseInt(key) - 1
          event.preventDefault()
          onNavigate(index)
        }
        break
        
      case 'escape':
        // Escape: Close modals, unfocus inputs
        if (isTyping()) {
          document.activeElement?.blur()
        }
        break
    }
  }

  // Media session API support for system media keys
  const setupMediaSession = () => {
    if ('mediaSession' in navigator) {
      navigator.mediaSession.setActionHandler('play', () => {
        if (onPlayPause) onPlayPause()
      })
      
      navigator.mediaSession.setActionHandler('pause', () => {
        if (onPlayPause) onPlayPause()
      })
      
      navigator.mediaSession.setActionHandler('previoustrack', () => {
        if (onPrevious) onPrevious()
      })
      
      navigator.mediaSession.setActionHandler('nexttrack', () => {
        if (onNext) onNext()
      })
    }
  }

  onMounted(() => {
    if (enabled) {
      document.addEventListener('keydown', handleKeyDown)
      setupMediaSession()
    }
  })

  onUnmounted(() => {
    document.removeEventListener('keydown', handleKeyDown)
  })

  return {
    isTyping
  }
}

export default useKeyboardShortcuts