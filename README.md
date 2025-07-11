# tinyDB

A fast, minimal, and distributed key-value store (<1000 LOC) optimized for reading and writing files between 1MB-1GB.

## Features
- Simple HTTP API (GET/PUT/DELETE)
- Hierarchical storage structure
- SHA-256 key hashing for efficient file naming
- Single binary deployment for ease of use

## API
- **GET /key**: Fetches a file. On the master server, this returns a 302 redirect to the appropriate volume server. Supports range requests for efficient streaming.
- **PUT /key**: Uploads a file. The operation blocks until the file is fully written to disk on a volume server, returning 201 (Created) on success.
- **DELETE /key**: Deletes a file. The operation blocks until the file is fully removed, returning 204 (No Content) on success.

## Performance

`tinyDB` is specifically designed for high memory efficiency and optimal throughput when handling large files (1MB-1GB). Benchmarks and real-world load tests confirm its ability to manage these files with minimal application-level memory consumption.

### Individual Operation Benchmarks (from `go test -bench`):

These benchmarks were conducted on a single volume server.

#### PUT 1GB File (`BenchmarkHandlePut1GB`)
- **Test:** PUT a 1GB file (example key: `bench_put_1gb_key_99`)
- **Result:** `2,114,560,500 ns/op` (approximately **2.11 seconds per operation**). This translates to a write throughput of around **474 MB/s**, demonstrating excellent write speed, primarily limited by disk I/O.
- **Memory Allocation:** `9549 B/op` (approximately **9.32 KB per operation**). This is an **exceptionally low** memory footprint, confirming that the PUT operation efficiently streams data to disk without buffering the entire file in Go's memory.
- **Allocations:** `62 allocs/op`. A very low number, indicating minimal garbage collection overhead.

#### GET 1GB File (`BenchmarkHandleGet1GB`)
- **Test:** GET a 1GB file (example key: `a31ef9d64878f50eed6cbcbc214685e0f7ccfc763d4ebff6df7ecfe22b8d730e_bench_get_1gb_file`)
- **Result:** `801,000,863 ns/op` (approximately **0.80 seconds per operation**). This translates to a read throughput of about **1.25 GB/s** from the client's perspective in the benchmark.
- **Memory Allocation (with crucial context):** `2,147,495,016 B/op` (approximately **2.14 GB per operation**).
    - **Important Note:** While this `go test` benchmark reports high memory allocation, it's primarily an artifact of `httptest.NewRecorder` (used by the testing framework) internally buffering the entire 1GB response to allow for test verification.
    - **Real-World Confirmation:** When the `volume` server was run independently and subjected to real `GET` requests for 1GB files (monitored with `htop`), its **Resident Memory Size (RES) remained stable and low, showing no significant spikes (i.e., not consuming gigabytes of RAM per file)**. This confirms that `http.ServeFile` successfully leverages OS-level zero-copy (`sendfile` on macOS/Linux) for highly efficient, low-memory file serving in production scenarios.
- **Allocations:** `67 allocs/op`. A very low number, similar to PUT.

### Real-World GET Load Test (using `hey`):

A load test was conducted against a live `volume` server serving a 1GB file.

- **Test:** 100 concurrent connections (`-c 100`) making 100 GET requests each (`-n 100`) for a 1GB file.
- **Total Data Transferred:** 100 GB (100 requests * 1 GB/request)
- **Total Test Duration:** ~20.0 seconds
- **Average Request Time:** ~16.18 seconds per 1GB request.
- **Aggregate Throughput:** ~5 GB/s.
- **Memory (htop):** During this intense load, the `volume` server's actual RAM usage (monitored via `htop`) remained stable and low, demonstrating `tinyDB`'s ability to handle high concurrent traffic for large files without memory exhaustion.

**Conclusion:**

`tinyDB` excels in memory efficiency for both writing and reading large files, utilizing direct-to-disk and zero-copy techniques where possible. Its performance is largely bottlenecked by the underlying disk I/O and network bandwidth, not by Go application-level memory consumption.
**(it still in beta so the test result may vary and its not really optimized, this version is just a working prototype)**

```sh
# Master server
go run master.go -port 3000

# Volume server
go run volume.go -port 3001

Inspired by minikeyvalue. Built to learn distributed systems fundamentals.
