#!/bin/bash

BASE_URL="http://localhost:7070/api/albums"

echo "=== Testing MPD Album Backend Performance ==="

echo -e "\n1. Listing Albums (Page 0, Limit 50)"
curl -w "\nTime: %{time_total}s\n" -s "${BASE_URL}?page=0&limit=50" | jq '.meta'

echo -e "\n2. Pagination (Page 5, Limit 50)"
curl -w "\nTime: %{time_total}s\n" -s "${BASE_URL}?page=5&limit=50" | jq '.meta'

echo -e "\n3. Fuzzy Search 'greatest hits'"
curl -w "\nTime: %{time_total}s\n" -s "${BASE_URL}/search?q=greatest+hits" | jq '.meta'

echo -e "\n4. Fuzzy Search (No match)"
curl -w "\nTime: %{time_total}s\n" -s "${BASE_URL}/search?q=xyz123impossible" | jq '.meta'
