package main

import (
	"log"
	"net/http"

	"github.com/kcalixto/go-module/toolkit"
)

func main() {
	//routes
	mux := routes()

	//server
	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		log.Fatal(err)
	}
}

func routes() http.Handler {
	mux := http.NewServeMux()

	// mux request handler
	currentDir := http.Dir(".")
	mux.Handle("/", http.StripPrefix("/", http.FileServer(currentDir)))

	// routes
	mux.HandleFunc("/download", downloadFile)

	return mux
}

func downloadFile(w http.ResponseWriter, r *http.Request) {
	var t toolkit.Tools
	t.DownloadStaticFile(w, r, "./files", "image.jpeg", "netflix.jpeg")
}
