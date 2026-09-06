package main

import (
	"crypto/rand"
	"encoding/base64"
)

// newID returns a URL-safe, cryptographically random identifier.
func newID() (string, error) {
	b := make([]byte, 16) // []byte is a slice of bytes, Go's closest thing to a JS array of numbers. make creates one with room for 16 entries, all set to 0.

	if _, err := rand.Read(b); err != nil { //  That's a common Go shape: you allocate the space, the function writes into it.
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}
