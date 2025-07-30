package main

import (
	"bytes"
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
	Primary  string
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
	{Primary: "http://localhost:3001", Replicas: []string{"http://localhost:3002", "http://localhost:3003"}},
}

func key2Volume(key string) VolumeGroup {
	//hash the key
	hash := xxhash.New().Sum([]byte(key))
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
		Timeout:   5 * time.Minute,
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

	// Get volume servers
	selectedSubVolume := key2Volume(key)
	bodyReader := bytes.NewReader(buf.Bytes())
	success, hashKeyFromResponse := writeToReplica(selectedSubVolume.Primary, bodyReader, key)
	if !success {
		http.Error(w, "Primary write failed", http.StatusInternalServerError)
		return
	}

	// Store in master DB
	allReplicas := append([]string{selectedSubVolume.Primary}, selectedSubVolume.Replicas...)
	value := strings.Join(allReplicas, ",")
	err = db.Put([]byte(hashKeyFromResponse), []byte(value), nil)
	if err != nil {
		http.Error(w, "Error saving key to master", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(fmt.Sprintf("Here is the key %s", hashKeyFromResponse)))
}

func writeToReplica(volumeString string, body io.Reader, key string) (bool, string) {
	s := key
	h := xxhash.New()
	h.Write([]byte(s))
	bs := h.Sum(nil)
	hashString := hex.EncodeToString(bs)
	
	parentDir := hashString[:2]
	childDir := hashString[2:4]
	fileDir := filepath.Join(parentDir, childDir)
	fileName := fmt.Sprintf("%s_%s", hashString, key)
	fullPath := filepath.Join(fileDir, fileName)

	baseURL := volumeString + "/files/" + fileName
	params := url.Values{}
	params.Add("filepath", fullPath)
	redirectURI := baseURL + "?" + params.Encode()
	
	request, err := http.NewRequest("PUT", redirectURI, body)
	if err != nil {
		return false, ""
	}

	client := httpClient
	resp, err := client.Do(request)
	if err != nil {
		return false, ""
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
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
		http.Error(w, "Key required", http.StatusBadRequest)
		return
	}

	fmt.Println("here")

	// Check if the key exists in our master
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

	// Redirect to the primary replica (the first one in the list)
	if len(rVolume) > 0 {
		redirectURI := rVolume[0] + "/files/" + key
		fmt.Println("redirectURI:", redirectURI)
		fmt.Println("rVolume:", rVolume)
		fmt.Printf("Redirecting to primary replica: %s\n", redirectURI)
		http.Redirect(w, r, string(redirectURI), http.StatusMovedPermanently)
	} else {
		http.Error(w, "No replicas found for key", http.StatusInternalServerError)
	}
}

func handleDelete(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Path[len("/"):]
	if key == "" {
		http.Error(w, "Key required", http.StatusBadRequest)
		return
	}

	v, err := db.Get([]byte(key), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			http.Error(w, "key not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	rVolume := strings.Split(string(v), ",")
	primary := rVolume[0] // First one is primary

	// Delete from primary only
	redirectURI := primary + "/files/" + key
	request, err := http.NewRequest("DELETE", redirectURI, nil)
	if err != nil {
		http.Error(w, "Delete request failed", http.StatusInternalServerError)
		return
	}

	resp, err := httpClient.Do(request)
	if err != nil {
		http.Error(w, "Delete failed", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == 204 {
		db.Delete([]byte(key), nil)
		w.WriteHeader(http.StatusNoContent)
	} else {
		http.Error(w, "Delete failed", resp.StatusCode)
	}
}
