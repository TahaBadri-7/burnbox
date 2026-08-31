package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
)

func main() {

	key, _ := newKey()

	box, _ := encrypt([]byte("hunter2"), key)
	log.Println("bytes:", len(box), base64.RawURLEncoding.EncodeToString(box))

	out, err3 := decrypt(box, key)
	log.Println("decrypted:", string(out), "err:", err3)

	box2, _ := encrypt([]byte("hunter2"), key)
	log.Println("same input, different output:",
		base64.RawURLEncoding.EncodeToString(box) != base64.RawURLEncoding.EncodeToString(box2))

	box[len(box)-1] ^= 0x01
	out2, err2 := decrypt(box, key)
	log.Println("tampered:", string(out2), "err:", err2)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "OK")
	})

	mux.HandleFunc("POST /api/secrets", handleCreate)

	mux.HandleFunc("POST /api/secrets/{id}/burn", handleBurn)

	log.Println("Listening on http://localhost:8080")

	err := http.ListenAndServe(":8080", mux)
	if err != nil {
		log.Fatal(err)
	}
}
