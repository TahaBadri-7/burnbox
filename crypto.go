package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
)

// newID returns a URL-safe, cryptographically random identifier.
func newID() (string, error) {
	b := make([]byte, 16) // []byte is a slice of bytes, Go's closest thing to a JS array of numbers. make creates one with room for 16 entries, all set to 0.

	if _, err := rand.Read(b); err != nil { //  That's a common Go shape: you allocate the space, the function writes into it.
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}

func newKey() ([]byte, error) {
	key := make([]byte, 32) // AES-256 requires a 32-byte key

	if _, err := rand.Read(key); err != nil {
		return nil, err
	}

	return key, nil
}

func encrypt(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

func decrypt(ciphertext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce := ciphertext[:nonceSize]
	ciphertext = ciphertext[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
