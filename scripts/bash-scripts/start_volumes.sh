#!/bin/bash
# start_volumes.sh
 ls
for port in {3001..3003}; do
    echo "Starting volume server on port $port"
    cd .. && cd ..&& go run cmd/volume/main.go $port &
done

wait
