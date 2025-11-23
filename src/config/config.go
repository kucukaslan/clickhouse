package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all application configuration
type Config struct {
	Port       string
	ClickHouse ClickHouseConfig
	Redis      RedisConfig
}

// ClickHouseConfig holds ClickHouse connection settings
type ClickHouseConfig struct {
	Host                   string
	Port                   string
	Database               string
	User                   string
	Password               string
	DSN                    string
	AsyncInsertEnabled     bool  // whether to use async inserts
	AsyncInsertWait        int   // wait_for_async_insert (0 or 1)
	AsyncInsertMaxDataSize int64 // async_insert_max_data_size in bytes
	AsyncInsertBusyTimeout int   // async_insert_busy_timeout_ms in milliseconds
	RedisCacheDurationMS   int64 // duration to cache in Redis in milliseconds
	BufferChannelCapacity  int   // capacity of the event buffer channel (default: 50,000)
	BatchSize              int   // number of events to batch before flushing (default: 10,000)
	FlushIntervalSeconds   int   // time interval in seconds to flush batches (default: 1)
}

// RedisConfig holds Redis connection settings
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	Endpoint string
}

// Load reads configuration from environment variables
func Load() *Config {
	return &Config{
		Port: getEnv("PORT", "3000"),
		ClickHouse: ClickHouseConfig{
			Host:                   getEnv("CLICKHOUSE_HOST", "127.0.0.1"),
			Port:                   getEnv("CLICKHOUSE_PORT", "9000"),
			Database:               getEnv("CLICKHOUSE_DATABASE", "default"),
			User:                   getEnv("CLICKHOUSE_USER", "app"),
			Password:               getEnv("CLICKHOUSE_PASSWORD", "clickhouse_app_password"),
			AsyncInsertEnabled:     getEnv("CLICKHOUSE_ASYNC_INSERT_ENABLED", "1") == "1",
			AsyncInsertWait:        getEnvAsInt("CLICKHOUSE_ASYNC_INSERT_WAIT", 1),
			AsyncInsertMaxDataSize: getEnvAsInt64("CLICKHOUSE_ASYNC_INSERT_MAX_DATA_SIZE", 10485760),
			AsyncInsertBusyTimeout: getEnvAsInt("CLICKHOUSE_ASYNC_INSERT_BUSY_TIMEOUT", 200),
			RedisCacheDurationMS:   getEnvAsInt64("CLICKHOUSE_REDIS_CACHE_DURATION_MS", 60*60*1000),
			BufferChannelCapacity:  getEnvAsInt("EVENT_BUFFER_CAPACITY", 50000),
			BatchSize:              getEnvAsInt("EVENT_BATCH_SIZE", 5000),
			FlushIntervalSeconds:   getEnvAsInt("EVENT_FLUSH_INTERVAL_SECONDS", 1),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "127.0.0.1"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			Endpoint: getEnv("REDIS_ENDPOINT", ""),
		},
	}
}

func (c *ClickHouseConfig) GetClickHouseDSN() string {
	if c.DSN != "" {
		return c.DSN
	}

	// Build DSN from components
	dsn := "clickhouse://"
	if c.User != "" {
		dsn += c.User
		if c.Password != "" {
			dsn += ":" + c.Password
		}
		dsn += "@"
	}
	dsn += c.Host + ":" + c.Port + "/" + c.Database

	var queryParams []string

	if c.AsyncInsertEnabled {
		// Add async insert settings as query parameters
		// These settings apply to all queries on this connection
		asyncParams := []string{
			fmt.Sprintf("wait_for_async_insert=%d", c.AsyncInsertWait),
			fmt.Sprintf("async_insert_max_data_size=%d", c.AsyncInsertMaxDataSize),
			fmt.Sprintf("async_insert_busy_timeout_ms=%d", c.AsyncInsertBusyTimeout),
		}
		queryParams = append(queryParams, asyncParams...)
	}

	if len(queryParams) > 0 {
		dsn += "?" + queryParams[0]
		for i := 1; i < len(queryParams); i++ {
			dsn += "&" + queryParams[i]
		}
	}

	return dsn
}

func (r *RedisConfig) GetRedisAddr() string {
	if r.Endpoint != "" {
		return r.Endpoint
	}
	return r.Host + ":" + r.Port
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}
