package main

import (
	"sync"
	"testing"
)

func TestBurnUnderLoad(t *testing.T) {
	secrets = make(map[string]secret)
	secrets["test-id"] = secret{ciphertext: "hunter2"}

	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			burn("test-id")
		}()
	}
	wg.Wait()
}
