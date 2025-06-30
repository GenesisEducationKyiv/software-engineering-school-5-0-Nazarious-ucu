package cache

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisClient[T any] struct {
	client *redis.Client
	logger *log.Logger
}

func NewRedisClient[T any](client *redis.Client, logger *log.Logger) *RedisClient[T] {
	return &RedisClient[T]{client: client, logger: logger}
}

func (c *RedisClient[T]) Set(
	ctx context.Context,
	key string,
	value T,
	expiration time.Duration,
) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	c.logger.Printf("setting %s to %s", key, string(data))
	return c.client.Set(ctx, key, data, expiration).Err()
}

func (c *RedisClient[T]) Get(ctx context.Context, key string, returnValue *T) error {
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}

	return json.Unmarshal(data, returnValue)
}
