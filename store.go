package main

import "sync"

var (
	mu      sync.Mutex
	secrets = make(map[string]secret)
)

type secret struct {
	ciphertext string
	creatorID  string
}

func save(shareID string, s secret) {
	mu.Lock()
	defer mu.Unlock()
	secrets[shareID] = s
}

// burn returns the ciphertext for id and removes it, so it can only
// succeed once. Returns ok=false if the id is unknown or already burned.

func burn(id string) (string, bool) {

	mu.Lock()
	defer mu.Unlock()

	s, ok := secrets[id]

	if !ok {
		return "", false
	}

	delete(secrets, id)
	return s.ciphertext, true
}
