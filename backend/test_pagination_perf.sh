#!/bin/bash

BASE_URL="http://localhost:7070/api/albums"

echo "Step 1: Fetch Page 0 (Expect Fast Basic Response)"
start_time=$(date +%s.%N)
# We use -o /dev/null but maybe we should grep for something to confirm it's basic?
# Basic albums won't have 'trackCount' populated from cache.go GetAlbumsPage (it returns raw structs from Refresh).
# Wait, Refresh populates ID, Album, Artist, Date, Genre. TrackCount is 0.
response=$(curl -s "$BASE_URL?page=0&limit=50&sort=name")
end_time=$(date +%s.%N)
duration=$(echo "$end_time - $start_time" | bc)
echo "Page 0 took: $duration seconds"

# Check if response contains many items
count=$(echo "$response" | grep -o '"id":' | wc -l)
echo "Page 0 item count: $count"

if (( $(echo "$duration < 1.0" | bc -l) )); then
  echo "PASS: Page 0 loaded instantly."
else
  echo "FAIL: Page 0 slow ($duration s)."
fi

echo "Waiting 5 seconds for background enrichment..."
sleep 5

echo "Step 2: Fetch Page 1 (Expect Fast ENRICHED Response from Cache)"
start_time=$(date +%s.%N)
response=$(curl -s "$BASE_URL?page=1&limit=50&sort=name")
end_time=$(date +%s.%N)
duration=$(echo "$end_time - $start_time" | bc)
echo "Page 1 took: $duration seconds"

# Check if enriched (look for "coverUrl")
# Basic albums have coverUrl="" unless enriched.
# Wait, cache.go Refresh DOES NOT populate coverUrl. EnrichAlbums does.
# So if we see coverUrl filled (contains "/api/coverart/"), it's enriched.
header_check=$(echo "$response" | grep -o "/api/coverart/" | wc -l)
echo "Page 1 enriched items (approx): $header_check"

if (( $(echo "$duration < 1.0" | bc -l) )); then
  echo "PASS: Page 1 loaded instantly."
else
  echo "FAIL: Page 1 slow ($duration s)."
fi

if [ "$header_check" -gt 0 ]; then
     echo "PASS: Page 1 data is enriched."
else
     echo "FAIL: Page 1 data is NOT enriched (Cache Miss?)."
fi
