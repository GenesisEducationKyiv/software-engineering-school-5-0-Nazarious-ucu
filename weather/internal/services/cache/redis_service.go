package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

type RedisClient[T any] struct {
	client     *redis.Client
	logger     zerolog.Logger
	expiration time.Duration
}

func NewRedisClient[T any](
	client *redis.Client,
	logger zerolog.Logger,
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
		c.logger.Error().
			Ctx(ctx).
			Err(err).
			Msg("failed to marshal value for cache")
		return err
	}

	c.logger.Info().
		Ctx(ctx).
		Str("key", key).
		Str("value", string(data)).
		Dur("expiration", c.expiration).
		Msg("writing to cache")

	if err := c.client.Set(ctx, key, data, c.expiration).Err(); err != nil {
		c.logger.Error().
			Ctx(ctx).
			Str("key", key).
			Err(err).
			Msg("cache write failed")
		return err
	}
	return nil
}

//nolint:ireturn
func (c *RedisClient[T]) Get(ctx context.Context, key string) (T, error) {
	var zero T

	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		c.logger.Error().
			Ctx(ctx).
			Str("key", key).
			Err(err).
			Msg("cache read failed")
		return zero, err
	}

	result := new(T)
	if err := json.Unmarshal(data, result); err != nil {
		c.logger.Error().
			Ctx(ctx).
			Str("key", key).
			Err(err).
			Msg("failed to unmarshal cached data")
		return zero, fmt.Errorf("unmarshal: %w", err)
	}

	c.logger.Info().
		Ctx(ctx).
		Str("key", key).
		Msg("cache hit")
	return *result, nil
}
