package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"strings"
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

var volumeServers = []VolumeGroup{
	{Replicas: []string{"http://localhost:3001", "http://localhost:3002", "http://localhost:3003"}},
	{Replicas: []string{"http://localhost:3004", "http://localhost:3005", "http://localhost:3006"}},
	{Replicas: []string{"http://localhost:3007", "http://localhost:3008", "http://localhost:3009"}},
	{Replicas: []string{"http://localhost:3010", "http://localhost:3011", "http://localhost:3012"}},
}

func key2Volume(key string) VolumeGroup {
	serverIndex := [4]int{0, 222, 333, 666}
	//hash the key
	hash := md5.Sum([]byte(key))
	//take the hash and calculate the volumeServer Index cool?
	x := int(hash[0]) % len(volumeServers)
	var subVolIndex int
	for index, element := range serverIndex {
		if x <= element {
			subVolIndex = index
			break
		}
	}

	shardedGroup := volumeServers[subVolIndex]

	return shardedGroup
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

	var buf bytes.Buffer
	body := io.TeeReader(r.Body, &buf)
	//we nee to write to all the three volumes
	for i := 0; i < len(rVolumesFromSelectedSubVol); i++ {

		if i != 0 {
			body = bytes.NewReader(buf.Bytes())
		}

		rVolume := rVolumesFromSelectedSubVol[i]
		redirectURI := rVolume + "/files/" + key
		request, err := http.NewRequest("PUT", redirectURI, body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		client := httpClient
		resp, err := client.Do(request)
		if err != nil {
			log.Printf("Master: Error sending PUT request to volume server %s: %v", redirectURI, err)
			http.Error(w, "Failed to store file: volume server unreachable or error", http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		data, err := io.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, "Failed to read response from volume server", http.StatusInternalServerError)
			return
		}

		var result map[string]string
		json.Unmarshal(data, &result)
		hashKeyFromResponse = result["key"]
		fmt.Println(hashKeyFromResponse, "inside the loop getting the key")
	}

	//TODO: figure out a way to add the subvolumes dynamically
	encoded := hex.EncodeToString([]byte(strings.Join(rVolumesFromSelectedSubVol, ",")))
	fmt.Println(encoded, "value", "value shit ")

	err := db.Put([]byte(hashKeyFromResponse), []byte(encoded), nil)
	if err != nil {
		http.Error(w, "Error saving key to master", http.StatusInternalServerError)
	}
	fmt.Printf("Here is the key %s", string(hashKeyFromResponse))
	w.WriteHeader(http.StatusCreated)
}

func handleGet(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Path[len("/"):]
	if key == "" {
		http.Error(w, "Key required", http.StatusBadRequest)
		return
	}

	//TODO check if the key exists in our master
	encoded, err := db.Get([]byte(key), nil)
	if err != nil {
		if err == leveldb.ErrNotFound {
			fmt.Println(err)
			http.Error(w, "key not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	decoded, _ := hex.DecodeString(string(encoded))
	fmt.Println(decoded, "decoded")

	rVolume := strings.Split(string(decoded), ",")

	redirectURI := string(rVolume[rand.Intn(2)]) + "/files/" + key
	http.Redirect(w, r, string(redirectURI), http.StatusMovedPermanently)
}

func handleDelete(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Path[len("/"):]
	if key == "" {
		http.Error(w, "Key required", http.StatusBadRequest)
		return
	}

	//TODO:ping volumes and load pick a random one and store to keyvalue store
	volumeServer, err := db.Get([]byte(key), nil)
	redirectURI := string(volumeServer) + "/files" + key
	fmt.Println(string(volumeServer))
	if err != nil {
		if err == leveldb.ErrNotFound {
			http.Error(w, "key not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
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

	if resp.StatusCode != 204 {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = db.Delete([]byte(key), nil)
	if err != nil {
		http.Error(w, "Database Error", http.StatusInternalServerError)
		return
	}

	fmt.Printf(" %s Deleted", string(volumeRespBody))
	w.WriteHeader(http.StatusCreated)
}
