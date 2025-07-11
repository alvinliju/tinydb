package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/syndtr/goleveldb/leveldb"
)

// now our master gets a put requests and
// it forwards it to the volume server and return something
var httpClient *http.Client

var db *leveldb.DB

var volumeServers = []string{
	"http://localhost:3001",
}

func selectVolumeServer() string {
	// Simple round-robin for now
	return volumeServers[time.Now().UnixNano()%int64(len(volumeServers))]
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

	// volumeServer := "http://localhost:3001/file/" + key
	//we recieve the request right?
	// we have the key and we have the data in the request body
	//create me put request and send the data to the volume shit
	volumeServer := selectVolumeServer()
	redirectURI := volumeServer + "/files/" + key
	request, err := http.NewRequest("PUT", redirectURI, r.Body)
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

	volumeRespBody, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to read response from volume server", http.StatusInternalServerError)
		return
	}

	err = db.Put([]byte(key), []byte(volumeServer), nil)
	if err != nil {
		http.Error(w, "Error saving key to master", http.StatusInternalServerError)
	}
	fmt.Printf("Here is the key %s", string(volumeRespBody))
	w.WriteHeader(http.StatusCreated)
}

func handleGet(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Path[len("/"):]
	if key == "" {
		http.Error(w, "Key required", http.StatusBadRequest)
		return
	}

	//TODO check if the key exists in our master
	volumeServer, err := db.Get([]byte(key), nil)
	fmt.Println(string(volumeServer))
	if err != nil {
		if err == leveldb.ErrNotFound {
			fmt.Println(err)
			http.Error(w, "key not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	redirectURI := string(volumeServer) + "/files/" + key
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
