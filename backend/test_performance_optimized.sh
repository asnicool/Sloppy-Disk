#!/bin/bash

echo "Testing Random Albums Performance..."
start_time=$(date +%s.%N)
curl -s "http://localhost:7070/api/albums/random?count=36&refresh=true" > /dev/null
end_time=$(date +%s.%N)
duration=$(echo "$end_time - $start_time" | bc)

echo "Request took: $duration seconds"

if (( $(echo "$duration < 1.0" | bc -l) )); then
  echo "PASS: Random albums loaded in under 1 second."
else
  echo "FAIL: Random albums took too long ($duration s)."
fi
