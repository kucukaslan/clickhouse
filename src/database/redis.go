package database

import (
	"context"
	"errors"
	"fmt"
	"kucukaslan/clickhouse/config"
	"kucukaslan/clickhouse/domain"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

var redisClient *redis.Client

type ClickHouseRedis struct {
	*redis.Client
	expirationMilliseconds int64
}

const RedisKeyPrefix = "clickhouse_event:"

func (r ClickHouseRedis) getExpirationDuration() (durationMilliseconds time.Duration) {
	if r.expirationMilliseconds <= 0 {
		return 0
	}
	return time.Duration(r.expirationMilliseconds) * time.Millisecond
}
func (r ClickHouseRedis) SetEventProcessed(ctx context.Context, request domain.EventRequest) {
	key := RedisKeyPrefix + request.GetUniqueKey()
	r.SetEx(ctx, key, "1", r.getExpirationDuration())
}

func (r ClickHouseRedis) IsEventProcessed(ctx context.Context, request domain.EventRequest) (bool, error) {
	key := RedisKeyPrefix + request.GetUniqueKey()
	result, err := r.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return result == "1", nil
}

// use msetex to set multiple keys with expiration
func (r ClickHouseRedis) SetMultipleEventsProcessed(ctx context.Context, requests []domain.EventRequest) error {
	pipe := r.Pipeline()
	for _, request := range requests {
		key := RedisKeyPrefix + request.GetUniqueKey()
		pipe.SetEx(ctx, key, "1", r.getExpirationDuration())
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (r ClickHouseRedis) AreEventsProcessed(ctx context.Context, requests []domain.EventRequest) (map[string]bool, error) {
	keys := make([]string, len(requests))
	for i, request := range requests {
		keys[i] = RedisKeyPrefix + request.GetUniqueKey()
	}

	results, err := r.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}

	processedMap := make(map[string]bool)
	for i, result := range results {
		if result == nil {
			processedMap[requests[i].GetUniqueKey()] = false
		} else if str, ok := result.(string); ok && str == "1" {
			processedMap[requests[i].GetUniqueKey()] = true
		}
	}
	return processedMap, nil
}

// InitRedis initializes the Redis client connection
func InitRedis(cfg *config.RedisConfig) error {
	addr := cfg.GetRedisAddr()

	opts := &redis.Options{
		Addr:     addr,
		Password: cfg.Password,
		DB:       0, // default DB
	}

	client := redis.NewClient(opts)

	// Test the connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	redisClient = client
	log.Println("Redis connection established successfully")
	return nil
}

// CloseRedis closes the Redis client connection
func CloseRedis() error {
	if redisClient != nil {
		if err := redisClient.Close(); err != nil {
			return fmt.Errorf("failed to close Redis connection: %w", err)
		}
		log.Println("Redis connection closed")
	}
	return nil
}

// RedisHealthCheck verifies that the Redis connection is alive
func RedisHealthCheck(ctx context.Context) error {
	if redisClient == nil {
		return fmt.Errorf("Redis connection is not initialized")
	}
	return redisClient.Ping(ctx).Err()
}

func GetRedisClient(redisCacheDurationMS int64) ClickHouseRedis {
	return ClickHouseRedis{redisClient, redisCacheDurationMS}
}
