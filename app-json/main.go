package main

import (
	"log"
	"net/http"

	"github.com/kcalixto/go-module/toolkit"
)

type RequestPayload struct {
	Action  string `json:"action"`
	Message string `json:"message"`
}

type ResponsePayload struct {
	Message    string `json:"message"`
	StatusCode int    `json:"status_code,omitempty"`
}

func main() {
	mux := http.NewServeMux()

	mux.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir("."))))

	mux.HandleFunc("/receive-post", receivePost)
	mux.HandleFunc("/remote-service", remoteService)
	mux.HandleFunc("/simulated-service", simulatedService)

	log.Println("starting service...")

	err := http.ListenAndServe(":8081", mux)
	if err != nil {
		log.Fatal(err)
	}
}

func receivePost(w http.ResponseWriter, r *http.Request) {
	var requestPayload RequestPayload
	var t toolkit.Tools

	err := t.ReadJSON(w, r, &requestPayload)
	if err != nil {
		t.ErrorJSONResponse(w, err)
		return
	}

	responsePayload := ResponsePayload{
		Message:    "hallo!",
		StatusCode: 200,
	}

	err = t.WriteJSON(w, http.StatusOK, responsePayload)
	if err != nil {
		log.Println(err)
	}
}

func remoteService(w http.ResponseWriter, r *http.Request) {
	var requestPayload RequestPayload
	var t toolkit.Tools

	err := t.ReadJSON(w, r, &requestPayload)
	if err != nil {
		t.ErrorJSONResponse(w, err)
		return
	}

	_, statusCode, err := t.PushJSONToRemote("http://localhost:8081/simulated-service", requestPayload)
	if err != nil {
		t.ErrorJSONResponse(w, err)
		return
	}

	responsePayload := ResponsePayload{
		Message:    "hallo!",
		StatusCode: statusCode,
	}

	err = t.WriteJSON(w, http.StatusOK, responsePayload)
	if err != nil {
		log.Println(err)
	}
}

func simulatedService(w http.ResponseWriter, r *http.Request) {
	payload := ResponsePayload{
		Message:    "okay",
		StatusCode: 200,
	}

	var t toolkit.Tools

	t.WriteJSON(w, http.StatusOK, &payload)
}
