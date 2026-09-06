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

func save(ctx context.Context, shareID, creatorID, ciphertext string, ttl time.Duration) error {
	pipe := rdb.TxPipeline()
	pipe.Set(ctx, "secret:"+shareID, ciphertext, ttl)
	pipe.Set(ctx, "creator:"+creatorID, shareID, ttl)

	_, err := pipe.Exec(ctx)
	return err
}

type secretStatus struct {
	Read      bool      `json:"read"`
	ExpiresAt time.Time `json:"expires_at"`
}


func status(ctx context.Context, creatorID string) (secretStatus, error) {
	key := "creator:" + creatorID

	shareID, err := rdb.Get(ctx, key).Result()
	if err != nil {
		return secretStatus{}, err
	}

	ttl, err := rdb.TTL(ctx, key).Result()
	if err != nil {
		return secretStatus{}, err
	}

	n, err := rdb.Exists(ctx, "secret:"+shareID).Result()
	if err != nil {
		return secretStatus{}, err
	}

	return secretStatus{
		Read:      n == 0,
		ExpiresAt: time.Now().Add(ttl),
	}, nil
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

func allow(ctx context.Context, ip string, limit int, window time.Duration) (bool, error) {
	key := "rate:" + ip

	pipe := rdb.TxPipeline()
	incr := pipe.Incr(ctx, key)
	pipe.ExpireNX(ctx, key, window)

	if _, err := pipe.Exec(ctx); err != nil {
		return false, err
	}

	return incr.Val() <= int64(limit), nil
}

