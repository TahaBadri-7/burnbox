package main

import (
	"log"
	"net/http"
)

func main() {

	initRedis("127.0.0.1:6379")

	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", handleHealth)

	mux.HandleFunc("POST /api/secrets", handleCreate)

	mux.HandleFunc("POST /api/secrets/{id}/burn", handleBurn)

	log.Println("Listening on http://localhost:8080")

	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		log.Fatal(err)
	}
}
