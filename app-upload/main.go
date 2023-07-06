package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/kcalixto/go-module/toolkit"
)

func main() {
	mux := routes()

	log.Println("starting server on port :8080")

	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		log.Fatal(err)
	}
}

func routes() http.Handler {
	mux := http.NewServeMux()

	mux.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir("."))))
	mux.HandleFunc("/upload", uploadFiles)
	mux.HandleFunc("/upload-one", uploadOneFile)

	return mux
}

func uploadFiles(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	t := toolkit.Tools{
		MaxFileSize:      1024 * 1024 * 1024, // ~1gb
		AllowedFileTypes: []string{"image/jpeg", "image/png", "image/gif"},
	}

	files, err := t.UploadFiles(req, "./uploads")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	out := ""
	for _, item := range files {
		out += fmt.Sprintf("uploaded %s to the uploads folder, renamed to %s\n", item.OriginalFileName, item.NewFileName)
	}

	_, _ = w.Write([]byte(out))
}

func uploadOneFile(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	t := toolkit.Tools{
		MaxFileSize:      1024 * 1024 * 1024, // ~1gb
		AllowedFileTypes: []string{"image/jpeg", "image/png", "image/gif"},
	}

	f, err := t.UploadOneFile(req, "./uploads")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	out := fmt.Sprintf("uploaded %s to the uploads folder, renamed to %s\n", f.OriginalFileName, f.NewFileName)

	_, _ = w.Write([]byte(out))
}
