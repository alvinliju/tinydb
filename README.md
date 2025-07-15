# TinyDB

Distributed key-value store in <1000 LOC. Fast, minimal, no bullshit.

## What it does
- PUT/GET/DELETE over HTTP
- 3-replica writes with fault tolerance
- Content-addressable storage (SHA256 keys)
- 2.5GB/sec throughput on single volume

## API
```bash
# Upload (returns hash)
curl -X PUT localhost:3000/myfile --data-binary @file.txt

# Download
curl localhost:3000/myfile

# Delete
curl -X DELETE localhost:3000/myfile
```

## Performance
- **PUT 1GB**: 474 MB/s write, 9KB memory usage
- **GET 1GB**: 1.25 GB/s read, zero-copy serving
- **Load test**: 100 concurrent × 1GB = 5GB/s aggregate

## Run it
```bash
# Master
go run master.go

# Volume servers
go run volume.go -port 3001
go run volume.go -port 3002
go run volume.go -port 3003
```

## Architecture
```
Client -> Master -> 3 Volume Servers (LevelDB)
```

Master tracks replicas. Volumes store files. Simple.

Built to learn distributed systems. Inspired by minikeyvalue.

**Status: Working prototype. Ships fast, breaks things.**
