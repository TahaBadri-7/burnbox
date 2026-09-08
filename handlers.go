package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
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

	if err := save(ctx, shareID, creatorID, req.Ciphertext, 24*time.Hour); err != nil {
		http.Error(w, "server error", http.StatusInternalServerError)
		return
	}

	// One global integer. No user, no IP, no timestamp — nothing that could
	// identify anybody. Deliberately the only thing counted anywhere.
	rdb.Incr(ctx, "stats:created")

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

	rdb.Incr(ctx, "stats:burned")

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

func handleStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	st, err := status(ctx, r.PathValue("id"))
	if errors.Is(err, redis.Nil) {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "unavailable", http.StatusServiceUnavailable)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(st)
}

// clientIP returns the address of the real caller.
//
// Behind Caddy every request arrives from the proxy, so RemoteAddr is useless
// for rate limiting. Caddy is configured to OVERWRITE X-Forwarded-For with the
// address it actually saw, so anything a client tried to put there is already
// gone. Taking the last entry is belt-and-braces: with one trusted proxy, the
// last value is always the one the proxy appended.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if i := strings.LastIndex(xff, ","); i >= 0 {
			xff = xff[i+1:]
		}
		return strings.TrimSpace(xff)
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// rateLimited wraps a handler so it only runs if the caller is under the limit.
func rateLimited(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		ok, err := allow(ctx, clientIP(r), 10, time.Minute)
		if err != nil {
			http.Error(w, "unavailable", http.StatusServiceUnavailable)
			return
		}

		if !ok {
			w.Header().Set("Retry-After", "60")
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}

		next(w, r)
	}
}
