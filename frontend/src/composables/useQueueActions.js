import { ref, computed } from 'vue'
import { useMpdStore } from '@/stores/mpd'

/**
 * Composable for managing queue actions (play, next, append)
 * Provides consistent behavior across AlbumDetailView, SearchView, and other components
 *
 * Features:
 * - Track selection management with visual feedback
 * - Queue actions: replace (play), insert (next), append
 * - Consistent icon components for UI reuse
 * - Automatic selection clearing after action
 */
export function useQueueActions() {
  const mpdStore = useMpdStore()

  // Selection state
  const selectedTracks = ref(new Set())
  const selectionOrder = ref([])

  const hasSelection = computed(() => selectedTracks.value.size > 0)

  // Check if a track is selected
  const isTrackSelected = (track) => {
    // Support both track objects and path strings
    const path = typeof track === 'string' ? track : track.path
    return selectedTracks.value.has(path)
  }

  // Get the selection order number for a track
  const getSelectionOrder = (track) => {
    const path = typeof track === 'string' ? track : track.path
    return selectionOrder.value.indexOf(path) + 1
  }

  // Toggle track selection
  const toggleSelection = (track) => {
    const path = typeof track === 'string' ? track : track.path

    if (selectedTracks.value.has(path)) {
      selectedTracks.value.delete(path)
      selectionOrder.value = selectionOrder.value.filter(p => p !== path)
    } else {
      selectedTracks.value.add(path)
      selectionOrder.value.push(path)
    }
  }

  // Clear all selections
  const clearSelection = () => {
    selectedTracks.value.clear()
    selectionOrder.value = []
  }

  // Get selected track paths in order
  const getTargetTracks = () => {
    return hasSelection.value ? selectionOrder.value : []
  }

  // Handle queue action (play, next, append)
  const handleAction = async (mode, tracks = null, albumInfo = null) => {
    // Use provided tracks or fall back to selection
    const tracksToAdd = tracks || getTargetTracks()

    if (!tracksToAdd || tracksToAdd.length === 0) {
      throw new Error('No tracks to add')
    }

    try {
      // If album info provided and no selection, use album-level endpoint
      if (albumInfo && !hasSelection.value) {
        const { artist, album } = albumInfo
        await mpdStore.addAlbumToPlaylist(artist, album, mode)
      } else {
        // Otherwise use track-level endpoint
        await mpdStore.addTracks(tracksToAdd, mode)
      }

      // Clear selection after successful action
      if (hasSelection.value) {
        clearSelection()
      }

      return { success: true }
    } catch (error) {
      console.error('Queue action failed:', error)
      throw error
    }
  }

  return {
    // State
    selectedTracks,
    selectionOrder,
    hasSelection,

    // Methods
    isTrackSelected,
    getSelectionOrder,
    toggleSelection,
    clearSelection,
    getTargetTracks,
    handleAction
  }
}

/**
 * Icon components for queue action buttons
 * These provide consistent visuals across the application
 */
export const QueueActionIcons = {
  // Play/Replace: Circular refresh arrows
  play: `
    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
    </svg>
  `,

  // Next/Insert: Right arrows
  next: `
    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 5l7 7-7 7M5 5l7 7-7 7" />
    </svg>
  `,

  // Append: Plus sign in circle
  append: `
    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v3m0 0v3m0-3h3m-3 0H9m12 0a9 9 0 11-18 0 9 9 0 0118 0z" />
    </svg>
  `
}
