<template>
  <span v-if="shouldShowBadge" class="inline-flex items-center justify-center w-4 h-4 rounded-sm" :class="badgeClass" :title="tooltip">
    <svg v-if="status === 'complete'" class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7" />
    </svg>
    <svg v-else-if="status === 'partial'" class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M12 9v2m0 4h.01" />
    </svg>
    <svg v-else class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M6 18L18 6M6 6l12 12" />
    </svg>
  </span>
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

const badgeClass = computed(() => {
  switch (status.value) {
    case 'complete':
      return 'bg-green-900/50 text-green-300 border border-green-700/50'
    case 'partial':
      return 'bg-yellow-900/50 text-yellow-300 border border-yellow-700/50 cursor-pointer hover:bg-yellow-900/70'
    case 'missing':
      return 'bg-red-900/50 text-red-300 border border-red-700/50 cursor-pointer hover:bg-red-900/70'
    default:
      return 'bg-neutral-900/50 text-neutral-300 border border-neutral-700/50'
  }
})

const tooltip = computed(() => {
  switch (status.value) {
    case 'complete':
      return 'Metadata complete'
    case 'partial':
      return 'Metadata incomplete - click to fix'
    case 'missing':
      return 'No metadata - click to fix'
    default:
      return 'Unknown metadata status'
  }
})

const shouldShowBadge = computed(() => {
  if (props.showOnly === '') return status.value !== 'unknown'
  return status.value === props.showOnly
})
</script>