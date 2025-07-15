#!/bin/bash

# Path to your Go source files
VOLUME_SERVER_DIR="./cmd/volume"   # Update if your volume server is in a different folder
MASTER_SERVER_DIR="./cmd/master"   # Update if your master server is in a different folder

# Volume server ports
VOLUME_PORTS=(3001 3002 3003)

# Start volume servers
for PORT in "${VOLUME_PORTS[@]}"; do
  echo "Starting volume server on port $PORT..."
  sudo go run "$VOLUME_SERVER_DIR/main.go" "$PORT" > "volume_$PORT.log" 2>&1 &
  echo $! > "volume_$PORT.pid"
done

# Start master server (assumed to run on port 3000)
echo "Starting master server on port 3000..."
sudo go run "$MASTER_SERVER_DIR/main.go" > "master.log" 2>&1 &
echo $! > "master.pid"

echo "✅ All servers started."
