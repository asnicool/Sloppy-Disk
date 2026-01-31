<template>
  <div v-if="shouldShowBadge" class="inline-flex items-center gap-1">
    <!-- Good Metadata Badge -->
    <span v-if="status === 'complete'" class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-green-900/50 text-green-300 border border-green-700/50" title="Metadata complete">
      <svg class="w-3 h-3 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
      Complete
    </span>

    <!-- Partial Metadata Badge -->
    <span v-else-if="status === 'partial'" class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-yellow-900/50 text-yellow-300 border border-yellow-700/50 cursor-pointer hover:bg-yellow-900/70" title="Metadata incomplete - click to fix">
      <svg class="w-3 h-3 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
      </svg>
      Missing
    </span>

    <!-- Missing Metadata Badge -->
    <span v-else-if="status === 'missing'" class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-red-900/50 text-red-300 border border-red-700/50 cursor-pointer hover:bg-red-900/70" title="No metadata - click to fix">
      <svg class="w-3 h-3 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.172 16.172a4 4 0 015.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
      Missing
    </span>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  album: {
    type: Object,
    required: true
  },
  showOnly: {
    type: String,
    default: '', // empty string means show all statuses
    validator: (value) => ['', 'complete', 'partial', 'missing'].includes(value)
  }
})

const status = computed(() => {
  const album = props.album
  if (!album) return 'unknown'

  // Check if we have key metadata fields
  const hasArtist = album.artist && album.artist !== 'Unknown Artist'
  const hasAlbum = album.name && album.name !== 'Unknown Album'
  const hasDate = album.date && album.date !== ''
  const hasGenre = album.genre && album.genre !== ''
  const hasCover = album.coverUrl && album.coverUrl !== ''

  const fields = [hasArtist, hasAlbum, hasDate, hasGenre, hasCover]
  const filledCount = fields.filter(Boolean).length

  if (filledCount === fields.length) return 'complete'
  if (filledCount > 2) return 'partial'
  return 'missing'
})

const shouldShowBadge = computed(() => {
  if (props.showOnly === '') return status.value !== 'unknown'
  return status.value === props.showOnly
})
</script>