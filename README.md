# tinyDB

A fast minimal distributed key-value store (<1000 LOC) optimized for reading files between 1MB-1GB.

## Features
- Simple HTTP API (GET/PUT/DELETE)
- Hierarchical storage structure
- SHA-256 key hashing
- Single binary deployment

## API
- GET /key - 302 redirect to volume server (supports range requests)
- PUT /key - Blocks until written (201 on success)
- DELETE /key - Blocks until deleted (204 on success)


## Run
```sh
# Master server
go run master.go -port 3000

# Volume server
go run volume.go -port 3001
```

Inspired by minikeyvalue. Built to learn distributed systems fundamentals.
