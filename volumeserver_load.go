package main

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

const (
	baseURL       = "http://localhost:3001/files"
	testFileSize  = 1 << 20 // 1MB
	numWorkers    = 100     // Number of concurrent goroutines
	numOperations = 1000    // Operations per worker
)

var (
	testData = make([]byte, testFileSize)
	client   = &http.Client{Timeout: 10 * time.Second}
)

func init() {
	rand.Read(testData) // Fill with random data
}

func TestNuclear(t *testing.T) {
	// Start by nuking the test directory
	os.RemoveAll("./tinydb_data")

	var wg sync.WaitGroup
	errorChan := make(chan error, numWorkers*numOperations)

	// Phase 1: Concurrent Writes
	t.Run("WriteStorm", func(t *testing.T) {
		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func(workerID int) {
				defer wg.Done()
				for j := 0; j < numOperations; j++ {
					key := fmt.Sprintf("worker%d-key%d", workerID, j)
					if err := put(key, testData); err != nil {
						errorChan <- fmt.Errorf("PUT %s failed: %v", key, err)
						return
					}
				}
			}(i)
		}
		wg.Wait()
	})

	// Phase 2: Read-Delete Chaos
	t.Run("ReadDeleteChaos", func(t *testing.T) {
		for i := 0; i < numWorkers; i++ {
			wg.Add(2) // One reader, one deleter per worker

			go func(workerID int) {
				defer wg.Done()
				for j := 0; j < numOperations; j++ {
					key := fmt.Sprintf("worker%d-key%d", workerID, j)
					if _, err := get(key); err != nil {
						errorChan <- err
						return
					}
				}
			}(i)

			go func(workerID int) {
				defer wg.Done()
				for j := 0; j < numOperations; j++ {
					key := fmt.Sprintf("worker%d-key%d", workerID, j)
					if err := delete(key); err != nil {
						errorChan <- err
						return
					}
				}
			}(i)
		}
		wg.Wait()
	})

	// Phase 3: Forensic Analysis
	t.Run("Forensics", func(t *testing.T) {
		close(errorChan)
		for err := range errorChan {
			t.Error(err)
		}

		// Check for orphaned files
		var orphanCount int
		filepath.Walk("./tinydb_data", func(path string, info os.FileInfo, err error) error {
			if !info.IsDir() {
				orphanCount++
				if orphanCount <= 5 {
					t.Logf("ORPHANED FILE: %s (size: %d)", path, info.Size())
				}
			}
			return nil
		})

		if orphanCount > 0 {
			t.Errorf("Found %d orphaned files", orphanCount)
		}
	})
}

// Helper functions
func put(key string, data []byte) error {
	req, _ := http.NewRequest("PUT", baseURL+"/"+key, bytes.NewReader(data))
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}
	return nil
}

func get(key string) ([]byte, error) {
	resp, err := client.Get(baseURL + "/" + key)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s failed: %d", key, resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func delete(key string) error {
	req, _ := http.NewRequest("DELETE", baseURL+"/"+key, nil)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("DELETE %s failed: %d", key, resp.StatusCode)
	}
	return nil
}
