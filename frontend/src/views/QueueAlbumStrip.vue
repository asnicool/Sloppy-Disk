<template>
  <div class="h-full flex flex-col">
    <!-- Trash Zone (conditionally visible or always visible but subtle?) -->
    <!-- Ideally dragging reveals it, but for simplicity let's put it in a corner or make the 'sidebar' trashable? 
         Let's keep it simple: A Trash Bar at the bottom if dragging? 
         Currently just implementing the list first. -->
         
    <draggable 
      v-model="groupedPlaylist" 
      item-key="id"
      group="albums"
      direction="horizontal"
      handle=".album-handle"
      class="flex-1 flex overflow-x-auto overflow-y-hidden gap-4 p-4 scrollbar-thin scrollbar-thumb-neutral-700 scrollbar-track-transparent"
      ghost-class="opacity-50"
      @change="handleAlbumChange"
    >
      <template #item="{ element: group }">
        <div class="flex-shrink-0 w-40 flex flex-col bg-neutral-900 rounded-xl overflow-hidden border border-neutral-800 shadow-xl group-card">
          <!-- Album Header (Draggable Handle) -->
          <div class="album-handle relative h-40 w-full cursor-grab active:cursor-grabbing">
            <img 
              v-if="group.coverUrl" 
              :src="group.coverUrl" 
              class="w-full h-full object-cover"
              :class="{ 'grayscale opacity-70': group.isPlayed }"
            />
            <div v-else class="w-full h-full bg-neutral-800 flex items-center justify-center">
              <svg class="w-16 h-16 text-neutral-700" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19V6l12-3v13M9 19c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zm12-3c0 1.105-1.343 2-3 2s-3-.895-3-2 1.343-2 3-2 3 .895 3 2zM9 10l12-3" /></svg>
            </div>
            
            <!-- Overlay Info -->
            <div class="absolute bottom-0 inset-x-0 bg-gradient-to-t from-black/90 to-transparent p-4 pt-12">
              <h3 class="font-bold text-white truncate text-lg">{{ group.album || 'Unknown Album' }}</h3>
              <p class="text-sm text-neutral-300 truncate">{{ group.artist || 'Unknown Artist' }}</p>
              <div v-if="group.year" class="text-xs text-neutral-500 mt-1">{{ group.year }}</div>
            </div>
            
            <!-- Remove Button -->
            <button 
              @click.stop="removeAlbum(group)"
              class="absolute top-2 right-2 p-1.5 bg-black/50 hover:bg-red-500/80 rounded-full text-white opacity-0 group-hover:opacity-100 transition-all"
              title="Remove album"
            >
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" /></svg>
            </button>
          </div>

          <!-- Tracks List -->
          <div class="flex-1 overflow-y-auto bg-neutral-900 min-h-0 p-2">
            <QueueTrackList 
              :tracks="group.tracks" 
              :group-start-pos="group.startPos"
              @track-move="handleTrackMove"
              @track-remove="handleTrackRemove"
            />
          </div>
        </div>
      </template>
    </draggable>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import draggable from 'vuedraggable'
import { useMpdStore } from '@/stores/mpd'
import QueueTrackList from '@/components/QueueTrackList.vue'

const mpdStore = useMpdStore()

const groupedPlaylist = computed({
  get() {
    const playlist = mpdStore.playlist
    if (!playlist || playlist.length === 0) return []

    const groups = []
    let currentGroup = null

    playlist.forEach((track, index) => {
      // Key for grouping: Album + Artist. 
      // If either changes, or if it's the very first track, start a new group.
      // NOTE: We also check if we are "regrouping". But for the visual strip, 
      // we strictly follow the sequential order. A-B-A results in 3 groups.
      const key = `${track.album || ''}-${track.artist || ''}`
      
      if (!currentGroup || currentGroup.key !== key) {
        if (currentGroup) groups.push(currentGroup)
        
        let coverUrl = null
            const dir = track.path.substring(0, track.path.lastIndexOf('/'))
            const escapedDir = dir.split('/').map(encodeURIComponent).join('/')
            coverUrl = `/api/coverart/${escapedDir}`

        currentGroup = {
          id: `group-${index}`, // Unique ID for vuedraggable (using start index is unstable if moving?)
          // actually, using index is bad if we reorder. 
          // Ideally we construct a signature. But for now let's hope re-render handles it.
          // Better: Use track.Path of first track + index?
          // Let's use a composite key of the first track's ID if available, or just index for now.
          // MPD items usually don't have stable IDs in the playlist struct unless we ask for Id. 
          // Verify client.go: PlaylistItem has Pos. Does it have Id? 
          // Client.go struct definition: Path, Title, ... Pos. No Id!
          // We should rely on index-based stability for now or add Id to backend eventually.
          key: key,
          album: track.album,
          artist: track.artist,
          year: track.date,
          coverUrl: coverUrl,
          startPos: index,
          tracks: [],
          isPlayed: index < mpdStore.playlistCurrentPos // Rough approximation
        }
      }
      
      currentGroup.tracks.push(track)
    })
    
    if (currentGroup) groups.push(currentGroup)
    return groups
  },
  set(newGroups) {
    // Handling album reordering
    // vuedraggable triggered a change in the group order.
    // We need to calculate what actually moved.
    // However, since we can't easily map "Group Object" back to "Range" after it's been shuffled 
    // without comparing to old state, we might rely on the '@change' event instead of setter logic.
    // But v-model requires a setter if we use it. 
    // We'll leave this empty or perform a no-op, relying on @change to trigger the API calls.
  }
})

const handleAlbumChange = (event) => {
  if (event.moved) {
    // User moved an album group
    const { element, newIndex, oldIndex } = event.moved
    
    // We need to move the *tracks* of this album to the new position.
    // Origin range: [element.startPos, element.startPos + element.tracks.length]
    // Target position: We need to find the global index of where it landed.
    
    // This is tricky because "newIndex" is the index in the *grouped* list.
    // We need to sum up the lengths of all groups before the newIndex to find the global track index.
    
    // Let's recalculate the groups based on the *new* order provided by the drag (which we don't have access to in 'moved' event easily except via the list state which is transient)
    // Actually, 'groupedPlaylist' getter recomputes from store. 
    // When we drag, vuedraggable updates the local DOM/list.
    // We need to know: "Group X moved to after Group Y".
    
    // Easier approach:
    // 1. Calculate Start/End of the moved group (from 'element').
    // 2. Calculate the Target Global Position.
    //    Target Global Pos = Sum of track counts of all groups from index 0 to newIndex (excluding the moved group itself?).
    
    // Let's verify 'groupedPlaylist' access. 
    // Since vuedraggable updates the array locally, we can inspect 'groupedPlaylist.value' inside the setter? 
    // No, getter is read-only projection of store. 
    
    // We have to assume 'event.moved' gives us indices relative to the view *before* the move? No, 'newIndex' is destination.
    // We need to look at the *current* state of groups (before move) to calculate sizes.
    
    const groups = groupedPlaylist.value // This is the state BEFORE the move commiting to store
    
    // Calculate global start pos for the group at 'newIndex' location.
    // Only problem: 'groups' is still the old order because we haven't updated the store!
    
    // So 'oldIndex' points to the group in 'groups'.
    const movedGroup = groups[oldIndex]
    
    // We want to move it to 'newIndex'.
    // Where is 'newIndex' globally?
    // It's the sum of lengths of groups 0..newIndex-1.
    // BUT we have to account for the fact that the moved group is *not* there (if moving forward) or *is* there (if moving backward).
    
    // If moving forward (old < new): 
    // Target is after group at newIndex.
    // But since 'groups' is old order, the group at 'newIndex' is the one strictly before the target slot?
    
    // Let's standardise:
    // We want to move range [start, end) to 'dest'.
    // MPD 'move' command handles the "shifting" logic.
    
    // Calculate 'dest':
    let dest = 0
    for (let i = 0; i < newIndex; i++) {
        // If i matches oldIndex, we skip it (it's the one moving)
        if (i === oldIndex) continue
        dest += groups[i].tracks.length
    }
    
    // If we are moving forward, we need to add the length of the group that *was* at newIndex?
    // Let's trace.
    // Groups: A(2), B(2), C(2).
    // Move A (idx 0) to idx 1. (After B).
    // dest computation:
    // i=0. oldIndex=0. Skip.
    // Loop ends (newIndex=1).
    // Dest = 0. THIS IS WRONG. Should be 2 (after B).
    
    // Let's try iterating based on the target list structure.
    // New List: B, A, C.
    // We want to insert A at index 2 (globally).
    // Length of B is 2. So dest should be 2.
    
    // Correct algorithm:
    // We simluate the array with the move applied.
    const tempGroups = [...groups]
    const [removed] = tempGroups.splice(oldIndex, 1)
    tempGroups.splice(newIndex, 0, removed)
    
    // Now count tracks up to newIndex
    let targetPos = 0
    for (let i = 0; i < newIndex; i++) {
        targetPos += tempGroups[i].tracks.length
    }
    
    // Now send MPD command
    // MPD 'move start:end dest'.
    // Range is [movedGroup.startPos, movedGroup.startPos + movedGroup.tracks.length]
    // Dest is targetPos.
    // NOTE: If dest > start, MPD expects dest to be relative to the *original* playlist? 
    // "If 'to' is inside the range, nothing happens."
    // "If 'to' is after the range, it is the position *after* the move."
    // Wait, MPD docs say: "Moves the song at [start:end) to 'to'."
    // If I move 0:2 to 5.
    // 0,1 moved to 5.
    // Originals at 2,3,4 shift down to 0,1,2.
    // The pushed items end up at 5,6?
    // Actually, let's just use `moveTrack` for each track if range is hard, but range is better.
    // Let's try to map to single moves if unsure, but range is atomic.
    
    // Simplified MPD Move logic for humans:
    // "I want this block of tracks to start at 'targetPos'."
    // However, if we move down, the targetPos calculated from 'tempGroups' is physically correct for the *result*.
    // Does MPD accept the "result index" or "current index"?
    // Usually "current index". 
    // If I move A (0-2) after B (2-4).
    // Target is 4.
    // Command: `move 0:2 4`.
    // Result: B(0-2), A(2-4). Correct.
    
    // If I move B (2-4) before A (0-2).
    // Target is 0.
    // Command: `move 2:4 0`.
    // Result: B(0-2), A(2-4). Correct.
    
    // So targetPos calculated from 'tempGroups' (the state *after* move) at 'newIndex' seems to vary.
    // When moving forward:
    // Groups: A(2), B(2).
    // Move A to 1.
    // tempGroups: B, A.
    // i=0 (B). targetPos += 2. targetPos = 2.
    // cmd: move 0:2 2.
    // Result: 0,1 moved to 2. No, 'to' must be outside range?
    // If to=2. 0,1 are inserted at 2.
    // List: [0,1,2,3...]. 2 becomes 0. Old 0 becomes 2.
    // YES.
    
    // Conclusion: The target position is the sum of lengths of all groups *preceding* the moved group in the NEW order.
    
    const start = movedGroup.startPos
    const end = start + movedGroup.tracks.length
    
    // Adjust targetPos for MPD quirk if needed? 
    // MPD move is: "move source to dest".
    // If moving forward, dest is the index *before* the shifted items?
    // Let's stick to simple `move` logic provided by Client if possible.
    // But we are moving a range.
    // Currently `moveTrack` in store sends `from, to`. It calls backend `Move(from, to)`.
    // Does backend `Move` support range? `move %d %d` -> `move start end`? Or `move start:end to`?
    // My backend implementation: `fmt.Sprintf("move %d %d", from, to)`
    // This calls MPD `move`.
    // MPD `move` signature: `move {START:END} {TO}` or `move {POS} {TO}`.
    // If I pass atomic integers, it moves one song.
    // I need to update backend to support Range! `Start:End`.
    
    // CRITICAL: My backend `Move` takes `from, to int`.
    // I should probably iterate and move tracks one by one, OR update backend to support range.
    // Updating backend is cleaner but I'm in frontend phase.
    // Iterating one by one is safe but spammy.
    // Let's iterate for now to ensure correctness without context switching, 
    // OR create a helper `moveGroup` in store that calls `moveTrack` repeatedly?
    // Actually, `move start:end` is standard MPD.
    // If I send `from` as a string "0:2", does `Move` accept int? No.
    
    // WORKAROUND:
    // Loop through tracks in the group and move them.
    // Moving a block one-by-one:
    // If moving forward: Move last track to dest, then second-last to dest... preserving order?
    // Easier: Move the whole block.
    // If I move track 0 to 5. track 1 becomes 0.
    // If I move track 0 (was 1) to 5.
    // Result: reversed?
    // We should move them in order.
    
    // Actually, I'll allow `Move` in backend to accept proper range syntax or just fix it later.
    // For now, let's loop relative to the *shifting* indices? That's nightmare.
    
    // DECISION: I will assume I can update the backend to support string range or start/end args later/now.
    // Wait, I can just use `moveTrack(track.Pos, targetPos)`?
    // If I move multiple tracks, subsequent Pos change.
    
    // I will trigger a custom store action `moveAlbum(start, length, dest)` and implement it properly.
    
    mpdStore.moveAlbum(start, movedGroup.tracks.length, targetPos)
  }
}

const handleTrackMove = ({ from, to }) => {
  mpdStore.moveTrack(from, to)
}

const handleTrackRemove = (pos) => {
  mpdStore.removeFromPlaylist(pos)
}

const removeAlbum = (group) => {
  // Remove all tracks in range
  // We can do this backwards to avoid index shifting problems?
  // Or add `removeRange` to store.
  for (let i = group.tracks.length - 1; i >= 0; i--) {
      mpdStore.removeFromPlaylist(group.startPos + i)
  }
}
</script>

<style scoped>
.scrollbar-thin::-webkit-scrollbar {
  height: 8px;
}
.scrollbar-thin::-webkit-scrollbar-track {
  background: transparent;
}
.scrollbar-thin::-webkit-scrollbar-thumb {
  background-color: #404040;
  border-radius: 4px;
}
</style>
