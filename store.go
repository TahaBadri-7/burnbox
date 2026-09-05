package main

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

var rdb *redis.Client

func initRedis(addr string) {
	rdb = redis.NewClient(&redis.Options{
		Addr: addr,
	})
}

func save(ctx context.Context, shareID, ciphertext string, ttl time.Duration) error {
	return rdb.Set(ctx, "secret:"+shareID, ciphertext, ttl).Err()
}

func burn(ctx context.Context, shareID string) (string, error) {
	return rdb.GetDel(ctx, "secret:"+shareID).Result()
}

// burnBroken is the naive version: read, then delete, as two separate trips
// to Redis. Kept only to demonstrate the race it creates. Never used in production.
func burnBroken(ctx context.Context, shareID string) (string, error) {
	key := "secret:" + shareID

	v, err := rdb.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}

	rdb.Del(ctx, key)

	return v, nil
}
