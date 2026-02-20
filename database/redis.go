package database

import (
	"context"
	"log"
	"splitwise-backend/config"

	"github.com/redis/go-redis/v9"
)

var Redis *redis.Client

func ConnectRedis() {
	Redis = redis.NewClient(&redis.Options{
		Addr: config.AppConfig.RedisURL,
	})

	_, err := Redis.Ping(context.Background()).Result()
	if err != nil {
		log.Println("⚠️  Redis not available, running without cache:", err)
		Redis = nil
		return
	}

	log.Println("✅ Redis connected successfully")
}
