package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
)

//go:embed web
var webFS embed.FS

func main() {

	initRedis("127.0.0.1:6379")

	mux := http.NewServeMux()

	web, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatal(err)
	}

	mux.Handle("GET /", http.FileServerFS(web))

	mux.HandleFunc("GET /s/{id}", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, web, "reveal.html")
	})

	mux.HandleFunc("GET /healthz", handleHealth)

	mux.HandleFunc("POST /api/secrets", handleCreate)

	mux.HandleFunc("POST /api/secrets/{id}/burn", handleBurn)

	mux.HandleFunc("GET /api/secrets/{id}/status", handleStatus)

	mux.HandleFunc("GET /c/{id}", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFileFS(w, r, web, "creator.html")
	})

	log.Println("Listening on http://localhost:8080")

	err = http.ListenAndServe(":8080", mux)
	if err != nil {
		log.Fatal(err)
	}
}
