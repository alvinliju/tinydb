package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/cespare/xxhash"
	"github.com/syndtr/goleveldb/leveldb"
)

// create an object for subvolumes
type SubVolume struct {
	Volume []string
}

var subVolumes = []SubVolume{
	{Volume: []string{"http://localhost:3001", "http://localhost:3002", "http://localhost:3003"}},
}

// leveldb record
type Record struct {
	replicas []string
}

var db *leveldb.DB

// create a struct for leveldb shit
func init() {
	var err error
	db, err = leveldb.OpenFile("/tmp/leveldb", nil)
	if err != nil {
		fmt.Println("awww it suckss")
		return
	}

}
func main() {
	http.HandleFunc("/", reqHandler)
	http.ListenAndServe(":8080", nil)
	defer func() {
		if db != nil {
			db.Close()
		}
	}()
}

func reqHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		handleGET(w, r)
	case "PUT":
		handlePUT(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// leveldb operations
func putKey(key []byte, value []byte) string {
	err := db.Put(key, value, nil)
	if err != nil {
		return ""
	}

	return string(key)
}

func getKey(key []byte) []byte {
	data, err := db.Get(key, nil)
	if err != nil {
		return nil
	}

	return data
}

// write a function to get the subvolumes from hash
func key2volume(key string) SubVolume {
	h := xxhash.New()
	h.Write([]byte(key))
	hash := h.Sum(nil)
	volIndex := int(hash[0]) % len(subVolumes)
	return subVolumes[volIndex]
}

// write a function to compute filename from key
func key2filename(key string) string {
	h := xxhash.New()
	h.Write([]byte(key))
	hash := h.Sum(nil)
	hashString := hex.EncodeToString(hash)
	filename := hashString + "_" + key
	return filename
}

// write a functi to write to replicas
func remote_put(volumeServers []string, key string, data io.Reader) (bool, string) {
	//get the key from url

	//we want to get the volume first

	fmt.Println("volumeServers", volumeServers)
	//hash the key for filename
	filename := key2filename(key)
	//append filename at the end of the hash with '_'
	//send a put request to the selected server
	//send a put request to the selected server
	fmt.Println("volumeServers.Volume", volumeServers)
	var buf bytes.Buffer
	body := io.TeeReader(data, &buf)
	for i, replicaUrl := range volumeServers {
		fmt.Println("replicaUrl", replicaUrl)
		fmt.Println("i", i)
		if i != 0 {
			body = bytes.NewReader(buf.Bytes())
		}

		url := replicaUrl + "/" + filename
		fmt.Println("url", url)
		req, err := http.NewRequest("PUT", url, body)
		if err != nil {
			return false, err.Error()
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return false, err.Error()
		}
		if resp == nil {
			fmt.Println("error: nil response")
			return false, ""
		}
		defer resp.Body.Close()
		if resp.StatusCode != 201 && resp.StatusCode != 204 {
			return false, ""
		}

	}

	//return success to as return with hashedKey
	return true, filename
}

//write getFunc,addFunc, deleteFunc for leveldb

func handlePUT(w http.ResponseWriter, r *http.Request) {
	path := r.URL.RequestURI()
	key := path[1:]
	fmt.Println("key", key)
	//what should this function do?
	//take in the r.Body
	//find a subvolume to put our things in
	//just write to all 3 volumes concurrently
	volumeServers := key2volume(key)

	success, hashedKey := remote_put(volumeServers.Volume, key, r.Body)
	fmt.Println("success", success)
	fmt.Println("hashedKey", hashedKey)
	if !success {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	//TODO create a seralization function for records
	value := strings.Join(volumeServers.Volume, ",")
	savedKey := putKey([]byte(hashedKey), []byte(value))
	if savedKey == "" {
		http.Error(w, "Error saving metadata", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(hashedKey))
	defer r.Body.Close()
	//return 200 else return false
}

func handleGET(w http.ResponseWriter, r *http.Request) {
	path := r.URL.RequestURI()
	key := path[1:]
	data := getKey([]byte(key))
	replicas := strings.Split(string(data), ",")
	redirectUrl := replicas[0] + "/" + key
	http.Redirect(w, r, redirectUrl, http.StatusPermanentRedirect)

}
