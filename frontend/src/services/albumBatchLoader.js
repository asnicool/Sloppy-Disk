import axios from 'axios'

class AlbumBatchLoader {
  constructor() {
    this.queue = []
    this.timer = null
    this.initialCount = 0
    this.MAX_INITIAL = 6
    this.BATCH_SIZE = 3  // Reduced from 6 to 3 for better responsiveness and to ensure status/playlist availability
    this.DEBOUNCE_MS = 100
    this.pendingRequests = new Map() // key -> {resolve, reject}
    this.isProcessing = false
  }

  requestDetails(artist, album) {
    const key = `${artist}|${album}`
    
    // If already in queue or processing, return existing promise
    if (this.pendingRequests.has(key)) {
      return this.pendingRequests.get(key).promise
    }

    let resolve, reject
    const promise = new Promise((res, rej) => {
      resolve = res
      reject = rej
    })

    const request = { artist, album, key, resolve, reject, promise }
    this.pendingRequests.set(key, request)

    // First N requests go out immediately
    if (this.initialCount < this.MAX_INITIAL) {
      this.initialCount++
      this.fetchIndividual(request)
    } else {
      this.enqueue(request)
    }

    return promise
  }

  async fetchIndividual(request) {
    try {
      const response = await axios.get(`/api/album/${encodeURIComponent(request.artist)}/${encodeURIComponent(request.album)}`)
      request.resolve(response.data)
    } catch (error) {
      if (error.response && error.response.status === 429) {
        // Server busy, put back in queue with delay
        setTimeout(() => this.enqueue(request), 1000 + Math.random() * 1000)
      } else {
        request.reject(error)
      }
    } finally {
      this.pendingRequests.delete(request.key)
    }
  }

  enqueue(request) {
    this.queue.push(request)
    if (!this.timer) {
      this.timer = setTimeout(() => this.processQueue(), this.DEBOUNCE_MS)
    }
  }

  async processQueue() {
    this.timer = null
    if (this.queue.length === 0) return

    const batch = this.queue.splice(0, this.BATCH_SIZE)
    const albumsToFetch = batch.map(r => ({ artist: r.artist, album: r.album }))

    try {
      const response = await axios.post('/api/albums/details/batch', { albums: albumsToFetch })
      const results = response.data.data // Map of "artist|album" -> details

      batch.forEach(request => {
        const result = results[request.key]
        if (result) {
          request.resolve({ success: true, data: result })
        } else {
          request.reject(new Error('Failed to fetch album details in batch'))
        }
        this.pendingRequests.delete(request.key)
      })
    } catch (error) {
      if (error.response && error.response.status === 429) {
        // Server busy, re-enqueue batch with backoff
        batch.forEach(request => {
          setTimeout(() => this.enqueue(request), 2000 + Math.random() * 2000)
        })
      } else {
        batch.forEach(request => {
          request.reject(error)
          this.pendingRequests.delete(request.key)
        })
      }
    }

    // Process next batch if any
    if (this.queue.length > 0) {
      this.timer = setTimeout(() => this.processQueue(), 50)
    }
  }
}

export const albumBatchLoader = new AlbumBatchLoader()
