package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
)

//now our master gets a put requests and
// it forwards it to the volume server and return something

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

	// case "DELETE":
	// 	handleDelete(w, r)

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
	volumeServer := "http://localhost:3001/files/" + key
	request, err := http.NewRequest("PUT", volumeServer, r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		log.Printf("Master: Error sending PUT request to volume server %s: %v", volumeServer, err)
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

	volumeServer := "http://localhost:3001/files/" + key

	http.Redirect(w, r, volumeServer, http.StatusMovedPermanently)

	w.WriteHeader(http.StatusCreated)
}
