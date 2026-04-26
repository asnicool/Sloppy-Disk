cd frontend
npx vite build
cd ../backend
go build -o sloppy-disk-backend cmd/server/main.go
cd ..
# Sync the directory to remote location, excluding config.json
rsync -av --exclude='config.json' /opt/mpd-client-modern/ m8box.local:/opt/mpd-client-modern/
# Optionally launch the server after sync
#cd backend && ./mpd-client-backend
ssh n@m8box.local -x "sudo service mcm restart"

