# Metadata Search UI Implementation Plan

## Overview

The backend already has complete metadata search functionality with providers for:
- MusicBrainz
- Discogs
- FreeDB
- AlbumArt.digital

This document outlines the frontend implementation to expose these capabilities to users.

## Backend API Endpoints Available

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/metadata/search` | GET | Search across all providers |
| `/api/metadata/details` | GET | Get detailed metadata for a release |
| `/api/metadata/apply` | POST | Apply metadata and cover art to files |

## Phase 7: Frontend Metadata Search UI

### 7.1 API Service Layer

Create `frontend/src/services/metadataService.js`:

```javascript
import axios from 'axios';

const api = axios.create({
  baseURL: '/api',
  timeout: 30000,
});

// Search metadata across all providers
export const searchMetadata = async (query, options = {}) => {
  const { providers, limit = 10 } = options;
  return api.get('/metadata/search', {
    params: {
      q: query,
      providers: providers?.join(','),
      limit,
    },
  });
};

// Get detailed metadata for a specific release
export const getMetadataDetails = async (provider, id) => {
  return api.get('/metadata/details', {
    params: { provider, id },
  });
};

// Apply metadata to files
export const applyMetadata = async (provider, id, options = {}) => {
  const { paths, applyTags = true, applyCover = true, coverIndex = 0 } = options;
  return api.post('/metadata/apply', {
    provider,
    id,
    paths,
    applyTags,
    applyCover,
    coverIndex,
  });
};

export default api;
```

### 7.2 Metadata Store

Create `frontend/src/stores/metadataStore.js`:

```javascript
import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import { searchMetadata, getMetadataDetails, applyMetadata } from '@/services/metadataService';

export const useMetadataStore = defineStore('metadata', () => {
  const searchResults = ref([]);
  const selectedRelease = ref(null);
  const isLoading = ref(false);
  const error = ref(null);
  const applyProgress = ref({ current: 0, total: 0, status: '' });

  const hasResults = computed(() => searchResults.value.length > 0);
  const isApplying = computed(() => applyProgress.value.total > 0);

  const search = async (query, providers = ['musicbrainz', 'discogs', 'freedb']) => {
    isLoading.value = true;
    error.value = null;
    searchResults.value = [];

    try {
      const response = await searchMetadata(query, { providers });
      searchResults.value = response.data.results;
    } catch (e) {
      error.value = e.response?.data?.error || 'Search failed';
    } finally {
      isLoading.value = false;
    }
  };

  const getDetails = async (provider, id) => {
    isLoading.value = true;
    error.value = null;

    try {
      const response = await getMetadataDetails(provider, id);
      selectedRelease.value = response.data;
      return response.data;
    } catch (e) {
      error.value = e.response?.data?.error || 'Failed to get details';
      return null;
    } finally {
      isLoading.value = false;
    }
  };

  const apply = async (provider, id, paths, options = {}) => {
    isLoading.value = true;
    error.value = null;
    applyProgress.value = { current: 0, total: paths.length, status: 'Starting...' };

    try {
      const response = await applyMetadata(provider, id, { ...options, paths });
      return response.data;
    } catch (e) {
      error.value = e.response?.data?.error || 'Failed to apply metadata';
      return null;
    } finally {
      isLoading.value = false;
      applyProgress.value = { current: 0, total: 0, status: '' };
    }
  };

  const clearResults = () => {
    searchResults.value = [];
    selectedRelease.value = null;
    error.value = null;
  };

  return {
    searchResults,
    selectedRelease,
    isLoading,
    error,
    applyProgress,
    hasResults,
    isApplying,
    search,
    getDetails,
    apply,
    clearResults,
  };
});
```

### 7.3 Metadata Search Component

Create `frontend/src/components/MetadataSearch.vue`:

```vue
<template>
  <div class="metadata-search">
    <!-- Search Header -->
    <div class="search-header">
      <h3>Search Metadata</h3>
      <p class="text-sm text-gray-500">
        Search MusicBrainz, Discogs, and FreeDB for album information
      </p>
    </div>

    <!-- Search Form -->
    <form @submit.prevent="handleSearch" class="search-form">
      <div class="form-row">
        <input
          v-model="searchQuery"
          type="text"
          placeholder="Search by artist, album, or track..."
          class="form-input"
          :disabled="store.isLoading"
        />
        <button
          type="submit"
          class="btn-primary"
          :disabled="store.isLoading || !searchQuery.trim()"
        >
          <span v-if="store.isLoading">Searching...</span>
          <span v-else>Search</span>
        </button>
      </div>

      <!-- Provider Filter -->
      <div class="provider-filter">
        <label v-for="provider in availableProviders" :key="provider.id">
          <input
            type="checkbox"
            v-model="selectedProviders"
            :value="provider.id"
            :disabled="store.isLoading"
          />
          {{ provider.name }}
        </label>
      </div>
    </form>

    <!-- Error Display -->
    <div v-if="store.error" class="error-message">
      {{ store.error }}
    </div>

    <!-- Results -->
    <div v-if="store.hasResults" class="results-section">
      <div class="results-header">
        <h4>Search Results ({{ store.searchResults.length }})</h4>
        <button @click="store.clearResults" class="btn-text">Clear</button>
      </div>

      <div class="results-list">
        <MetadataResultItem
          v-for="result in store.searchResults"
          :key="`${result.provider}-${result.id}`"
          :result="result"
          @select="handleSelect"
        />
      </div>
    </div>

    <!-- Loading State -->
    <div v-if="store.isLoading && !store.hasResults" class="loading-state">
      <div class="spinner"></div>
      <p>Searching metadata databases...</p>
    </div>

    <!-- Empty State -->
    <EmptyState
      v-if="!store.isLoading && !store.hasResults && hasSearched"
      title="No results found"
      message="Try adjusting your search query or different providers"
    />
  </div>
</template>

<script setup>
import { ref, computed } from 'vue';
import { useMetadataStore } from '@/stores/metadataStore';
import MetadataResultItem from './MetadataResultItem.vue';
import EmptyState from './EmptyState.vue';

const store = useMetadataStore();

const searchQuery = ref('');
const selectedProviders = ref(['musicbrainz', 'discogs', 'freedb']);
const hasSearched = ref(false);

const availableProviders = [
  { id: 'musicbrainz', name: 'MusicBrainz' },
  { id: 'discogs', name: 'Discogs' },
  { id: 'freedb', name: 'FreeDB' },
  { id: 'albumart', name: 'AlbumArt.digital' },
];

const handleSearch = async () => {
  if (!searchQuery.value.trim()) return;
  hasSearched.value = true;
  await store.search(searchQuery.value, selectedProviders.value);
};

const handleSelect = (result) => {
  emit('select', result);
};

const emit = defineEmits(['select']);
</script>
```

### 7.4 Result Item Component

Create `frontend/src/components/MetadataResultItem.vue`:

```vue
<template>
  <div
    class="result-item"
    :class="{ selected: isSelected }"
    @click="$emit('select', props.result)"
  >
    <div class="result-cover">
      <img
        v-if="props.result.coverUrl"
        :src="props.result.coverUrl"
        :alt="props.result.title"
      />
      <div v-else class="no-cover">
        <MusicIcon />
      </div>
    </div>

    <div class="result-info">
      <h4 class="result-title">{{ props.result.title }}</h4>
      <p class="result-artist">{{ props.result.artist }}</p>
      <p class="result-year" v-if="props.result.year">{{ props.result.year }}</p>
      <p class="result-provider" :class="`provider-${props.result.provider}`">
        {{ props.result.provider }}
      </p>
    </div>

    <div class="result-actions">
      <button
        v-if="props.result.trackCount"
        class="btn-sm"
        @click.stop="$emit('details', props.result)"
      >
        View Details
      </button>
    </div>
  </div>
</template>

<script setup>
import { MusicIcon } from './icons';

const props = defineProps({
  result: {
    type: Object,
    required: true,
  },
  isSelected: {
    type: Boolean,
    default: false,
  },
});

defineEmits(['select', 'details']);
</script>
```

### 7.5 Details Modal Component

Create `frontend/src/components/MetadataDetailsModal.vue`:

```vue
<template>
  <BaseModal :open="open" @close="$emit('close')" size="lg">
    <template #title>Apply Metadata</template>

    <div class="details-content" v-if="release">
      <div class="details-header">
        <img
          :src="release.coverUrl || '/img/placeholder-album.png'"
          class="cover-large"
        />
        <div class="details-info">
          <h3>{{ release.title }}</h3>
          <p class="artist">{{ release.artist }}</p>
          <p class="year" v-if="release.year">{{ release.year }}</p>
          <p class="genre" v-if="release.genre">{{ release.genre }}</p>
          <p class="track-count">{{ release.tracks?.length || 0 }} tracks</p>
        </div>
      </div>

      <!-- Cover Art Selection -->
      <div v-if="release.coverOptions?.length > 1" class="cover-selection">
        <h4>Select Cover Art</h4>
        <div class="cover-grid">
          <img
            v-for="(cover, idx) in release.coverOptions"
            :key="idx"
            :src="cover.url"
            :class="{ selected: selectedCoverIndex === idx }"
            @click="selectedCoverIndex = idx"
          />
        </div>
      </div>

      <!-- Track List -->
      <div class="track-list">
        <h4>Track List</h4>
        <div
          v-for="(track, idx) in release.tracks"
          :key="idx"
          class="track-item"
        >
          <span class="track-number">{{ track.number }}</span>
          <span class="track-title">{{ track.title }}</span>
          <span class="track-duration">{{ track.duration }}</span>
        </div>
      </div>
    </div>

    <template #footer>
      <button @click="$emit('close')" class="btn-secondary">Cancel</button>
      <button
        @click="handleApply"
        class="btn-primary"
        :disabled="store.isApplying"
      >
        {{ store.isApplying ? 'Applying...' : 'Apply Metadata' }}
      </button>
    </template>
  </BaseModal>
</template>

<script setup>
import { ref, watch } from 'vue';
import { useMetadataStore } from '@/stores/metadataStore';
import BaseModal from './BaseModal.vue';

const props = defineProps({
  open: Boolean,
  release: Object,
  paths: {
    type: Array,
    default: () => [],
  },
});

const emit = defineEmits(['close', 'applied']);

const store = useMetadataStore();
const selectedCoverIndex = ref(0);

watch(() => props.release, () => {
  selectedCoverIndex.value = 0;
});

const handleApply = async () => {
  if (!props.release) return;

  const result = await store.apply(
    props.release.provider,
    props.release.id,
    props.paths,
    {
      applyCover: true,
      coverIndex: selectedCoverIndex.value,
    }
  );

  if (result) {
    emit('applied', result);
    emit('close');
  }
};
</script>
```

### 7.6 Integration with Album Detail View

Update `frontend/src/views/AlbumDetailView.vue` to add metadata search button:

```vue
<template>
  <div class="album-detail">
    <!-- Existing album info... -->

    <!-- Metadata Actions -->
    <div class="metadata-actions">
      <button @click="showMetadataSearch = true" class="btn-secondary">
        <SearchIcon /> Find Metadata
      </button>
    </div>

    <!-- Metadata Search Modal -->
    <MetadataSearch
      v-model:open="showMetadataSearch"
      @select="handleMetadataSelect"
    />

    <!-- Details Modal -->
    <MetadataDetailsModal
      v-model:open="showDetailsModal"
      :release="selectedRelease"
      :paths="selectedPaths"
      @applied="handleMetadataApplied"
    />
  </div>
</template>

<script setup>
import { ref } from 'vue';
import MetadataSearch from '@/components/MetadataSearch.vue';
import MetadataDetailsModal from '@/components/MetadataDetailsModal.vue';
import { useAlbumStore } from '@/stores/albumStore';
import { useMetadataStore } from '@/stores/metadataStore';

const albumStore = useAlbumStore();
const metadataStore = useMetadataStore();

const showMetadataSearch = ref(false);
const showDetailsModal = ref(false);
const selectedRelease = ref(null);
const selectedPaths = ref([]);

const handleMetadataSelect = async (result) => {
  selectedRelease.value = await metadataStore.getDetails(result.provider, result.id);
  selectedPaths.value = albumStore.currentAlbum?.tracks?.map(t => t.path) || [];
  showDetailsModal.value = true;
};

const handleMetadataApplied = () => {
  // Refresh album data
  albumStore.fetchAlbum(albumStore.currentAlbum.path);
};
</script>
```

### 7.7 Route Integration

Add route to `frontend/src/router/index.js`:

```javascript
{
  path: '/metadata-search',
  name: 'MetadataSearch',
  component: () => import('@/views/MetadataSearchView.vue'),
  meta: { title: 'Search Metadata' },
},
```

Create `frontend/src/views/MetadataSearchView.vue`:

```vue
<template>
  <div class="metadata-search-view">
    <header class="page-header">
      <h1>Search Metadata</h1>
      <p>Find and apply metadata from online music databases</p>
    </header>

    <MetadataSearch @select="handleSelect" />
  </div>
</template>

<script setup>
import MetadataSearch from '@/components/MetadataSearch.vue';
import { useMetadataStore } from '@/stores/metadataStore';
import MetadataDetailsModal from '@/components/MetadataDetailsModal.vue';
import { ref } from 'vue';

const store = useMetadataStore();
const selectedRelease = ref(null);
const showDetailsModal = ref(false);

const handleSelect = async (result) => {
  selectedRelease.value = await store.getDetails(result.provider, result.id);
  showDetailsModal.value = true;
};
</script>
```

## Phase 8: Configuration

Update `backend/config.json` with metadata provider settings:

```json
{
  "metadata": {
    "providers": {
      "musicbrainz": {
        "enabled": true,
        "baseUrl": "https://musicbrainz.org/ws/2",
        "rateLimit": 1,
        "userAgent": "mpd-client-modern/1.0"
      },
      "discogs": {
        "enabled": true,
        "baseUrl": "https://api.discogs.com",
        "token": "${DISCOGS_TOKEN}"
      },
      "freedb": {
        "enabled": true,
        "baseUrl": "https://freedb.freedb.org/~cddb/cddb.cgi"
      },
      "albumart": {
        "enabled": true,
        "baseUrl": "https://api.albumart.digital"
      }
    },
    "cache": {
      "ttl": 86400,
      "maxSize": 1000
    }
  }
}
```

## File Changes Summary

| File | Action |
|------|--------|
| `frontend/src/services/metadataService.js` | Create |
| `frontend/src/stores/metadataStore.js` | Create |
| `frontend/src/components/MetadataSearch.vue` | Create |
| `frontend/src/components/MetadataResultItem.vue` | Create |
| `frontend/src/components/MetadataDetailsModal.vue` | Create |
| `frontend/src/views/MetadataSearchView.vue` | Create |
| `frontend/src/router/index.js` | Update |
| `frontend/src/views/AlbumDetailView.vue` | Update |
| `backend/config.json` | Update |
| `docs/API_Documentation.md` | Update |

## Next Steps

1. Create the API service and store (high priority)
2. Build the search component UI
3. Build the details modal with cover selection
4. Integrate with album detail view
5. Add configuration options
6. Test end-to-end functionality
