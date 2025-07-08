package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisClient[T any] struct {
	client     *redis.Client
	logger     *log.Logger
	expiration time.Duration
}

func NewRedisClient[T any](
	client *redis.Client,
	logger *log.Logger,
	expiration time.Duration,
) *RedisClient[T] {
	return &RedisClient[T]{client: client, logger: logger, expiration: expiration}
}

func (c *RedisClient[T]) Set(
	ctx context.Context,
	key string,
	value T,
) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	c.logger.Printf("setting %s to %s", key, string(data))
	return c.client.Set(ctx, key, data, c.expiration).Err()
}

//nolint:ireturn
func (c *RedisClient[T]) Get(ctx context.Context, key string) (T, error) {
	data, err := c.client.Get(ctx, key).Bytes()

	var zero T
	if err != nil {
		return zero, err
	}

	result := new(T)

	var v T

	if err := json.Unmarshal(data, result); err != nil {
		return zero, fmt.Errorf("unmarshal: %w", err)
	}
	return v, nil
}
