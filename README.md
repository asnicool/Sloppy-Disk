# Sloppy Disk

A lightweight, modern MPD (Music Player Daemon) web client built with Go backend and Vue 3 frontend. Designed for low-memory systems with excellent mobile support and a native app-like experience.

## The Unique Feature: Random Album View

**Let chance decide what you listen to.**

Sloppy Disk's standout feature is its **Random Album View** - a deliberately anti-algorithmic approach to discovering your own music collection. Instead of AI-powered recommendations, popularity rankings, or "listeners also enjoyed" suggestions, Random Album View simply picks albums at random from your library. No machine learning, no collaborative filtering, no "because you listened to..." - just pure, unadulterated chance.

The idea is simple: you own your music, and sometimes the best way to rediscover it is to let randomness guide you. Hit the refresh button and get a fresh batch of albums you may have forgotten you had. It's the digital equivalent of flipping through your record collection with your eyes closed and pulling out whatever your hand lands on.

## A Confession: This Is Pure AI Vibe Coding

Let's be completely transparent: **I cannot code.** Like, at all. This entire application was built through AI-assisted "vibe coding" - the process of describing what you want to a large language model and hoping for the best. Every line of Go, every Vue component, every WebSocket handler - all generated through conversations with AI.

Why? Because I needed a music player that worked the way I wanted it to, and the existing options didn't quite fit. I had the ideas, the taste, and the stubbornness to keep iterating until the AI produced something that actually worked. The result is Sloppy Disk - a functional, decent-looking MPD client born entirely from the gap between "I know what I want" and "I don't know how to build it."

If you're a developer looking at this code and wincing - I get it. But it works, it's lightweight, and it does exactly what I need. Sometimes that's enough.

## Inspired By

This project was originally inspired by **[mpd-nodejs-client](https://github.com/kpillis/mpd-nodejs-client)** by **Krisztian Pillis** ([@kpillis](https://github.com/kpillis)). That project proved that a lightweight web interface for MPD was not only possible but genuinely useful. Sloppy Disk started as an attempt to modernize that concept with a Go backend and Vue 3 frontend, then grew into its own thing with features I needed (especially that random album picker).

Thank you, Krisztian, for showing the way.

## Open Source Packages Used

This project stands on the shoulders of giants. Here are the open source packages that make Sloppy Disk possible:

### Backend (Go)
- **[Go](https://github.com/golang/go)** - The programming language itself
- **[gorilla/mux](https://github.com/gorilla/mux)** - HTTP router and URL matcher
- **[gorilla/websocket](https://github.com/gorilla/websocket)** - WebSocket protocol implementation
- **[dhowden/tag](https://github.com/dhowden/tag)** - Audio tag reading (MP3, MP4, FLAC, OGG)
- **[michiwend/gomusicbrainz](https://github.com/michiwend/gomusicbrainz)** - MusicBrainz API client
- **[sahilm/fuzzy](https://github.com/sahilm/fuzzy)** - Fuzzy string matching
- **[tetratelabs/wazero](https://github.com/tetratelabs/wazero)** - Zero dependency WebAssembly runtime
- **[senan/taglib](https://github.com/senan-dev/taglib)** - TagLib bindings for Go (audio metadata)

### Frontend (Vue 3)
- **[Vue.js](https://github.com/vuejs/core)** - Reactive JavaScript framework
- **[Vue Router](https://github.com/vuejs/router)** - Official router for Vue.js
- **[Pinia](https://github.com/vuejs/pinia)** - State management for Vue.js
- **[Vite](https://github.com/vitejs/vite)** - Next generation frontend build tool
- **[Tailwind CSS](https://github.com/tailwindlabs/tailwindcss)** - Utility-first CSS framework
- **[Axios](https://github.com/axios/axios)** - Promise-based HTTP client
- **[Fuse.js](https://github.com/krisk/Fuse)** - Lightweight fuzzy-search library
- **[SortableJS](https://github.com/SortableJS/Sortable)** - Drag-and-drop reordering
- **[vuedraggable](https://github.com/SortableJS/Vue.Draggable)** - Vue component for SortableJS
- **[lodash-es](https://github.com/lodash/lodash)** - Modern JavaScript utility library

## Heartfelt Thanks

To the countless humans who made this possible:

- **The MPD team** - For building the rock-solid Music Player Daemon that has been serving music lovers for decades. Your work is the foundation everything here is built on.
- **The Go team** - For creating a language that's fast, simple, and efficient. Perfect for someone who needs the computer to do the heavy lifting.
- **The Vue.js team** - For a framework that makes reactive UIs almost understandable, even to someone who can't code.
- **Every open source maintainer listed above** - You built the tools, fixed the bugs, wrote the docs, and answered the issues. This project is 100% built on your unpaid labor and generosity.
- **Krisztian Pillis** - For the original mpd-nodejs-client that inspired this whole adventure.
- **The AI models** that patiently turned my vague descriptions into working code. You didn't judge my lack of coding skills, and I appreciate that.

## Quick Start

### Prerequisites
- Go 1.21+
- Node.js 18+
- MPD server running on localhost:6600

### Installation

1. **Install frontend dependencies:**
   ```bash
   cd frontend
   npm install
   ```

2. **Install backend dependencies:**
   ```bash
   cd backend
   go mod tidy
   ```

### Development

1. **Start backend:**
   ```bash
   cd backend
   go run cmd/server/main.go
   ```
   Backend runs on `http://localhost:7070`

2. **Start frontend (new terminal):**
   ```bash
   cd frontend
   npm run dev
   ```
   Frontend runs on `http://localhost:7071`

### Production Build

1. **Build frontend:**
   ```bash
   cd frontend
   npm run build
   ```

2. **Build backend:**
   ```bash
   cd backend
   go build -o sloppy-disk-backend cmd/server/main.go
   ```

3. **Deploy:**
   - Copy `frontend/dist/` to your web server
   - Run `./sloppy-disk-backend` with your MPD server
   - Configure web server to serve frontend and proxy API to backend

## Architecture

### Backend Structure
```
backend/
├── cmd/server/main.go       # Main application entry point
├── internal/
│   ├── api/                 # HTTP handlers
│   ├── mpd/                 # MPD client wrapper
│   ├── models/              # Data models
│   ├── config/              # Configuration management
│   ├── coverart/            # Cover art management
│   ├── metadata/            # Metadata providers (MusicBrainz, Discogs, GNUDB)
│   ├── albumcache/          # Album caching layer
│   ├── search/              # Search functionality
│   ├── sync/                # Sync operations
│   ├── tags/                # Tag reading/writing
│   ├── artistimage/         # Artist image management
│   └── n50/                 # N50 HIFI component integration
└── go.mod
```

### Frontend Structure
```
frontend/
├── src/
│   ├── components/          # Reusable Vue components
│   ├── views/               # Page components
│   ├── stores/              # Pinia state management
│   ├── composables/         # Vue composables
│   ├── services/            # Service layer
│   ├── utils/               # Helper functions
│   ├── router/              # Vue Router configuration
│   └── styles/              # CSS and styling
├── public/                  # Static assets (PWA, icons)
├── index.html               # Main HTML template
├── package.json             # Dependencies and scripts
└── vite.config.js           # Vite configuration
```

## API Endpoints

### Status
- `GET /api/status` - Current MPD status
- `WebSocket /ws` - Real-time status updates

### Browse Music
- `GET /api/albums?page=1&limit=50&search=artist:beatles` - Paginated album list
- `GET /api/albums/random?count=30` - Random album selection
- `GET /api/artists?page=1&limit=50` - Paginated artist list
- `GET /api/album/{artist}/{album}?page=1&limit=50` - Album songs

### Search
- `GET /api/search?q=bohemian&type=song&page=1&limit=30` - Unified search

### Playback Control
- `POST /api/play` - Start playback
- `POST /api/pause` - Pause playback
- `POST /api/next` - Next track
- `POST /api/previous` - Previous track
- `POST /api/volume/{0-100}` - Set volume

### Playlist
- `POST /api/playlist/add/{uri}` - Add song to playlist
- `POST /api/playlist/remove/{position}` - Remove from playlist

## Deployment

### Docker (Recommended)
```bash
docker-compose up --build
```

### Manual Deployment
1. Build both frontend and backend
2. Serve frontend with nginx/apache
3. Run backend binary
4. Configure reverse proxy

## License

MIT License - see LICENSE file for details

## Final Words

This project exists because open source software and AI made it possible for someone with zero coding skills to build exactly the tool they needed. If Sloppy Disk is useful to you, that's wonderful. If the code makes you cringe, well - now you know why. Either way, be kind to each other and support open source maintainers. They're the real heroes here.
