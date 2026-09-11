package src

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client

const OptimizationTTL = 30 * time.Minute

func InitRedis() error {
	addr := os.Getenv("REDIS_ADDR")
	if addr == "" {
		addr = "localhost:6379"
	}

	RedisClient = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: os.Getenv("REDIS_PASSWORD"),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := RedisClient.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %v", err)
	}

	return nil
}

//Is optimization key needed? Can a simple string be used instead? As long as the GET refers to the same one?

/* func OptimizationCacheKey(input OptimizationInput) (string, error) {
	data, err := json.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("failed to marshal input for cache key: %v", err)
	}
	hash := sha256.Sum256(data)
	return fmt.Sprintf("opt:%x", hash), nil
} */

func SaveOptimizationResult(key string, result OptimizationResultForSequence, ttl time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal optimization result: %v", err)
	}

	return RedisClient.Set(ctx, key, data, ttl).Err()
}

func SaveTPResponse(key string, response techProviderResponse, ttl time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	data, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal optimization result: %v", err)
	}

	return RedisClient.Set(ctx, key, data, ttl).Err()
}

func GetOptimizationResult(key string) (*OptimizationResultForSequence, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	data, err := RedisClient.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get optimization result from cache: %v", err)
	}

	var result OptimizationResultForSequence
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached result: %v", err)
	}

	return &result, nil
}

func GetTPData(key string) (*techProviderResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	data, err := RedisClient.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get optimization result from cache: %v", err)
	}

	var result techProviderResponse
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached result: %v", err)
	}

	return &result, nil
}
