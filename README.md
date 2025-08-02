# TinyDB v0.3

**Radically simple distributed storage. Fast, minimal, hackable.**

Built for hackers who want to understand how distributed systems actually work—without the enterprise bullshit.

## What it does

* **HTTP API** → PUT/GET files with automatic replication
* **Content-addressable** → SHA256 keys, deduplication built-in
* **3x replication** → Fault tolerant, works with 1 replica down
* **LevelDB metadata** → Fast lookups, persistent storage
* **Zero-copy redirects** → No bottlenecks, no proxying
* **nginx volume servers** → Battle-tested, production-ready storage

## Architecture

```
Client → Master (LevelDB) → [nginx:3001, nginx:3002, nginx:3003]
```

* **Master**: Routes requests, enforces quorum, stores metadata in LevelDB
* **Volumes**: nginx servers storing actual files, keyed by SHA256
* **Replication**: 3x replication for fault tolerance

## Performance

**Tested on recycled laptop running Alpine Linux:**

| File Size | RPS | Avg Latency | Success Rate | Status |
|-----------|-----|-------------|--------------|---------|
| Small (<1MB) | 6,696 | 25ms | 100% | ✅ Blazing fast |
| Large (100MB) | 21 | 769ms | 100% | ✅ Solid |

**6.7K RPS for small files puts us in the same league as production systems.**

## Quick start

```bash
# Start volume servers (nginx)
cd cmd/volume && ./volume.sh 3001 &
cd cmd/volume && ./volume.sh 3002 &
cd cmd/volume && ./volume.sh 3003 &

# Start master server
cd cmd/master && go run main.go &

# Store shit
curl -X PUT localhost:8080/hack.txt -d "hello world"

# Get shit back
curl localhost:8080/hack.txt

# Check metadata
ls /tmp/leveldb/  # See your LevelDB files
```

## How it works

1. **PUT request** → Master hashes content, stores in 3 volume servers
2. **GET request** → Master looks up metadata, redirects to healthy replica
3. **Fault tolerance** → Works with 1 replica down(currently working on it)
4. **Content addressing** → Same content = same key, automatic deduplication

## Why nginx?

* **Battle-tested** → nginx handles millions of requests
* **Zero config** → Just works
* **Production ready** → Used by 40% of the internet
* **Simple** → No custom storage layer bullshit

## Code structure

```
cmd/
├── master/          # Master server with LevelDB metadata
│   └── main.go     # HTTP API + replication logic
└── volume/          # nginx volume servers
    ├── volume.sh    # Startup script
    └── nginx.conf   # nginx config
```

**<1500 lines of Go. Read the source, hack it.**

## Status

 *Hackable, Not PROD ready...but still works..*

*Fault tolerant, fast, minimal. Everything you need, nothing you don't.*

## About

Built by [@alvinliju](https://github.com/alvinliju) because most "distributed storage" is either:
1. A black box you can't understand
2. 50,000 lines of enterprise Java
3. Both


