package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"

	_ "net/http/pprof"

	"github.com/cespare/xxhash"
	"github.com/syndtr/goleveldb/leveldb"
)

// now our master gets a put requests and
// it forwards it to the volume server and return something

var httpClient *http.Client

var db *leveldb.DB

type VolumeGroup struct {
	Replicas []string
}

type Result struct {
	ID      int
	Success bool
	Data    string
}

type ReadResult struct {
	ID      int
	Success bool
	Replica string
	Exists  bool
}

var volumeServers = []VolumeGroup{
	{Replicas: []string{"http://localhost:3001", "http://localhost:3002", "http://localhost:3003"}},
	{Replicas: []string{"http://localhost:3004", "http://localhost:3005", "http://localhost:3006"}},
	{Replicas: []string{"http://localhost:3007", "http://localhost:3008", "http://localhost:3009"}},
	{Replicas: []string{"http://localhost:3010", "http://localhost:3011", "http://localhost:3012"}},
}

func key2Volume(key string) VolumeGroup {
	//hash the key
	hash := md5.Sum([]byte(key))
	//take the hash and calculate the volumeServer Index cool?
	x := int(hash[0]) % len(volumeServers)
	fmt.Println("Volume Group Index:", x)
	return volumeServers[x]
}

func getWorker(id int, ch chan<- Result, wg *sync.WaitGroup, replicaUrl string, body io.Reader, key string) {
	defer wg.Done()

	success, hashedKey := writeToReplica(replicaUrl, body, key)

	if !success {
		ch <- Result{
			ID:      id,
			Success: false,
			Data:    fmt.Sprintf("Task %d not completed", id),
		}
	} else {
		ch <- Result{
			ID:      id,
			Success: true,
			Data:    fmt.Sprintf("%s", hashedKey),
		}
	}

}

func init() {
	var tr = &http.Transport{
		MaxIdleConns:        100,              // Total max idle connections
		MaxIdleConnsPerHost: 10,               // Max idle connections per host
		IdleConnTimeout:     30 * time.Second, // Idle connection timeout
		DisableKeepAlives:   false,
	}
	httpClient = &http.Client{
		Transport: tr,
	}

	var err error
	db, err = leveldb.OpenFile("./tinydb_master", nil)
	if err != nil {
		fmt.Println("Error connecting leveldb", err)
		return
	}

}

func main() {
	http.HandleFunc("/", handleRequests)

	log.Fatal(http.ListenAndServe(":3000", nil))
}

func handleRequests(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		handleGet(w, r)
	case "PUT":
		handlePut(w, r)

	case "DELETE":
		handleDelete(w, r)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}

}

func handlePut(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Path[len("/"):]
	if key == "" {
		http.Error(w, "Key required", http.StatusBadRequest)
		return
	}

	// Read the entire body into a buffer
	var buf bytes.Buffer
	_, err := io.Copy(&buf, r.Body)

	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}

	//hashes filename in our volume server
	// since the key and hash algo is the same all the servers should return the same hashed file name
	var hashKeyFromResponse string = ""

	//get volume servers
	selectedSubVolume := key2Volume(key)

	rVolumesFromSelectedSubVol := selectedSubVolume.Replicas
	fmt.Println(rVolumesFromSelectedSubVol)
	fmt.Println(key)
	var wg sync.WaitGroup
	resultChan := make(chan Result, 3)
	for i := 1; i <= 3; i++ {
		wg.Add(1)
		bodyReader := bytes.NewReader(buf.Bytes())
		go getWorker(i, resultChan, &wg, selectedSubVolume.Replicas[i-1], bodyReader, key)
	}

	wg.Wait()
	close(resultChan)

	// Collect results
	var results []Result
	for result := range resultChan {
		results = append(results, result)
		fmt.Printf("Received: %+v\n", result)
	}

	var successCount int = 0
	for index := range results {
		if results[index].Success == true {
			hashKeyFromResponse = results[index].Data
			successCount++
		}
	}

	if successCount < 2 {
		http.Error(w, "Not enough quoram writes", http.StatusInternalServerError)
		return
	}

	//TODO: figure out a way to add the subvolumes dynamically
	value := strings.Join(rVolumesFromSelectedSubVol, ",")
	fmt.Println(value, "value stored in db")

	err = db.Put([]byte(hashKeyFromResponse), []byte(value), nil)
	if err != nil {
		http.Error(w, "Error saving key to master", http.StatusInternalServerError)
	}

	userPayload := fmt.Sprintf("Here is the key %s", string(hashKeyFromResponse))
	w.Write([]byte(userPayload))
	w.WriteHeader(http.StatusCreated)
}

func writeToReplica(volumeString string, body io.Reader, key string) (bool, string) {
	s := key
	h := xxhash.New()

	h.Write([]byte(s))

	bs := h.Sum(nil)
	hashString := hex.EncodeToString(bs)
	// create a hirearchical directory structure
	// // based on first 2 ßchar and then insie that another dir with another 2 char
	parentDir := hashString[:2]
	childDir := hashString[2:4]
	fileDir := filepath.Join(parentDir, childDir)
	fileName := fmt.Sprintf("%s_%s", hashString, key)
	// construct the full filepath with filename
	fullPath := filepath.Join(fileDir, fileName)
	fmt.Println(fullPath)

	baseURL := volumeString + "/files/" + fileName
	params := url.Values{}
	params.Add("filepath", fullPath)
	redirectURI := baseURL + "?" + params.Encode()
	fmt.Println(redirectURI)
	request, err := http.NewRequest("PUT", redirectURI, body)
	if err != nil {
		log.Println(err.Error(), http.StatusInternalServerError)
		return false, ""
	}

	client := httpClient
	resp, err := client.Do(request)
	if err != nil {
		log.Printf("Master: Error sending PUT request to volume server %s: %v", redirectURI, err)
		return false, ""
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response from volume server")
		return false, ""
	}

	var result map[string]string
	json.Unmarshal(data, &result)
	hashKeyFromResponse := result["key"]

	return true, hashKeyFromResponse
}

func handleGet(w http.ResponseWriter, r *http.Request) {
	fmt.Println("here")
	key := r.URL.Path[len("/"):]
	if key == "" {
		http.Error(w, "Key requiredddd", http.StatusBadRequest)
		return
	}

	fmt.Println("here")

	//TODO check if the key exists in our master
	v, err := db.Get([]byte(key), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			fmt.Println(err)
			http.Error(w, "key not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	fmt.Println(string(v), "decoded shits")

	rVolume := strings.Split(string(v), ",")

	//after we get rVolumes
	// create a read result channel
	// create 3 workers and a read functions
	// check if max numbers of quorums are met then redirect

	var wg sync.WaitGroup
	resultChan := make(chan ReadResult, len(rVolume))

	for i, replicaUrl := range rVolume {
		wg.Add(1)
		go readWorker(i+1, resultChan, &wg, replicaUrl, key)
	}

	wg.Wait()
	close(resultChan)

	//collect results
	var results []ReadResult
	var healthyReplicas []string

	// fmt.Println(rVolume, "rVolumes")
	// var healthyReplica string
	// for i := 0; i < len(rVolume); i++ {
	// 	//send a health check to the servers and choose a healthy one
	// 	url := rVolume[i] + "/health"
	// 	request, err := http.NewRequest("GET", url, nil)
	// 	if err != nil {
	// 		fmt.Println("Error during health check:", err)
	// 		continue
	// 	}

	// 	fmt.Println(request)
	// 	client := httpClient
	// 	resp, err := client.Do(request)
	// 	if err != nil {
	// 		fmt.Println("Error during health check:", err)
	// 		continue
	// 	}
	// 	defer resp.Body.Close()
	// 	fmt.Println(resp.Status)
	// 	if resp.StatusCode == 200 {
	// 		healthyReplica = rVolume[i]
	// 		fmt.Println(healthyReplica)
	// 		resp.Body.Close()
	// 		break
	// 	}
	// }

	for result := range resultChan {
		results = append(results, result)
		if result.Success && result.Exists {
			healthyReplicas = append(healthyReplicas, result.Replica)
		}
	}

	totalReplicas := len(rVolume)
	requiredQuorum := (totalReplicas / 2) + 1

	if len(healthyReplicas) < requiredQuorum {
		http.Error(w, "File not available on enough replicas", http.StatusServiceUnavailable)
		return
	}

	redirectURI := healthyReplicas[0] + "/files/" + key
	fmt.Println("redirectURI:", redirectURI)
	fmt.Println("rVolume:", rVolume)
	fmt.Printf("Quorum satisfied (%d/%d), redirecting to: %s\n", len(healthyReplicas), totalReplicas, redirectURI)
	http.Redirect(w, r, string(redirectURI), http.StatusMovedPermanently)
}

func readWorker(id int, ch chan<- ReadResult, wg *sync.WaitGroup, replicaUrl string, key string) {
	defer wg.Done()

	//check if file exists with HEAD request
	checkURI := replicaUrl + "/files/" + key
	request, err := http.NewRequest("GET", checkURI, nil)
	if err != nil {
		fmt.Println("Error getting repsonse to HEAD request", err)
		ch <- ReadResult{
			ID:      id,
			Success: false,
			Replica: replicaUrl,
			Exists:  false,
		}
		return
	}

	fmt.Println(request)
	client := httpClient
	resp, err := client.Do(request)
	if err != nil {
		fmt.Println("Error during HEAD request", err)
		ch <- ReadResult{
			ID:      id,
			Success: false,
			Replica: replicaUrl,
			Exists:  false,
		}
		return
	}
	defer resp.Body.Close()

	exists := resp.StatusCode == 200

	ch <- ReadResult{
		ID:      id,
		Success: true,
		Replica: replicaUrl,
		Exists:  exists,
	}
}

func handleDelete(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Path[len("/"):]
	if key == "" {
		http.Error(w, "Key required", http.StatusBadRequest)
		return
	}

	//TODO:ping volumes and load pick a random one and store to keyvalue store
	v, err := db.Get([]byte(key), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			fmt.Println(err)
			http.Error(w, "key not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	fmt.Println(string(v), "decoded shits")

	rVolume := strings.Split(string(v), ",")

	var wg sync.WaitGroup
	resultChan := make(chan Result, 3)
	for i := 1; i <= 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			redirectURI := string(rVolume[i-1]) + "/files/" + key
			request, err := http.NewRequest("DELETE", string(redirectURI), r.Body)
			if err != nil {
				resultChan <- Result{ID: i + 1, Success: false, Data: err.Error()}
				return
			}

			client := httpClient
			resp, err := client.Do(request)
			if err != nil {
				log.Printf("Master: Error sending DEL request to volume server %s: %v", redirectURI, err)
				// A 502 Bad Gateway is appropriate if the upstream server (Volume Server) is unreachable or errors out.
				http.Error(w, "Failed to store file: volume server unreachable or error", http.StatusBadGateway)
				return
			}
			defer resp.Body.Close()

			volumeRespBody, err := io.ReadAll(resp.Body)
			if err != nil {
				resultChan <- Result{ID: i + 1, Success: false, Data: err.Error()}
				return
			}

			_ = volumeRespBody

			if resp.StatusCode != 204 {
				resultChan <- Result{
					ID:      i + 1,
					Success: false,
					Data:    fmt.Sprintf("Task %d not completed", i),
				}
				return
			}

			success := resp.StatusCode == 204
			resultChan <- Result{ID: i + 1, Success: success, Data: fmt.Sprintf("Status: %d", resp.StatusCode)}

			w.WriteHeader(http.StatusNoContent)
		}()
	}

	wg.Wait()
	close(resultChan)

	var failures []string
	for result := range resultChan {
		if !result.Success {
			failures = append(failures, fmt.Sprintf("replica_%d", result.ID))
		}
	}

	// Only delete from master if ALL replicas succeeded
	if len(failures) > 0 {
		log.Printf("DELETE failed on replicas: %v", failures)
		http.Error(w, fmt.Sprintf("Failed to delete from %d replicas", len(failures)),
			http.StatusPartialContent)
		return
	}

	err = db.Delete([]byte(key), nil)
	if err != nil {
		http.Error(w, "Database Error", http.StatusInternalServerError)
		return
	}

	fmt.Printf(" %s Deleted", string(key))
	w.WriteHeader(http.StatusCreated)
}
