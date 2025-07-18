# TinyDB

**TinyDB is a distributed, content-addressable key-value store designed to be fast, minimal, and radically simple.** Store files at high speed with just a few lines—built for hackers, tinkerers, and anyone exploring distributed storage.

## Features

* **PUT/GET/DELETE over HTTP**—store and retrieve files via simple REST API.
* **3-replica writes** with fault tolerance—distributed across volume servers.
* **Content-addressable storage** using SHA256 keys; enables deduplication and immutability.
* **Performance-focused:** 2.5GB/sec on a single volume in optimal conditions.
* <1000 lines of clean Go code.

## Quickstart

### Prerequisites

* Go 1.20+
* Linux/macOS/Windows
* (Optional) Docker support

### Run a Local Cluster

```bash
# Start the master
go run master.go

# Start some volume servers (use different ports)
go run volume.go -port 3001
go run volume.go -port 3002
go run volume.go -port 3003
```

### Example API Usage

```bash
# Upload a file (returns hash)
curl -X PUT localhost:3000/myfile --data-binary @file.txt

# Download a file
curl localhost:3000/myfile

# Delete a file
curl -X DELETE localhost:3000/myfile
```

## Architecture

```
[Client] <-> [Master] <-> [3x Volume Servers (LevelDB)]
```

* **Master**: Tracks where files are stored, routes requests, manages metadata in LevelDB.
* **Volume servers**: Store file contents on disk, keyed by SHA256.

**Inspired by minikeyvalue and the spirit of small, hackable infrastructure.**

## Performance

* **PUT 1GB:** ~474 MB/s, 9KB RAM usage per op
* **GET 1GB:** ~1.25 GB/s, zero-copy serving
* **Concurrent test:** 100×1GB = ~5GB/s aggregate (multi-volume)

## Why TinyDB?

TinyDB aims to:

* **Demystify distributed storage**—great for education, hacking, and prototyping.
* Serve as a minimal backbone for larger, S3-like object stores.
* Empower users to run their own distributed storage with full transparency and no unnecessary bloat.

## Status

🚧 **Working prototype. Breaking changes possible. Not production-ready—use for learning, testing, and fun!**

*Last updated: 2025-07-18*

## Contributing

Contributions, feedback, and issues are welcome!

* See CONTRIBUTING.md *(coming soon)*
* File an issue or open a pull request.

## License

MIT. See LICENSE for details.

*Contact & discussion: Open a GitHub issue or join our Discord (TBD).*
