package database

import (
	"context"
	"log"
	"quickdesk/signaling/internal/config"

	"github.com/redis/go-redis/v9"
)

// InitRedis initializes the Redis client
func InitRedis(cfg *config.Config) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr(),
		Password: cfg.Redis.Password,
		DB:       0, // use default DB
	})

	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	log.Printf("Successfully connected to Redis at %s", cfg.Redis.Addr())

	return client
}
