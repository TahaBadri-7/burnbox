package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"

)

const maxBodyBytes = 64 * 1024 // 64KB

type createRequest struct {
	Ciphertext string `json:"ciphertext"`
}

type createRespnse struct {
	ShareID   string `json:"share_id"`
	CreatorID string `json:"creator_id"`
}

func handleCreate(w http.ResponseWriter, r *http.Request) {
	
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
			return
		}

		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if req.Ciphertext == "" {
		http.Error(w, "Missing ciphertext", http.StatusBadRequest)
		return
	}

	shareID, err := newID()
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	creatorID, err := newID()
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if err := save(ctx, shareID, req.Ciphertext, 24*time.Hour); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createRespnse{
		ShareID:   shareID,
		CreatorID: creatorID,
	})
}

type burnResponse struct {
	Ciphertext string `json:"ciphertext"`
}

func handleBurn(w http.ResponseWriter, r *http.Request) {

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	
	id := r.PathValue("id")

	ciphertext, err := burn(ctx, id)
	if errors.Is(err, redis.Nil) {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(burnResponse{
		Ciphertext: ciphertext,
	})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		http.Error(w, "DataBase not available", http.StatusServiceUnavailable)
		return
	}

	fmt.Fprintln(w, "ok")
}
