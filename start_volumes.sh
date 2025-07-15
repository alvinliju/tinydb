#!/bin/bash
# start_volumes.sh

for port in {3001..3012}; do
    echo "Starting volume server on port $port"
    sudo go run cmd/volume/main.go $port &
done

wait
