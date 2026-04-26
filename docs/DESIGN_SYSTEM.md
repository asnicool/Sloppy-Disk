# Sloppy Disk Design System

## Overview

This document outlines the design system for the Sloppy Disk frontend, including design tokens, components, patterns, and best practices.

---

## Design Tokens

### Color Palette

We use Tailwind's `neutral` scale for consistency:

| Token | Value | Usage |
|-------|-------|-------|
| `--color-background` | `#111827` | Main app background |
| `--color-surface` | `#1f2937` | Cards, modals, elevated surfaces |
| `--color-surface-hover` | `#374151` | Hover states on surfaces |
| `--color-border` | `#3f3f46` | Borders, dividers |
| `--color-text` | `#f9fafb` | Primary text |
| `--color-text-muted` | `#9ca3af` | Secondary text, placeholders |
| `--color-primary` | `#3b82f6` | Primary actions, links |
| `--color-primary-hover` | `#2563eb` | Primary hover state |
| `--color-success` | `#10b981` | Success states |
| `--color-warning` | `#f59e0b` | Warning states |
| `--color-error` | `#ef4444` | Error states |

### Typography

| Element | Font | Size | Weight | Line Height |
|---------|------|------|--------|-------------|
| H1 | Inter | 30px (text-3xl) | 700 (bold) | 1.2 |
| H2 | Inter | 24px (text-2xl) | 600 (semibold) | 1.3 |
| H3 | Inter | 20px (text-xl) | 600 (semibold) | 1.4 |
| Body | Inter | 16px (text-base) | 400 (normal) | 1.6 |
| Small | Inter | 14px (text-sm) | 400 (normal) | 1.5 |
| Caption | Inter | 12px (text-xs) | 400 (normal) | 1.5 |

### Spacing

| Token | Value | Usage |
|-------|-------|-------|
| `space-1` | 4px | Tight spacing |
| `space-2` | 8px | Default element spacing |
| `space-3` | 12px | Component internal spacing |
| `space-4` | 16px | Section spacing |
| `space-6` | 24px | Large section spacing |
| `space-8` | 32px | Page-level spacing |

### Border Radius

| Token | Value | Usage |
|-------|-------|-------|
| `rounded-sm` | 2px | Small elements |
| `rounded` | 4px | Buttons, inputs |
| `rounded-lg` | 8px | Cards |
| `rounded-xl` | 12px | Modals, large cards |
| `rounded-full` | 9999px | Pills, avatars |

### Shadows

| Token | Value | Usage |
|-------|-------|-------|
| `shadow` | 0 1px 3px rgba(0,0,0,0.3) | Subtle elevation |
| `shadow-lg` | 0 10px 25px rgba(0,0,0,0.3) | Modals, dropdowns |
| `shadow-xl` | 0 20px 40px rgba(0,0,0,0.4) | High elevation |

---

## Components

### BaseButton

A standardized button component with consistent styling and behavior.

**Props:**
| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `variant` | String | `'primary'` | `primary`, `secondary`, `danger`, `ghost` |
| `size` | String | `'md'` | `sm`, `md`, `lg` |
| `type` | String | `'button'` | HTML button type |
| `disabled` | Boolean | `false` | Disabled state |
| `loading` | Boolean | `false` | Shows spinner |
| `block` | Boolean | `false` | Full width |

**Usage:**
```vue
<BaseButton variant="primary" @click="handleClick">
  Click Me
</BaseButton>

<BaseButton variant="danger" size="sm" :loading="isLoading">
  Delete
</BaseButton>
```

### BaseCard

A flexible card component for displaying content.

**Props:**
| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `title` | String | `''` | Card title |
| `subtitle` | String | `''` | Card subtitle |
| `imageUrl` | String | `''` | Image URL |
| `imageRatio` | String | `'square'` | `square`, `video`, `portrait`, `auto` |
| `variant` | String | `'default'` | `default`, `outlined`, `elevated`, `flat` |
| `size` | String | `'md'` | `sm`, `md`, `lg` |
| `hoverable` | Boolean | `true` | Enable hover effects |
| `clickable` | Boolean | `false` | Make entire card clickable |
| `loading` | Boolean | `false` | Show loading state |

**Slots:**
- `default` - Card content
- `image` - Custom image content
- `overlay` - Overlay content (e.g., play button)
- `placeholder` - Custom placeholder when no image
- `prepend` - Content before title
- `actions` - Action buttons at bottom

**Usage:**
```vue
<BaseCard
  title="Album Name"
  subtitle="Artist Name"
  image-url="/path/to/cover.jpg"
  clickable
  @click="navigateToAlbum"
>
  <template #actions>
    <BaseButton size="sm">Play</BaseButton>
  </template>
</BaseCard>
```

### BaseModal

An accessible modal component with focus trap.

**Props:**
| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `modelValue` | Boolean | `false` | Show/hide modal |
| `title` | String | `''` | Modal title |
| `closeable` | Boolean | `true` | Show close button |
| `closeOnBackdrop` | Boolean | `true` | Close on backdrop click |
| `closeOnEscape` | Boolean | `true` | Close on Escape key |
| `closeLabel` | String | `'Close modal'` | ARIA label for close button |

**Slots:**
- `default` - Modal body content
- `header` - Custom header content
- `footer` - Footer content (receives `close` function)

**Usage:**
```vue
<BaseModal v-model="showModal" title="Confirm Action">
  <p>Are you sure you want to proceed?</p>
  <template #footer="{ close }">
    <BaseButton variant="ghost" @click="close">Cancel</BaseButton>
    <BaseButton variant="primary" @click="confirm">Confirm</BaseButton>
  </template>
</BaseModal>
```

### EmptyState

A component for empty states with consistent styling.

**Props:**
| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `icon` | String | `'default'` | `search`, `music`, `playlist`, `wifi`, `error`, `default` |
| `title` | String | `required` | Main message |
| `description` | String | `''` | Secondary message |
| `variant` | String | `'default'` | `default`, `error`, `warning` |

**Slots:**
- `action` - Action buttons

**Usage:**
```vue
<EmptyState
  icon="search"
  title="No results found"
  description="Try adjusting your search terms"
>
  <template #action>
    <BaseButton @click="clearSearch">Clear Search</BaseButton>
  </template>
</EmptyState>
```

### ErrorBoundary

Catches JavaScript errors in child components.

**Props:**
| Prop | Type | Default | Description |
|------|------|---------|-------------|
| `title` | String | `'Something went wrong'` | Error title |
| `description` | String | `'An unexpected error occurred'` | Error description |
| `buttonText` | String | `'Try Again'` | Reset button text |
| `showHomeButton` | Boolean | `true` | Show "Go Home" button |
| `stopPropagation` | Boolean | `true` | Stop error from bubbling |

**Slots:**
- `error` - Custom error display (receives `error` and `reset`)

**Usage:**
```vue
<ErrorBoundary @error="logError">
  <YourComponent />
</ErrorBoundary>
```

---

## Composables

### usePullToRefresh

Implements pull-to-refresh functionality for mobile.

**Options:**
| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `onRefresh` | Function | `required` | Callback when refresh triggered |
| `pullDistance` | Number | `80` | Minimum pull distance (px) |
| `maxPullDistance` | Number | `120` | Maximum visual pull (px) |

**Returns:**
| Property | Type | Description |
|----------|------|-------------|
| `isPulling` | Ref<boolean> | Currently pulling |
| `isRefreshing` | Ref<boolean> | Currently refreshing |
| `pullProgress` | Ref<number> | Pull progress (0-1) |
| `indicatorStyle` | Computed | CSS style for indicator |
| `setup` | Function | Attach to element |
| `cleanup` | Function | Detach from element |

**Usage:**
```vue
<template>
  <PullToRefresh :on-refresh="refreshData">
    <YourContent />
  </PullToRefresh>
</template>

<script setup>
const refreshData = async () => {
  await fetchNewData()
}
</script>
```

### useFocusTrap

Manages focus within a modal or dialog.

**Options:**
| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `initialFocus` | Boolean | `true` | Focus first element on activate |
| `returnFocus` | Boolean | `true` | Return focus on deactivate |

**Returns:**
| Property | Type | Description |
|----------|------|-------------|
| `containerRef` | Ref | Template ref for container |
| `isActive` | Ref<boolean> | Trap is active |
| `activate` | Function | Activate focus trap |
| `deactivate` | Function | Deactivate focus trap |
| `focusFirst` | Function | Focus first element |
| `focusLast` | Function | Focus last element |

### useApiError

Handles API errors with retry logic.

**Options:**
| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `maxRetries` | Number | `3` | Max retry attempts |
| `retryDelay` | Number | `1000` | Base delay between retries (ms) |
| `onError` | Function | `null` | Error callback |
| `onRetry` | Function | `null` | Retry callback |

**Returns:**
| Property | Type | Description |
|----------|------|-------------|
| `error` | Ref | Current error object |
| `isError` | Ref<boolean> | Has error |
| `isRetrying` | Ref<boolean> | Currently retrying |
| `executeWithRetry` | Function | Execute with retry logic |
| `withErrorHandling` | Function | Wrap function with error handling |

### useKeyboardShortcuts

Manages global keyboard shortcuts.

**Options:**
| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `onPlayPause` | Function | `null` | Space/K key |
| `onNext` | Function | `null` | Right arrow/L key |
| `onPrevious` | Function | `null` | Left arrow/J key |
| `onVolumeUp` | Function | `null` | Up arrow |
| `onVolumeDown` | Function | `null` | Down arrow |
| `onMute` | Function | `null` | M key |
| `onSearch` | Function | `null` | / or S key |
| `onNavigate` | Function | `null` | 1-9 keys |
| `enabled` | Boolean | `true` | Enable shortcuts |

---

## Patterns

### Page Structure

Standard page layout:

```vue
<template>
  <div class="space-y-6">
    <!-- Page Header -->
    <div class="flex justify-between items-center">
      <h1 class="text-3xl font-bold text-white">Page Title</h1>
      <!-- Actions -->
    </div>
    
    <!-- Loading State -->
    <div v-if="loading" class="text-neutral-400">Loading...</div>
    
    <!-- Content -->
    <div v-else class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-6">
      <BaseCard v-for="item in items" :key="item.id" ... />
    </div>
    
    <!-- Empty State -->
    <EmptyState v-if="!loading && items.length === 0" ... />
    
    <!-- Pagination -->
    <div v-if="totalPages > 1" class="flex justify-center items-center space-x-4 mt-6">
      <BaseButton ...>Previous</BaseButton>
      <span class="text-white">Page {{ currentPage }} of {{ totalPages }}</span>
      <BaseButton ...>Next</BaseButton>
    </div>
  </div>
</template>
```

### Error Handling

```vue
<template>
  <ErrorBoundary @error="handleError">
    <PullToRefresh :on-refresh="refresh">
      <YourContent />
    </PullToRefresh>
  </ErrorBoundary>
</template>

<script setup>
import { useApiError } from '@/composables/useApiError'

const { executeWithRetry, error } = useApiError({
  maxRetries: 3,
  onError: (err) => console.error('API Error:', err)
})

const fetchData = async () => {
  const result = await executeWithRetry(
    () => api.get('/endpoint'),
    'Fetching data'
  )
  
  if (result.success) {
    data.value = result.data
  }
}
</script>
```

### Modal Usage

```vue
<template>
  <BaseButton @click="showModal = true">Open Modal</BaseButton>
  
  <BaseModal
    v-model="showModal"
    title="Modal Title"
    @close="handleClose"
  >
    <p>Modal content goes here</p>
    
    <template #footer="{ close }">
      <BaseButton variant="ghost" @click="close">Cancel</BaseButton>
      <BaseButton variant="primary" @click="save">Save</BaseButton>
    </template>
  </BaseModal>
</template>
```

---

## Accessibility

### ARIA Labels

Always provide aria-label for icon-only buttons:

```vue
<button aria-label="Close dialog">
  <XIcon />
</button>
```

### Focus Management

- All interactive elements must have visible focus states
- Modals trap focus and return focus on close
- Skip navigation link for keyboard users

### Keyboard Shortcuts

Available globally:
- `Space` / `K` - Play/Pause
- `←` / `J` - Previous track
- `→` / `L` - Next track
- `↑` - Volume up
- `↓` - Volume down
- `M` - Mute
- `/` / `S` - Focus search
- `1-9` - Navigate to views
- `Esc` - Close modals / Unfocus

### Color Contrast

All text meets WCAG AA standards:
- Normal text: 4.5:1 minimum
- Large text: 3:1 minimum

---

## Best Practices

1. **Always use base components** - Don't create one-off styles
2. **Use composables for reusable logic** - Keep components focused on presentation
3. **Handle loading and error states** - Never leave users without feedback
4. **Test keyboard navigation** - Ensure all features are accessible
5. **Use semantic HTML** - Proper heading hierarchy, landmarks
6. **Keep components small** - Split when they grow too large
7. **Document props and events** - Use JSDoc comments

---

## Migration Guide

### From Custom Store to Pinia

Old:
```js
import { useMpdStore } from '@/stores/mpd'
const store = useMpdStore()
```

New:
```js
import { useMpdStore } from '@/stores/mpdStore'
const store = useMpdStore()
```

The old import still works but shows a deprecation warning.

### From Gray to Neutral Colors

Replace all `gray-*` classes with `neutral-*`:
- `bg-gray-800` → `bg-neutral-800`
- `text-gray-400` → `text-neutral-400`
- `border-gray-700` → `border-neutral-700`

---

## Changelog

### v1.1.0
- Added Pinia store (mpdStore.js)
- Deprecated custom store (mpd.js)
- Added BaseButton, BaseCard, BaseModal components
- Added usePullToRefresh, useFocusTrap, useApiError composables
- Added keyboard shortcuts support
- Standardized on neutral color palette
- Added page transitions