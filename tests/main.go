package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

func main() {
	fmt.Println("=== TinyDB Load Test ===")

	// Test 1: Sequential writes
	fmt.Println("\n1. Sequential write test (100 files)...")
	start := time.Now()
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("seq-test-%d", i)
		data := fmt.Sprintf("sequential data %d", i)

		resp, err := http.Post(
			fmt.Sprintf("http://localhost:3000/store/%s", key),
			"application/octet-stream",
			bytes.NewBuffer([]byte(data)),
		)
		if err != nil {
			fmt.Printf("FAIL: %s - %v\n", key, err)
			continue
		}
		resp.Body.Close()
	}
	fmt.Printf("Sequential: 100 writes in %v\n", time.Since(start))

	// Test 2: Concurrent writes
	fmt.Println("\n2. Concurrent write test (1000 files)...")
	var wg sync.WaitGroup
	var failures int
	var mu sync.Mutex

	start = time.Now()
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("concurrent-test-%d", id)
			data := fmt.Sprintf("concurrent data %d", id)

			resp, err := http.Post(
				fmt.Sprintf("http://localhost:3000/store/%s", key),
				"application/octet-stream",
				bytes.NewBuffer([]byte(data)),
			)
			if err != nil {
				mu.Lock()
				failures++
				mu.Unlock()
				return
			}
			resp.Body.Close()
		}(i)
	}
	wg.Wait()
	fmt.Printf("Concurrent: 1000 writes in %v (failures: %d)\n", time.Since(start), failures)

	// Test 3: Read performance
	fmt.Println("\n3. Read performance test...")
	start = time.Now()
	failures = 0

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("concurrent-test-%d", id)
			expected := fmt.Sprintf("concurrent data %d", id)

			resp, err := http.Get(fmt.Sprintf("http://localhost:3000/store/%s", key))
			if err != nil {
				mu.Lock()
				failures++
				mu.Unlock()
				return
			}

			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()

			if err != nil || string(body) != expected {
				mu.Lock()
				failures++
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
	fmt.Printf("Reads: 1000 reads in %v (failures: %d)\n", time.Since(start), failures)

	// Test 4: Mixed workload
	fmt.Println("\n4. Mixed workload test (50% reads, 50% writes)...")
	start = time.Now()
	failures = 0

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			if id%2 == 0 {
				// Write
				key := fmt.Sprintf("mixed-test-%d", id)
				data := fmt.Sprintf("mixed data %d", id)

				resp, err := http.Post(
					fmt.Sprintf("http://localhost:3000/store/%s", key),
					"application/octet-stream",
					bytes.NewBuffer([]byte(data)),
				)
				if err != nil {
					mu.Lock()
					failures++
					mu.Unlock()
					return
				}
				resp.Body.Close()
			} else {
				// Read
				key := fmt.Sprintf("concurrent-test-%d", id/2)
				resp, err := http.Get(fmt.Sprintf("http://localhost:3000/store/%s", key))
				if err != nil {
					mu.Lock()
					failures++
					mu.Unlock()
					return
				}
				resp.Body.Close()
			}
		}(i)
	}
	wg.Wait()
	fmt.Printf("Mixed: 1000 ops in %v (failures: %d)\n", time.Since(start), failures)

	// Test 5: Large file test
	fmt.Println("\n5. Large file test (1MB files)...")
	largeData := make([]byte, 1024*1024) // 1MB
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	start = time.Now()
	failures = 0

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			key := fmt.Sprintf("large-test-%d", id)

			resp, err := http.Post(
				fmt.Sprintf("http://localhost:3000/store/%s", key),
				"application/octet-stream",
				bytes.NewBuffer(largeData),
			)
			if err != nil {
				mu.Lock()
				failures++
				mu.Unlock()
				return
			}
			resp.Body.Close()
		}(i)
	}
	wg.Wait()
	fmt.Printf("Large files: 10 x 1MB writes in %v (failures: %d)\n", time.Since(start), failures)

	fmt.Println("\n=== Test Complete ===")
}
