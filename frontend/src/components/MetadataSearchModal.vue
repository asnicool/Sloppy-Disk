<template>
  <BaseModal
    :model-value="isOpen"
    @update:model-value="$emit('close')"
    title="Find Metadata & Cover Art"
    class="metadata-modal"
  >
    <div class="space-y-6">
      <!-- Search Form -->
      <div class="space-y-4">
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div class="space-y-1">
            <label class="text-xs font-medium text-neutral-500 uppercase">Artist</label>
            <input
              v-model="displayArtist"
              placeholder="Artist"
              class="w-full px-3 py-2 bg-neutral-900 border border-neutral-700 rounded-lg text-white placeholder-neutral-500 focus:border-blue-500 focus:ring-1 focus:ring-blue-500 transition-all"
              @keyup.enter="handleSearch"
            />
          </div>
          <div class="space-y-1">
            <label class="text-xs font-medium text-neutral-500 uppercase">Album</label>
            <input
              v-model="displayAlbum"
              placeholder="Album"
              class="w-full px-3 py-2 bg-neutral-900 border border-neutral-700 rounded-lg text-white placeholder-neutral-500 focus:border-blue-500 focus:ring-1 focus:ring-blue-500 transition-all"
              @keyup.enter="handleSearch"
            />
          </div>
        </div>
        
        <button
          @click="handleSearch"
          :disabled="loading || !displayArtist || !displayAlbum"
          class="w-full py-2.5 bg-blue-600 hover:bg-blue-500 text-white rounded-lg font-bold transition-all shadow-lg shadow-blue-900/20 disabled:opacity-50 flex items-center justify-center gap-2"
        >
          <svg v-if="loading" class="animate-spin h-5 w-5" viewBox="0 0 24 24">
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" fill="none" />
            <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
          </svg>
          {{ loading ? 'Searching...' : 'Search Metadata' }}
        </button>
      </div>

      <!-- Provider Filter -->
      <div class="flex flex-wrap gap-2">
        <button
          v-for="provider in providers"
          :key="provider"
          @click="toggleProvider(provider)"
          :class="[
            'px-3 py-1.5 text-xs font-bold rounded-full transition-all duration-200 border',
            isProviderSelected(provider)
              ? 'bg-blue-600/10 border-blue-500 text-blue-400'
              : 'bg-neutral-900 border-neutral-700 text-neutral-500 hover:border-neutral-500 hover:text-neutral-300'
          ]"
        >
          {{ provider }}
        </button>
      </div>

      <!-- Manual Cover Input -->
      <div class="space-y-2">
        <h4 class="text-xs font-medium text-neutral-500 uppercase">Manual Cover Art</h4>
        <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
          <div class="md:col-span-3 space-y-2">
            <div 
              class="relative border-2 border-dashed border-neutral-700 rounded-xl p-4 transition-colors hover:border-neutral-500 group flex flex-col items-center justify-center min-h-[120px]"
              :class="{ 'border-blue-500 bg-blue-500/5': isDragging }"
              @dragover.prevent="isDragging = true"
              @dragleave.prevent="isDragging = false"
              @drop.prevent="handleDrop"
            >
              <input 
                type="file" 
                ref="fileInput" 
                class="hidden" 
                accept="image/*" 
                @change="handleFileSelect"
              />
              <div v-if="!uploadedFile" class="text-center space-y-2">
                <svg class="w-8 h-8 text-neutral-600 mx-auto group-hover:text-neutral-400 transition-colors" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
                </svg>
                <p class="text-sm text-neutral-500">
                  <button @click="fileInput.click()" class="text-blue-500 hover:underline font-bold">Upload a file</button>
                  or drag and drop
                </p>
              </div>
              <div v-else class="flex items-center gap-3 w-full">
                <div class="w-16 h-16 rounded overflow-hidden flex-shrink-0 bg-neutral-800">
                  <img :src="uploadedFileUrl" class="w-full h-full object-cover" />
                </div>
                <div class="flex-1 min-w-0">
                  <p class="text-sm text-white font-medium truncate">{{ uploadedFile.name }}</p>
                  <p class="text-xs text-neutral-500">{{ (uploadedFile.size / 1024).toFixed(1) }} KB</p>
                </div>
                <button @click="clearUpload" class="text-neutral-500 hover:text-red-500">
                  <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
            </div>
            
            <input
              v-model="manualCoverUrl"
              placeholder="Paste image URL here..."
              class="w-full px-3 py-2 bg-neutral-900 border border-neutral-700 rounded-lg text-white placeholder-neutral-500 focus:border-blue-500 focus:ring-1 focus:ring-blue-500 transition-all text-sm"
            />
          </div>
          <div class="flex flex-col gap-2">
             <button
              @click="handleApplyManualCover"
              :disabled="applying || (!manualCoverUrl && !uploadedFile)"
              class="h-full px-4 bg-green-600 hover:bg-green-500 text-white rounded-lg font-bold transition-all disabled:opacity-50"
            >
              Apply Cover
            </button>
          </div>
        </div>
      </div>

      <hr class="border-neutral-800" />

      <!-- Results -->
      <div v-if="candidates.length > 0" class="space-y-4">
        <div class="flex items-center justify-between">
          <h4 class="text-xs font-medium text-neutral-500 uppercase">Search Results ({{ candidates.length }})</h4>
        </div>
        
        <div class="grid grid-cols-1 gap-3">
          <div
            v-for="candidate in candidates"
            :key="`${candidate.source}-${candidate.externalId}`"
            class="group"
          >
            <div
              @click="selectCandidate(candidate)"
              :class="[
                'p-4 rounded-xl cursor-pointer transition-all border',
                selectedCandidate?.externalId === candidate.externalId && selectedCandidate?.source === candidate.source
                  ? 'bg-blue-600/10 border-blue-500/50 shadow-lg shadow-blue-900/10'
                  : 'bg-neutral-800/50 border-neutral-700 hover:bg-neutral-800 hover:border-neutral-600'
              ]"
            >
              <div class="flex justify-between items-start">
                <div class="min-w-0">
                  <div class="font-bold text-white truncate">{{ candidate.album }}</div>
                  <div class="text-sm text-neutral-400 truncate">{{ candidate.artist }}</div>
                  <div class="flex items-center gap-2 mt-2">
                    <span class="text-[10px] font-bold px-1.5 py-0.5 rounded bg-neutral-700 text-neutral-300 uppercase leading-none">
                      {{ candidate.source }}
                    </span>
                    <span v-if="candidate.year" class="text-xs text-neutral-500">{{ candidate.year }}</span>
                    <span v-if="candidate.genre" class="text-xs text-neutral-500 truncate max-w-[100px] border-l border-neutral-700 pl-2">
                      {{ candidate.genre }}
                    </span>
                  </div>
                </div>
                <div class="text-right flex-shrink-0">
                  <div :class="[
                    'text-xl font-black italic',
                    candidate.confidence > 80 ? 'text-green-500' : candidate.confidence > 50 ? 'text-amber-500' : 'text-neutral-500'
                  ]">
                    {{ Math.round(candidate.confidence) }}%
                  </div>
                  <div v-if="candidate.tracks" class="text-[10px] text-neutral-600 font-bold uppercase mt-1">
                    {{ candidate.tracks.length }} tracks
                  </div>
                </div>
              </div>

              <!-- Selected Details -->
              <div v-if="selectedCandidate?.externalId === candidate.externalId && selectedCandidate?.source === candidate.source" class="mt-6 space-y-6 animate-in slide-in-from-top-2 duration-300" @click.stop>
                <div class="grid grid-cols-2 gap-4">
                  <div class="space-y-1">
                    <label class="text-[10px] font-bold text-neutral-500 uppercase">Album</label>
                    <input v-model="selectedCandidate.album" @click.stop class="w-full px-2 py-1.5 bg-neutral-900 border border-neutral-700 rounded text-sm text-white focus:border-blue-500 outline-none"/>
                  </div>
                  <div class="space-y-1">
                    <label class="text-[10px] font-bold text-neutral-500 uppercase">Artist</label>
                    <input v-model="selectedCandidate.artist" @click.stop class="w-full px-2 py-1.5 bg-neutral-900 border border-neutral-700 rounded text-sm text-white focus:border-blue-500 outline-none"/>
                  </div>
                  <div class="space-y-1">
                    <label class="text-[10px] font-bold text-neutral-500 uppercase">Year</label>
                    <input v-model="selectedCandidate.year" @click.stop class="w-full px-2 py-1.5 bg-neutral-900 border border-neutral-700 rounded text-sm text-white focus:border-blue-500 outline-none"/>
                  </div>
                  <div class="space-y-1">
                    <label class="text-[10px] font-bold text-neutral-500 uppercase">Genre</label>
                    <input v-model="selectedCandidate.genre" @click.stop class="w-full px-2 py-1.5 bg-neutral-900 border border-neutral-700 rounded text-sm text-white focus:border-blue-500 outline-none"/>
                  </div>
                </div>

                <!-- Track Matching & Granular Controls -->
                <div v-if="selectedCandidate.tracks?.length && localTracks.length > 0" class="space-y-4" @click.stop>
                  <div class="flex items-center justify-between">
                    <div class="text-[10px] font-bold text-neutral-500 uppercase">Track Matching</div>
                    <button @click.stop="runTrackMatching" class="text-xs text-blue-400 hover:text-blue-300">
                      {{ trackMatchResult ? 'Refresh Match' : 'Match Tracks' }}
                    </button>
                  </div>

                  <!-- Match Statistics -->
                  <div v-if="trackMatchResult" class="flex flex-wrap gap-2 text-[10px]">
                    <span class="px-2 py-1 rounded bg-green-900/30 text-green-400 border border-green-700/30">
                      ✓ {{ trackMatchResult.stats.matched }} matched
                    </span>
                    <span v-if="trackMatchResult.stats.libraryOnly > 0" class="px-2 py-1 rounded bg-amber-900/30 text-amber-400 border border-amber-700/30">
                      − {{ trackMatchResult.stats.libraryOnly }} only in library
                    </span>
                    <span v-if="trackMatchResult.stats.metadataOnly > 0" class="px-2 py-1 rounded bg-blue-900/30 text-blue-400 border border-blue-700/30">
                      + {{ trackMatchResult.stats.metadataOnly }} only in metadata
                    </span>
                    <span v-if="trackMatchResult.stats.orderChanged" class="px-2 py-1 rounded bg-purple-900/30 text-purple-400 border border-purple-700/30">
      ⇄ Different order
                    </span>
                  </div>

                  <!-- Matched Tracks (Side-by-Side) -->
                  <div v-if="trackMatchResult?.matches?.length" class="space-y-2">
                    <div class="text-[10px] font-bold text-neutral-600 uppercase">Matched Tracks</div>
                    <div class="max-h-80 overflow-y-auto rounded-lg border border-neutral-700/50 bg-neutral-900/30 divide-y divide-neutral-800">
                      <div
                        v-for="(match, idx) in trackMatchResult.matches"
                        :key="`match-${idx}`"
                        class="p-3 hover:bg-neutral-800/50 transition-colors"
                        :class="{
                          'bg-green-900/10': match.matchType === 'exact' && match.titleSimilarity >= 95,
                          'bg-amber-900/10': match.matchType === 'fuzzy' || match.titleSimilarity < 95
                        }"
                      >
                        <!-- Match Header -->
                        <div class="flex items-center justify-between mb-2">
                          <div class="flex items-center gap-2">
                            <span class="text-[10px] font-mono text-neutral-500">
                              {{ match.library.disc ? match.library.disc + '-' : '' }}{{ String(match.library.track || 0).padStart(2, '0') }}
                            </span>
                            <span :class="['text-[10px] font-bold', getMatchStatusClass(match)]">
                              {{ getMatchStatus(match) }}
                            </span>
                            <span class="text-[10px] text-neutral-600">
                              {{ Math.round(match.titleSimilarity) }}% similar
                            </span>
                          </div>
                          <button
                            @click.stop="toggleTrackUpdate(match)"
                            :class="[
                              'px-2 py-1 rounded text-[10px] font-bold uppercase transition-all',
                              trackUpdatesEnabled[getTrackId(match)]
                                ? 'bg-green-600 text-white'
                                : 'bg-neutral-700 text-neutral-400 hover:bg-neutral-600'
                            ]"
                          >
                            {{ trackUpdatesEnabled[getTrackId(match)] ? '✓ Update' : 'Skip' }}
                          </button>
                        </div>

                        <!-- Side-by-Side Comparison -->
                        <div class="grid grid-cols-2 gap-3 text-xs">
                          <!-- Library Track (Current) -->
                          <div class="space-y-1">
                            <div class="text-[10px] font-bold text-neutral-500 uppercase">Library (Current)</div>
                            <input
                              :value="match.library.title"
                              @input="updateLibraryTrackTitle(match, $event.target.value)"
                              class="w-full px-2 py-1 bg-neutral-900 border border-neutral-700 rounded text-neutral-300 focus:border-blue-500 outline-none"
                              @click.stop
                            />
                          </div>

                          <!-- Metadata Track (New) -->
                          <div class="space-y-1">
                            <div class="text-[10px] font-bold text-blue-400 uppercase">Metadata (New)</div>
                            <input
                              v-model="match.metadata.title"
                              class="w-full px-2 py-1 bg-neutral-900 border border-blue-700/50 rounded text-white focus:border-blue-500 outline-none"
                              @click.stop
                            />
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>

                  <!-- Library-Only Tracks -->
                  <div v-if="trackMatchResult?.libraryOnly?.length" class="space-y-2">
                    <div class="text-[10px] font-bold text-amber-600 uppercase">Only in Library (No Metadata Match)</div>
                    <div class="max-h-40 overflow-y-auto rounded-lg border border-amber-900/30 bg-amber-900/10 divide-y divide-amber-900/20">
                      <div
                        v-for="(track, idx) in trackMatchResult.libraryOnly"
                        :key="`libonly-${idx}`"
                        class="p-2 flex items-center gap-3"
                      >
                        <span class="text-[10px] font-mono text-amber-700">
                          {{ track.disc ? track.disc + '-' : '' }}{{ String(track.track || 0).padStart(2, '0') }}
                        </span>
                        <input
                          :value="track.title"
                          disabled
                          class="flex-1 px-2 py-1 bg-neutral-900/50 border border-neutral-800 rounded text-neutral-500 text-xs"
                        />
                        <span class="text-[10px] text-amber-600 font-bold">No Match</span>
                      </div>
                    </div>
                  </div>

                  <!-- Metadata-Only Tracks -->
                  <div v-if="trackMatchResult?.metadataOnly?.length" class="space-y-2">
                    <div class="text-[10px] font-bold text-blue-600 uppercase">Only in Metadata (Not in Library)</div>
                    <div class="max-h-40 overflow-y-auto rounded-lg border border-blue-900/30 bg-blue-900/10 divide-y divide-blue-900/20">
                      <div
                        v-for="(track, idx) in trackMatchResult.metadataOnly"
                        :key="`metaonly-${idx}`"
                        class="p-2 flex items-center gap-3"
                      >
                        <span class="text-[10px] font-mono text-blue-700">
                          {{ track.disc ? track.disc + '-' : '' }}{{ String(track.track || 0).padStart(2, '0') }}
                        </span>
                        <input
                          v-model="track.title"
                          class="flex-1 px-2 py-1 bg-neutral-900 border border-blue-700/30 rounded text-white text-xs"
                          @click.stop
                        />
                        <span class="text-[10px] text-blue-600 font-bold">Extra</span>
                      </div>
                    </div>
                  </div>

                  <!-- Granular Update Controls -->
                  <div class="pt-4 border-t border-neutral-800 space-y-3">
                    <div class="text-[10px] font-bold text-neutral-500 uppercase">Update Options</div>

                    <div class="grid grid-cols-2 gap-3">
                      <!-- Album-level metadata -->
                      <div class="space-y-2 p-3 rounded-lg bg-neutral-900/50 border border-neutral-800">
                        <div class="text-[10px] font-bold text-neutral-400 uppercase mb-2">Album Fields</div>

                        <label class="flex items-center gap-2 text-xs cursor-pointer hover:text-white">
                          <input
                            type="checkbox"
                            v-model="updateOptions.genre"
                            class="w-4 h-4 rounded border-neutral-700 text-blue-600 focus:ring-blue-500 focus:ring-offset-0"
                          />
                          <span>Genre</span>
                        </label>

                        <label class="flex items-center gap-2 text-xs cursor-pointer hover:text-white">
                          <input
                            type="checkbox"
                            v-model="updateOptions.year"
                            class="w-4 h-4 rounded border-neutral-700 text-blue-600 focus:ring-blue-500 focus:ring-offset-0"
                          />
                          <span>Year</span>
                        </label>

                        <label class="flex items-center gap-2 text-xs cursor-pointer hover:text-white">
                          <input
                            type="checkbox"
                            v-model="updateOptions.album"
                            class="w-4 h-4 rounded border-neutral-700 text-blue-600 focus:ring-blue-500 focus:ring-offset-0"
                          />
                          <span>Album Name</span>
                        </label>

                        <label class="flex items-center gap-2 text-xs cursor-pointer hover:text-white">
                          <input
                            type="checkbox"
                            v-model="updateOptions.artist"
                            class="w-4 h-4 rounded border-neutral-700 text-blue-600 focus:ring-blue-500 focus:ring-offset-0"
                          />
                          <span>Artist</span>
                        </label>
                      </div>

                      <!-- Track-level metadata -->
                      <div class="space-y-2 p-3 rounded-lg bg-neutral-900/50 border border-neutral-800">
                        <div class="text-[10px] font-bold text-neutral-400 uppercase mb-2">Track Fields</div>

                        <label class="flex items-center gap-2 text-xs cursor-pointer hover:text-white">
                          <input
                            type="checkbox"
                            v-model="updateOptions.trackTitles"
                            class="w-4 h-4 rounded border-neutral-700 text-blue-600 focus:ring-blue-500 focus:ring-offset-0"
                          />
                          <span>Track Titles</span>
                        </label>

                        <label class="flex items-center gap-2 text-xs cursor-pointer hover:text-white">
                          <input
                            type="checkbox"
                            v-model="updateOptions.trackNumbers"
                            class="w-4 h-4 rounded border-neutral-700 text-blue-600 focus:ring-blue-500 focus:ring-offset-0"
                          />
                          <span>Track Numbers</span>
                        </label>

                        <label class="flex items-center gap-2 text-xs cursor-pointer hover:text-white">
                          <input
                            type="checkbox"
                            v-model="updateOptions.coverArt"
                            class="w-4 h-4 rounded border-neutral-700 text-blue-600 focus:ring-blue-500 focus:ring-offset-0"
                          />
                          <span>Cover Art</span>
                        </label>

                        <div class="pt-2 border-t border-neutral-800">
                          <div class="text-[10px] text-neutral-500 mb-1">Individual Track Control</div>
                          <div class="text-[10px] text-neutral-600">
                            {{ Object.values(trackUpdatesEnabled).filter(v => v).length }} / {{ trackMatchResult?.matches?.length || 0 }} tracks enabled
                          </div>
                        </div>
                      </div>
                    </div>

                    <div class="text-[10px] text-neutral-500 italic">
                      💡 If nothing is checked, all metadata will be updated. Check specific items to update only those.
                    </div>
                  </div>
                </div>

                <!-- Fallback: Original track list when no library tracks -->
                <div v-else-if="selectedCandidate.tracks?.length" class="space-y-2" @click.stop>
                  <div class="text-[10px] font-bold text-neutral-500 uppercase">Track List</div>
                  <div class="max-h-60 overflow-y-auto rounded-lg border border-neutral-700/50 bg-neutral-900/50 divide-y divide-neutral-800">
                    <div v-for="track in selectedCandidate.tracks" :key="track.track" class="flex gap-4 items-center p-2 group/track">
                       <span class="text-xs font-mono text-neutral-600 group-hover/track:text-blue-500 transition-colors">
                        {{ track.disc ? track.disc + '-' : '' }}{{ track.track.toString().padStart(2, '0') }}
                      </span>
                      <input v-model="track.title" @click.stop class="flex-1 bg-transparent border-none text-xs text-white p-0 focus:ring-0 placeholder-neutral-700" placeholder="Unnamed Track"/>
                    </div>
                  </div>
                  <p class="text-[10px] text-neutral-600">Library tracks not available - edit metadata directly above</p>
                </div>

                <!-- Cover Art -->
                <div v-if="coverArtOptions.length > 0" class="space-y-2" @click.stop>
                  <div class="text-[10px] font-bold text-neutral-500 uppercase">Cover Selection</div>
                  <div class="flex gap-3 flex-wrap">
                    <div
                      v-for="(art, idx) in coverArtOptions.slice(0, 6)"
                      :key="idx"
                      class="relative w-24 h-24 group/art rounded-lg overflow-hidden cursor-pointer"
                      @click.stop="selectedCoverArt = art"
                    >
                      <img
                        :src="art.thumbnail || art.url"
                        :class="[
                          'w-full h-full object-cover transition-all duration-300',
                          selectedCoverArt?.url === art.url ? 'ring-2 ring-blue-500 scale-105' : 'opacity-60 grayscale hover:opacity-100 hover:grayscale-0'
                        ]"
                      />
                      <!-- Dimension Badge -->
                      <div v-if="art.width && art.height" class="absolute bottom-1 right-1 px-1.5 py-0.5 bg-black/70 backdrop-blur-md rounded text-[9px] font-bold text-white opacity-0 group-hover/art:opacity-100 transition-opacity">
                        {{ art.width }}x{{ art.height }}
                      </div>
                      
                      <div v-if="selectedCoverArt?.url === art.url" class="absolute inset-0 bg-blue-500/20 flex items-center justify-center">
                        <svg class="w-6 h-6 text-white" fill="currentColor" viewBox="0 0 20 20">
                          <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" />
                        </svg>
                      </div>
                    </div>
                  </div>
                </div>

                <div class="pt-2 space-y-3" @click.stop>
                  <button
                    @click.stop="handleApply"
                    :disabled="applying"
                    class="w-full py-3 bg-green-600 hover:bg-green-500 text-white rounded-xl font-black uppercase tracking-wider transition-all shadow-lg shadow-green-900/20 disabled:opacity-50"
                  >
                    {{ applying === 'metadata' ? 'Tagging Files...' : 'Apply All Changes' }}
                  </button>

                  <button
                    @click.stop="handleApplyCoverOnly"
                    :disabled="applying || !selectedCoverArt"
                    class="w-full py-3 bg-blue-600 hover:bg-blue-500 text-white rounded-xl font-bold uppercase tracking-wider transition-all shadow-lg shadow-blue-900/20 disabled:opacity-50"
                  >
                    {{ applying === 'cover' ? 'Setting Cover...' : 'Apply Cover Only' }}
                  </button>

                  <p v-if="!selectedCoverArt" class="text-xs text-neutral-500 text-center">
                    Select a cover image above to enable "Apply Cover Only"
                  </p>

                  <div v-if="applyResult" class="mt-4 p-4 bg-green-900/20 border border-green-500/30 rounded-xl">
                    <div class="flex items-center gap-2 text-green-400 font-bold text-sm">
                      <svg class="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
                        <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
                      </svg>
                      Update Successful
                    </div>
                    <p class="text-xs text-neutral-400 mt-1">
                      Applied to {{ applyResult.updatedFiles }} of {{ applyResult.totalFiles }} files.
                    </p>
                    <div v-if="applyResult.errors?.length" class="mt-2 space-y-1">
                      <div v-for="(err, idx) in applyResult.errors" :key="idx" class="text-[10px] text-amber-500 font-medium">
                        ⚠ {{ err }}
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- No Results / Initial State -->
      <div v-else-if="searched && !loading" class="py-12 text-center space-y-2">
        <svg class="w-12 h-12 text-neutral-800 mx-auto" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9.172 9.172a4 4 0 015.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
        <p class="text-neutral-500 font-medium">No metadata found for this search.</p>
        <button @click="clearSearch" class="text-xs text-blue-500 hover:underline">Clear search terms</button>
      </div>
      
      <div v-else-if="!searched && !loading" class="py-12 text-center text-neutral-600">
        Enter search terms to find metadata and high-res cover art.
      </div>
    </div>
  </BaseModal>
</template>

<script setup>
import { ref, watch, computed } from 'vue'
import { useMetadata } from '../composables/useMetadata'
import { useMpdStore } from '../stores/mpdStore'
import BaseModal from './BaseModal.vue'
import { matchTracks, getMatchStatus, getMatchStatusClass } from '@/utils/trackMatcher'

const props = defineProps({
  isOpen: Boolean,
  initialArtist: String,
  initialAlbum: String,
  albumPath: String,
  trackCount: Number,
  duration: Number,
  libraryTracks: Array // New prop: library tracks for matching
})

const emit = defineEmits(['close', 'applied', 'coverUpdated'])

const mpdStore = useMpdStore()
const {
  candidates,
  selectedCandidate,
  loading,
  error,
  applyResult,
  searchMetadata,
  getMetadataDetails,
  applyMetadata,
  searchCoverArt,
  clearSelection
} = useMetadata()

const searchArtist = ref('')
const searchAlbum = ref('')
const selectedProviders = ref(['MusicBrainz', 'Discogs', 'GNUDb', 'AlbumArt.digital'])
const providers = ['MusicBrainz', 'Discogs', 'GNUDb', 'AlbumArt.digital']
const coverArtOptions = ref([])
const selectedCoverArt = ref(null)
const applying = ref(false)
const searched = ref(false)

// Manual Cover State
const manualCoverUrl = ref('')
const isDragging = ref(false)
const uploadedFile = ref(null)
const uploadedFileUrl = ref(null)
const fileInput = ref(null)

// Track Matching State
const localTracks = ref([])
const trackMatchResult = ref(null)
const trackUpdatesEnabled = ref({}) // Map of track IDs to boolean (enable/disable individual track updates)

// Granular Update Options
const updateOptions = ref({
  genre: false,
  year: false,
  album: false,
  artist: false,
  trackTitles: false,
  trackNumbers: false,
  coverArt: false
})

// Display value for inputs
const displayArtist = computed({
  get: () => searchArtist.value || props.initialArtist,
  set: (val) => { searchArtist.value = val }
})

const displayAlbum = computed({
  get: () => searchAlbum.value || props.initialAlbum,
  set: (val) => { searchAlbum.value = val }
})

const handleSearch = async () => {
  const artist = displayArtist.value
  const album = displayAlbum.value
  
  if (!artist || !album) return
  
  searched.value = true
  clearSelection()
  coverArtOptions.value = []
  selectedCoverArt.value = null
  
  await searchMetadata(
    artist,
    album,
    selectedProviders.value,
    props.trackCount,
    props.duration
  )
  
  // Also search for cover art
  const results = await searchCoverArt(artist, album)
  coverArtOptions.value = results
}

const selectCandidate = async (candidate) => {
  if (selectedCandidate.value?.externalId === candidate.externalId && selectedCandidate.value?.source === candidate.source) {
    selectedCandidate.value = null
    trackMatchResult.value = null
    return
  }

  selectedCandidate.value = candidate
  await getMetadataDetails(candidate.source, candidate.externalId)

  // Fetch library tracks for matching
  await fetchLibraryTracks()
}

const handleApply = async () => {
  if (!selectedCandidate.value || !props.albumPath) return

  applying.value = 'metadata'
  try {
    // Build metadata to apply based on granular options
    const metadataToApply = buildMetadataToApply()

    const result = await applyMetadata(
      props.albumPath,
      metadataToApply,
      selectedCoverArt.value?.url || ''
    )
    if (result) {
      emit('applied', result)
    }
  } catch (e) {
    console.error('Failed to apply metadata:', e)
  } finally {
    applying.value = false
  }
}

const buildMetadataToApply = () => {
  const candidate = selectedCandidate.value

  // If no granular options selected, return everything (current behavior)
  if (!hasUpdateOptions.value) {
    return candidate
  }

  // Build filtered metadata based on options
  const filtered = {
    ...candidate,
    // Album-level fields
    album: updateOptions.value.album ? candidate.album : undefined,
    artist: updateOptions.value.artist ? candidate.artist : undefined,
    year: updateOptions.value.year ? candidate.year : undefined,
    genre: updateOptions.value.genre ? candidate.genre : undefined,

    // Track-level fields
    tracks: candidate.tracks?.map(track => {
      const filteredTrack = { ...track }

      // Filter track title
      if (!updateOptions.value.trackTitles) {
        delete filteredTrack.title
      }

      // Filter track/disc numbers
      if (!updateOptions.value.trackNumbers) {
        delete filteredTrack.track
        delete filteredTrack.disc
      }

      return filteredTrack
    }).filter(t => Object.keys(t).length > 0)
  }

  // Remove undefined keys
  Object.keys(filtered).forEach(key => {
    if (filtered[key] === undefined) {
      delete filtered[key]
    }
  })

  return filtered
}

const fetchLibraryTracks = async () => {
  if (!props.initialArtist || !props.initialAlbum) return

  try {
    const response = await mpdStore.fetchAlbumSongs(props.initialArtist, props.initialAlbum)

    if (response.success && response.data?.tracks) {
      localTracks.value = response.data.tracks
      console.log('[MetadataSearchModal] Loaded library tracks:', localTracks.value.length)
    } else if (response.tracks) {
      // Cached response
      localTracks.value = response.tracks
      console.log('[MetadataSearchModal] Loaded library tracks (cached):', localTracks.value.length)
    }
  } catch (error) {
    console.error('[MetadataSearchModal] Failed to fetch library tracks:', error)
  }
}

const handleApplyCoverOnly = async () => {
  if (!selectedCoverArt.value || !props.albumPath) return
  
  applying.value = 'cover'
  try {
    await mpdStore.applyCoverArt(props.albumPath, selectedCoverArt.value.url)
    emit('coverUpdated')
  } catch (e) {
    console.error('Failed to apply cover:', e)
  } finally {
    applying.value = false
  }
}

const handleApplyManualCover = async () => {
  if (!props.albumPath) return
  
  applying.value = true
  try {
    if (uploadedFile.value) {
      await mpdStore.uploadCoverArt(props.albumPath, uploadedFile.value)
    } else if (manualCoverUrl.value) {
      await mpdStore.applyCoverArt(props.albumPath, manualCoverUrl.value)
    }
    emit('coverUpdated')
    manualCoverUrl.value = ''
    clearUpload()
  } catch (e) {
    console.error('Failed to apply manual cover:', e)
  } finally {
    applying.value = false
  }
}

// Drag & Drop Handlers
const handleDrop = (e) => {
  isDragging.value = false
  const files = e.dataTransfer.files
  if (files && files.length > 0) {
    processFile(files[0])
  }
}

const handleFileSelect = (e) => {
  const files = e.target.files
  if (files && files.length > 0) {
    processFile(files[0])
  }
}

const processFile = (file) => {
  if (!file.type.startsWith('image/')) {
    alert('Please select an image file')
    return
  }
  uploadedFile.value = file
  uploadedFileUrl.value = URL.createObjectURL(file)
}

const clearUpload = () => {
  if (uploadedFileUrl.value) {
    URL.revokeObjectURL(uploadedFileUrl.value)
  }
  uploadedFile.value = null
  uploadedFileUrl.value = null
  if (fileInput.value) fileInput.value.value = ''
}

const clearSearch = () => {
  searchArtist.value = ''
  searchAlbum.value = ''
  searched.value = false
  candidates.value = []
}

const toggleProvider = (provider) => {
  const index = selectedProviders.value.indexOf(provider)
  if (index === -1) {
    selectedProviders.value.push(provider)
  } else {
    selectedProviders.value.splice(index, 1)
  }
}

const isProviderSelected = (provider) => {
  return selectedProviders.value.includes(provider)
}

// Track Matching Functions
const runTrackMatching = () => {
  if (!selectedCandidate.value?.tracks || !localTracks.value.length) {
    console.warn('[MetadataSearchModal] Cannot run track matching - missing data')
    return
  }

  const result = matchTracks(localTracks.value, selectedCandidate.value.tracks)
  trackMatchResult.value = result

  // Initialize track update flags (all enabled by default)
  result.matches.forEach((match, idx) => {
    const trackId = getTrackId(match)
    trackUpdatesEnabled.value[trackId] = true
  })

  console.log('[MetadataSearchModal] Track matching complete:', result.stats)
}

const getTrackId = (match) => {
  return `${match.library.disc || 1}-${match.library.track || 0}`
}

const toggleTrackUpdate = (match) => {
  const trackId = getTrackId(match)
  trackUpdatesEnabled.value[trackId] = !trackUpdatesEnabled.value[trackId]
}

const updateLibraryTrackTitle = (match, newTitle) => {
  // Update the library track title in the match result
  if (match && match.library) {
    match.library.title = newTitle

    // Also update in local tracks if present
    const localTrack = localTracks.value.find(t =>
      t.disc === match.library.disc &&
      t.track === match.library.track
    )
    if (localTrack) {
      localTrack.title = newTitle
    }
  }
}

// Check if any update options are selected
const hasUpdateOptions = computed(() => {
  return Object.values(updateOptions.value).some(v => v)
})

// Initialize with props
watch(() => props.isOpen, async (isOpen) => {
  if (isOpen) {
    searchArtist.value = ''
    searchAlbum.value = ''
    searched.value = false
    clearSelection()
    clearUpload()
    manualCoverUrl.value = ''

    // Load library tracks for matching
    await fetchLibraryTracks()
  } else {
    // Clean up on close
    localTracks.value = []
    trackMatchResult.value = null
    trackUpdatesEnabled.value = {}
    updateOptions.value = {
      genre: false,
      year: false,
      album: false,
      artist: false,
      trackTitles: false,
      trackNumbers: false,
      coverArt: false
    }
  }
})
</script>

<style scoped>
.metadata-modal :deep(.relative) {
  scroll-behavior: smooth;
}

::-webkit-scrollbar {
  width: 4px;
}

::-webkit-scrollbar-track {
  background: transparent;
}

::-webkit-scrollbar-thumb {
  background: #333;
  border-radius: 10px;
}

::-webkit-scrollbar-thumb:hover {
  background: #444;
}
</style>
