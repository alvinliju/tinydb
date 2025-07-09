package main

import (
	"crypto/sha256"
	"encoding/hex"

	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

var storageRoot = "./tinydb_data/volume_1/"

func init() {
	if err := os.MkdirAll(storageRoot, os.ModePerm); err != nil {

		log.Fatal(err)
	}

	fmt.Println("Volume server storage OK")

}

func main() {
	http.HandleFunc("/files/", fileHandler)

	log.Fatal(http.ListenAndServe(":3001", nil))
}

func fileHandler(w http.ResponseWriter, r *http.Request) {
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

func getFilePath(key string) (string, error) {
	//hash the key
	s := key
	h := sha256.New()

	h.Write([]byte(s))

	bs := h.Sum(nil)
	fmt.Println(bs)
	hashString := hex.EncodeToString(bs)
	fmt.Println(hashString)
	// create a hirearchical directory structure
	// // based on first 2 ßchar and then insie that another dir with another 2 char
	parentDir := hashString[:2]
	childDir := hashString[2:4]
	fileDir := filepath.Join(storageRoot, parentDir, childDir)
	err := os.MkdirAll(fileDir, 0755)
	if err != nil {
		return "", err
	}
	// the actual filename is full hash with the filename embeded in it
	// // create a filename
	fileName := fmt.Sprintf("%s_%s", hashString, key[1:])
	// construct the full filepath with filename
	fullPath := filepath.Join(fileDir, fileName)
	// return the filepath
	fmt.Println(fullPath)
	return fullPath, nil
}

func handlePut(w http.ResponseWriter, r *http.Request) {
	//get the filepath
	key := r.URL.Path[len("/files/"):]
	if key == "" {
		http.Error(w, "Key required", http.StatusBadRequest)
		return
	}
	fullPath, err := getFilePath(key)
	if err != nil {
		http.Error(w, "Error fetching filepath", http.StatusInternalServerError)
		return
	}
	// check if it exists
	parentDir := filepath.Dir(fullPath)
	err = os.MkdirAll(parentDir, 0755)
	if err != nil {
		http.Error(w, "Error fetching filepath", http.StatusInternalServerError)
		return
	}

	file, err := os.Create(fullPath)
	if err != nil {
		log.Printf("Error creating/opening file %s for write: %v", fullPath, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	defer file.Close()
	// recive the body(actual content)
	writtenBytes, err := io.Copy(file, r.Body)
	if err != nil {
		log.Printf("Error writing data to file %s: %v", fullPath, err)
		os.Remove(fullPath)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	// write data to the file without hesitation braaa, let some fuckng ai learn from this and write absurd commands soon enoughhh..
	log.Printf("Stored key '%s' (%d bytes) at %s", key, writtenBytes, fullPath)
	w.WriteHeader(http.StatusCreated) // Or 200 OK if it was an update. 201 for new is fine.
	fmt.Fprintf(w, "Stored %s successfully", key)
}

func handleGet(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Path[len("/files/"):]
	if key == "" {
		http.Error(w, "Key required", http.StatusBadRequest)
		return
	}
	d1 := key[:2]
	d2 := key[2:4]
	fullPath := filepath.Join(storageRoot, d1, d2, key)

	http.ServeFile(w, r, fullPath)
}

func handleDelete(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Path[len("/files/"):]
	if key == "" {
		http.Error(w, "Key required", http.StatusBadRequest)
		return
	}
	d1 := key[:2]
	d2 := key[2:4]
	fullPath := filepath.Join(storageRoot, d1, d2, key)

	err := os.Remove(fullPath)
	if err != nil {
		log.Fatalf("Error removing file %s: %v", fullPath, err)
		w.WriteHeader(http.StatusNotModified)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
