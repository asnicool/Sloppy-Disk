/**
 * @deprecated This file is kept for backward compatibility.
 * Please use `import { useMpdStore } from '@/stores/mpdStore'` instead.
 * This will be removed in a future version.
 */

import { useMpdStore as useMpdStorePinia } from './mpdStore'

// Re-export the Pinia store with the same interface
export function useMpdStore() {
  console.warn('[DEPRECATED] useMpdStore from @/stores/mpd is deprecated. Use @/stores/mpdStore instead.')
  return useMpdStorePinia()
}

export default useMpdStore
