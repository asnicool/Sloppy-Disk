cd frontend
npx vite build
cd ../backend
go build -o server cmd/server/main.go
cd ..
# Sync the directory to remote location, excluding config.json
rsync -av --exclude='config.json' /opt/mpd-client-modern/ m8box.local:/opt/mpd-client-modern/
# Optionally launch the server after sync
cd backend && ./server

