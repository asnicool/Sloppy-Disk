# MPD Web Client - Modern Rewrite

A lightweight, modern MPD (Music Player Daemon) web client built with Go backend and Vue 3 frontend. Designed for low-memory systems with excellent mobile support and native app-like experience.

## Features

### Backend (Go)
- **Memory Efficient**: Uses only 2-10MB memory baseline
- **MPD Integration**: Direct connection to MPD with smart connection pooling
- **Pagination**: Prevents MPD overload with intelligent data pagination
- **WebSocket Support**: Real-time status updates
- **RESTful API**: Clean API design with proper error handling
- **Single Binary**: Easy deployment without runtime dependencies

### Frontend (Vue 3)
- **Mobile-First**: Optimized for touch devices and small screens
- **PWA Support**: Installable web app with offline capabilities
- **Real-time Updates**: WebSocket connection for live MPD status
- **Responsive Design**: Works seamlessly on mobile, tablet, and desktop
- **Modern Vue 3**: Composition API, Pinia state management
- **Tailwind CSS**: Utility-first styling with custom design system

### Key Improvements Over Original
- ✅ **No Database**: Relies entirely on MPD database
- ✅ **Smart Pagination**: Prevents MPD connection issues
- ✅ **Modern Stack**: Go + Vue 3 instead of Node.js + Angular 1.5
- ✅ **Mobile Optimized**: Touch-friendly interface
- ✅ **PWA**: Can be installed like a native app
- ✅ **Real-time**: WebSocket for instant updates
- ✅ **Lightweight**: Minimal bundle size and memory usage

## Quick Start

### Prerequisites
- Go 1.21+
- Node.js 18+
- MPD server running on localhost:6600

### Installation

1. **Clone and setup:**
   ```bash
   cd mpd-client-modern
   ```

2. **Install frontend dependencies:**
   ```bash
   cd frontend
   npm install
   ```

3. **Install backend dependencies:**
   ```bash
   cd ../backend
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
   go build -o mpd-server cmd/server/main.go
   ```

3. **Deploy:**
   - Copy `frontend/dist/` to your web server
   - Run `./mpd-server` with your MPD server
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
│   └── middleware/          # CORS, logging, etc.
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
│   ├── utils/               # Helper functions
│   ├── router/              # Vue Router configuration
│   └── styles/              # CSS and styling
├── public/                  # Static assets
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

## Performance Targets

- **Initial Load**: < 2 seconds on 3G
- **API Response**: < 100ms for cached data
- **MPD Commands**: < 50ms end-to-end latency
- **Memory Usage**: < 10MB backend, < 5MB frontend
- **Bundle Size**: < 100KB total (gzipped)

## Mobile Optimization

### Touch-Friendly Interface
- Minimum 44px touch targets
- Swipe gestures for navigation
- Long-press context menus
- Pull-to-refresh functionality

### PWA Features
- Installable on home screen
- Offline functionality
- Background sync
- Push notifications

### Native App Experience
- Full-screen mode
- Status bar styling
- App-like navigation
- Smooth animations

## Configuration

### Backend Configuration
Edit `backend/cmd/server/main.go` to change:
- MPD host/port (default: localhost:6600)
- API port (default: 7070)
- Pagination limits
- CORS settings

### Frontend Configuration
Edit `frontend/vite.config.js` to change:
- API proxy settings
- PWA configuration
- Build optimization

## Deployment

### Docker (Recommended)
```bash
# Build and run with Docker Compose
docker-compose up --build
```

### Manual Deployment
1. Build both frontend and backend
2. Serve frontend with nginx/apache
3. Run backend binary
4. Configure reverse proxy

### Systemd Service
Create `/etc/systemd/system/mpd-client.service`:
```ini
[Unit]
Description=MPD Web Client
After=network.target

[Service]
Type=simple
User=mpd
WorkingDirectory=/opt/mpd-client
ExecStart=/opt/mpd-client/mpd-server
Restart=always

[Install]
WantedBy=multi-user.target
```

## Troubleshooting

### Connection Issues
- Verify MPD is running: `mpd status`
- Check firewall settings
- Ensure MPD allows TCP connections

### Performance Issues
- Adjust pagination limits in backend
- Enable MPD database updates
- Monitor MPD connection count

### Mobile Issues
- Enable HTTPS for PWA features
- Check viewport meta tags
- Test touch targets

## Contributing

1. Fork the repository
2. Create feature branch
3. Make changes with tests
4. Submit pull request

## License

MIT License - see LICENSE file for details

## Acknowledgments

- MPD team for the excellent music player daemon
- Vue.js team for the reactive framework
- Go team for the efficient programming language