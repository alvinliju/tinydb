# TinyDB

**Radically simple distributed storage. Fast, minimal, hackable.**

Built for hackers who want to understand how distributed systems actually work—without the enterprise bullshit.

## What it does

* **HTTP API** → PUT/GET/DELETE files 
* **Quorum everything** → 2/3 replicas for reads + writes
* **SHA256 keys** → Content-addressable, deduplication 
* **Fault tolerant** → Works with 1 replica down
* **Zero-copy redirects** → No bottlenecks, no proxying
* **<1500 lines of Go** → Read the source, hack it

## Quick start

```bash
# Fire up the cluster
cd cmd/master && go run main.go &
cd cmd/volume && go run main.go 3001 &
cd cmd/volume && go run main.go 3002 &
cd cmd/volume && go run main.go 3003 &

# Or use the script
./scripts/bash-scripts/start_volumes.sh

# Store shit
curl -X PUT localhost:3000/hack.txt -d "hello world"

# Get shit back
curl localhost:3000/hack.txt

# Delete shit
curl -X DELETE localhost:3000/hack.txt
```

## How it works

```
Client → Master → [Volume1, Volume2, Volume3]
```

* **Master**: Routes requests, enforces quorum, redirects to healthy replicas
* **Volumes**: Store actual files, keyed by SHA256
* **Quorum**: Need 2/3 replicas for any operation

Kill a volume server. Watch it keep working. That's distributed systems.

## Why?

Because most "distributed storage" is either:
1. A black box you can't understand
2. 50,000 lines of enterprise Java 
3. Both

TinyDB is 1500 lines you can read in an hour. Fork it. Break it. Learn from it.

## Status

🚧 **Working prototype. Perfect for learning, hacking, tinkering.**

*Not production ready. Don't store your crypto keys here.*
