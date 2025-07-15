#!/bin/bash

# Kill master server
if [ -f master.pid ]; then
  kill $(cat master.pid) && echo "Stopped master server"
  rm master.pid
fi

# Kill volume servers
for PIDFILE in volume_*.pid; do
  if [ -f "$PIDFILE" ]; then
    kill $(cat "$PIDFILE") && echo "Stopped volume server from $PIDFILE"
    rm "$PIDFILE"
  fi
done

echo "🛑 All servers stopped."
