package main

import (
	"encoding/json"
	"errors"
	"net/http"
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

	save(shareID, secret{
		ciphertext: req.Ciphertext,
		creatorID:  creatorID,
	})

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
	id := r.PathValue("id")

	ciphertext, ok := burn(id)
	if !ok {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(burnResponse{
		Ciphertext: ciphertext,
	})
}
