package main

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	_ "net/http/pprof"

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

func worker(id int, ch chan<- Result, wg *sync.WaitGroup, replicaUrl string, body io.Reader, key string) {
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
	httpClient = &http.Client{
		Timeout: 10 * time.Second,
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

	//hashes filename in our volume server
	// since the key and hash algo is the same all the servers should return the same hashed file name
	var hashKeyFromResponse string = ""

	//get volume servers
	selectedSubVolume := key2Volume(key)

	rVolumesFromSelectedSubVol := selectedSubVolume.Replicas
	fmt.Println(rVolumesFromSelectedSubVol)

	var buf bytes.Buffer
	io.TeeReader(r.Body, &buf)
	//we nee to write to all the three volumes
	var wg sync.WaitGroup
	resultChan := make(chan Result, 3)
	for i := 1; i <= 3; i++ {
		wg.Add(1)
		bodyReader := bytes.NewReader(buf.Bytes())
		go worker(i, resultChan, &wg, selectedSubVolume.Replicas[i-1], bodyReader, key)
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

	err := db.Put([]byte(hashKeyFromResponse), []byte(value), nil)
	if err != nil {
		http.Error(w, "Error saving key to master", http.StatusInternalServerError)
	}

	userPayload := fmt.Sprintf("Here is the key %s", string(hashKeyFromResponse))
	w.Write([]byte(userPayload))
	w.WriteHeader(http.StatusCreated)
}

func writeToReplica(volumeString string, body io.Reader, key string) (bool, string) {
	redirectURI := volumeString + "/files/" + key
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

	fmt.Println(rVolume, "rVolumes")
	var healthyReplica string
	for i := 0; i < len(rVolume); i++ {
		//send a health check to the servers and choose a healthy one
		url := rVolume[i] + "/health"
		request, err := http.NewRequest("GET", url, nil)
		if err != nil {
			fmt.Println("Error during health check:", err)
			continue
		}

		fmt.Println(request)
		client := httpClient
		resp, err := client.Do(request)
		if err != nil {
			fmt.Println("Error during health check:", err)
			continue
		}
		defer resp.Body.Close()
		fmt.Println(resp.Status)
		if resp.StatusCode == 200 {
			healthyReplica = rVolume[i]
			fmt.Println(healthyReplica)
			resp.Body.Close()
			break
		}
	}

	if healthyReplica == "" {
		http.Error(w, "All replicas failed", http.StatusServiceUnavailable)
		return
	}

	redirectURI := healthyReplica + "/files/" + key
	fmt.Println("redirectURI:", redirectURI)
	fmt.Println("rVolume:", rVolume)
	http.Redirect(w, r, string(redirectURI), http.StatusMovedPermanently)
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

	for _, elm := range rVolume {

		redirectURI := string(elm) + "/files/" + key
		request, err := http.NewRequest("DELETE", string(redirectURI), r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
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
			http.Error(w, "Failed to read response from volume server", http.StatusInternalServerError)
			return
		}

		_ = volumeRespBody

		if resp.StatusCode != 204 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	err = db.Delete([]byte(key), nil)
	if err != nil {
		http.Error(w, "Database Error", http.StatusInternalServerError)
		return
	}

	fmt.Printf(" %s Deleted", string(key))
	w.WriteHeader(http.StatusCreated)
}
