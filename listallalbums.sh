echo "list album group date group albumartist group artist group genre" | \
  nc m8box.local 6600 -w 1 |\
  awk -F ': ' '/Date:/ {date=$2} /Artist:/ {artist=$2} /Albumartist:/ {artist=$2} /Genre:/ {genre=$2} /Album:/ {print $2"|"date"|"artist"|"genre}' |\
  sort |uniq
