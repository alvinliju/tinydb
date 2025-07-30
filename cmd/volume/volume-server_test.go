package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var testStorageRoot string

func initTestStorage(tb testing.TB) {
	tempDir, err := os.MkdirTemp("", "tinydb_test_data_*")
	if err != nil {
		tb.Fatalf("Failed to create temp dir: %v", err)
	}
	testStorageRoot = tempDir
	// Overwrite the global storageRoot for tests
	oldStorageRoot := storageRoot
	storageRoot = testStorageRoot

	// Ensure the temp directory is cleaned up after the test finishes
	tb.Cleanup(func() {
		os.RemoveAll(testStorageRoot)
		// Restore the original storageRoot if needed (important for benchmarks if they run after tests)
		storageRoot = oldStorageRoot
	})
}

func calculateExpectedFileName(key string) string {
	h := sha256.New()
	h.Write([]byte(key))
	hashString := hex.EncodeToString(h.Sum(nil))
	return fmt.Sprintf("%s_%s", hashString, key)
}

// first unit test and we need to pass t
func TestHandlePut(t *testing.T) {
	initTestStorage(t)

	//write the functionality in reverse
	testKey := "my_test_file.txt"
	test_content := "This is some sample content for the file. It's quite long to simulate a small blob."
	bodyBytes := []byte(test_content)

	//create mock HTTP req
	req := httptest.NewRequest("PUT", "/files/"+testKey, bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/octet-stream")

	//record handlers output
	rr := httptest.NewRecorder()

	//call your handler function directly
	fileHandler(rr, req)

	//checking http status codes and what happeend
	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v. Response body: %s",
			status, http.StatusCreated, rr.Body.String())
	}

	if contentType := rr.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("handler returned wrong content-type: got %q want %q", contentType, "application/json")
	}

	//capture json res
	var resp struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("could not unmarshal response JSON: %v. Body: %s", err, rr.Body.String())
	}

	//assertion on filename(important part)
	expectedFileName := calculateExpectedFileName(testKey)
	if resp.Key != expectedFileName {
		t.Errorf("response JSON key mismatch: got %q want %q", resp.Key, expectedFileName)
	}

	expectedFullPath, err := getFilePath(testKey)
	if err != nil {
		t.Fatalf("error getting expected file path: %v", err)
	}

	// Check if the file actually exists
	if _, err := os.Stat(expectedFullPath); os.IsNotExist(err) {
		t.Errorf("file was not created at %q", expectedFullPath)
	}

	createdFileContent, err := os.ReadFile(expectedFullPath)
	if err != nil {
		t.Fatalf("failed to read created file at %q: %v", expectedFullPath, err)
	}
	if string(createdFileContent) != test_content {
		t.Errorf("file content mismatch. Got %q, want %q", string(createdFileContent), test_content)
	}

}

func BenchmarkHandleGet1GB(b *testing.B) {
	// 1. Setup: Prepare the environment and the test file.
	// This ensures each benchmark run starts with a clean, temporary storage.
	initTestStorage(b)

	// Define the size of the file we want to GET.
	contentSize := 1024 * 1024 * 1024 // 1GB

	// Teach: We don't need to generate 1GB of random data in memory here.
	// We'll create a sparse file (or a file with repeating data) directly on disk.
	// This is more efficient for setup if the exact content doesn't matter,
	// and accurately reflects the "GET" scenario where the file already exists.

	// Create a dummy content slice for calculating the hash and getting length.
	// The actual file on disk can be sparse or have simpler content to speed up creation.
	dummyContent := make([]byte, contentSize)
	// For actual disk I/O, filling with data ensures non-sparse file and real read.
	// if _, err := rand.Read(dummyContent); err != nil {
	//     b.Fatalf("Failed to generate random content: %v", err)
	// }

	// Define the logical key for the file.
	testKey := "bench_get_1gb_file"
	// Calculate its expected hashed filename on disk.
	expectedHashedFilename := calculateExpectedFileName(testKey)
	// Get the full file path where it should be stored.
	filePath, err := getFilePath(testKey)
	if err != nil {
		b.Fatalf("getFilePath error during setup: %v", err)
	}

	// Teach: Crucial for GET benchmarks: The file MUST exist BEFORE the benchmark starts.
	// Create the necessary directory structure.
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		b.Fatalf("Failed to create test directory %s: %v", filepath.Dir(filePath), err)
	}
	// Write the 1GB file to disk *before* the benchmark timing begins.
	// Use dummyContent here to write the file, or generate it on the fly if rand.Read is slow.
	if err := os.WriteFile(filePath, dummyContent, 0644); err != nil {
		b.Fatalf("Failed to pre-write 1GB file for GET benchmark setup: %v", err)
	}
	// Now, the 1GB file is ready on disk for every iteration of the benchmark.

	// 2. Reset Timer:
	// This tells the benchmark to start measuring time from this point.
	// All the file creation and setup above is excluded from the benchmark results.
	b.ResetTimer()

	// 3. Benchmark Loop:
	// The code inside this loop is what will be repeatedly executed and measured.
	for i := 0; i < b.N; i++ {
		// Create a new GET request for the pre-existing 1GB file.
		// `expectedHashedFilename` is used in the URL to directly target the file.
		req := httptest.NewRequest("GET", "/files/"+expectedHashedFilename, nil)
		// `httptest.NewRecorder` captures the response from your handler.
		rr := httptest.NewRecorder()

		// Call your main HTTP handler. This is the core operation being benchmarked.
		fileHandler(rr, req)

		// Teach: Error checking within a benchmark loop should be minimal but essential.
		// If the handler returns an error status, it invalidates the benchmark run.
		if rr.Code != http.StatusOK {
			b.Fatalf("Benchmark failed: unexpected status %v for key %s. Body: %s", rr.Code, testKey, rr.Body.String())
		}

		// Teach: VERY IMPORTANT for GET benchmarks!
		// You MUST read the entire response body to simulate a client receiving the data.
		// If you don't read it, Go's compiler might optimize away the sending of the file,
		// giving you artificially low times and memory allocations that don't reflect real network I/O.
		// `io.Discard` is an `io.Writer` that just throws away any data written to it,
		// so it consumes the response without storing it in memory unnecessarily in the test.
		_, err := io.Copy(io.Discard, rr.Result().Body)
		if err != nil && err != io.EOF { // io.EOF is expected if the body is fully read
			b.Fatalf("Failed to read response body in benchmark: %v", err)
		}
	}

	// 4. Report Allocations:
	// Tells the Go testing framework to include memory allocation statistics (Bytes/op and allocs/op)
	// in the benchmark results. This is crucial for evaluating memory efficiency.
	b.ReportAllocs()
}

func TestHandleGet(t *testing.T) {
	initTestStorage(t)
	//write the functionality in reverse
	//keep doint this
	testKey := "my_test_file.txt"
	test_content := "This is some sample content for the file. It's quite long to simulate a small blob."
	bodyBytes := []byte(test_content)

	//to make this handler work lets break down the actual function
	//this get function takes in a key and checks if its there or not thats it and it forward it to volume shit right?
	//so we need to put a file in and retrive it using get request thats all

	//assertion on filename(important part)
	expectedFileName := calculateExpectedFileName(testKey)
	filePath, err := getFilePath(testKey)
	if err != nil {
		t.Fatalf("Failed to get file path for test setup: %v", err)
	}

	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		t.Fatalf("Failed to create test directory %s: %v", filepath.Dir(filePath), err)
	}

	// Write the dummy content to the calculated file path
	if err := os.WriteFile(filePath, bodyBytes, 0644); err != nil {
		t.Fatalf("Failed to write test file %s for GET: %v", filePath, err)
	}

	req := httptest.NewRequest("GET", "http://localhost:3001/files/"+expectedFileName, nil)

	//record handlers output
	rr := httptest.NewRecorder()

	//now use your get handler to get the file
	handleGet(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("GET handler returned wrong status code: got %v want %v. Response body: %s",
			status, http.StatusOK, rr.Body.String())
		return
	}

	retrievedContent, err := io.ReadAll(rr.Result().Body)
	if err != nil {
		t.Fatalf("Failed to read GET response body: %v", err)
	}

	if string(retrievedContent) != test_content {
		t.Errorf("retrieved content mismatch.\nGot: %q\nWant: %q", string(retrievedContent), test_content)
	}

	ext := filepath.Ext(filePath)
	expectedContent := mime.TypeByExtension(ext)

	// Teach: Assert on important headers for static file serving
	if contentType := rr.Header().Get("Content-Type"); !strings.HasPrefix(contentType, expectedContent) {
		t.Errorf("handler returned unexpected Content-Type: got %q want starts with %q", contentType, "video/mp4")
	}
	if acceptRanges := rr.Header().Get("Accept-Ranges"); acceptRanges != "bytes" {
		t.Errorf("handler returned unexpected Accept-Ranges: got %q want %q", acceptRanges, "bytes")
	}
	if contentLength := rr.Header().Get("Content-Length"); contentLength != fmt.Sprintf("%d", len(bodyBytes)) {
		t.Errorf("handler returned unexpected Content-Length: got %q want %d", contentLength, len(bodyBytes))
	}

}

func BenchmarkHandlePut1GB(b *testing.B) {
	// 1. Setup: Prepare the environment for the benchmark.
	// This uses your initTestStorage helper, which creates a temporary directory.
	initTestStorage(b)

	// Define the size and content for the file.
	contentSize := 1024 * 1024 * 1024 // 1GB
	content := make([]byte, contentSize)
	// Optional: Fill with random data for more realistic I/O patterns.
	// For actual performance, using random data is better than all zeros.
	// if _, err := rand.Read(content); err != nil {
	//     b.Fatalf("Failed to generate random content: %v", err)
	// }

	// 2. Reset Timer: Crucial step!
	// This discards any time spent in the setup phase (like creating the 1GB content).
	// The benchmark will only measure the code within the loop after this call.
	b.ResetTimer()

	// 3. Benchmark Loop: The core of the benchmark.
	// `b.N` is dynamically determined by the Go testing framework.
	// It runs your code enough times to get a stable measurement, aiming for at least 1 second.
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("bench_put_1gb_key_%d", i) // Create a unique key for each iteration
		bodyReader := bytes.NewReader(content)        // Create a new reader for each request

		// Simulate the HTTP request (PUT)
		req := httptest.NewRequest("PUT", "/files/"+key, bodyReader)
		req.Header.Set("Content-Type", "application/octet-stream")

		rr := httptest.NewRecorder()
		fileHandler(rr, req) // Execute your handler

		// Error checking within a benchmark: Important but should be minimal
		if rr.Code != http.StatusCreated {
			b.Fatalf("Benchmark failed: unexpected status %v for key %s. Body: %s", rr.Code, key, rr.Body.String())
		}
		// Read the response body: Essential to ensure the full operation (including response writing)
		// is measured, and to prevent the compiler from optimizing away the write.
		_, err := io.Copy(io.Discard, rr.Result().Body)
		if err != nil && err != io.EOF {
			b.Fatalf("Failed to read response body in benchmark: %v", err)
		}
	}
	// 4. Report Allocations:
	// Tells the benchmark to include memory allocation statistics in the output (B/op, allocs/op).
	b.ReportAllocs()
}
