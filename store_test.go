package main

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestBurnExactlyOnce(t *testing.T) {
	initRedis("127.0.0.1:6379")
	ctx := context.Background()

	const id = "race-test-good"
	if err := save(ctx, id, "hunter2", time.Minute); err != nil {
		t.Fatal("could not reach Redis:", err)
	}

	var winners atomic.Int64
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := burn(ctx, id); err == nil {
				winners.Add(1)
			}
		}()
	}

	wg.Wait()

	got := winners.Load()
	t.Logf("GETDEL: 100 goroutines tried, %d got the secret", got)

	if got != 1 {
		t.Fatalf("expected exactly 1 winner, got %d", got)
	}
}

func TestBurnBrokenOverServes(t *testing.T) {
	initRedis("127.0.0.1:6379")
	ctx := context.Background()

	const id = "race-test-broken"
	if err := save(ctx, id, "hunter2", time.Minute); err != nil {
		t.Fatal("could not reach Redis:", err)
	}

	var winners atomic.Int64
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := burnBroken(ctx, id); err == nil {
				winners.Add(1)
			}
		}()
	}

	wg.Wait()

	t.Logf("GET+DEL: 100 goroutines tried, %d got the secret", winners.Load())
}
