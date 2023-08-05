package repository

import (
	"github.com/go-redis/redis"
)

type repository struct {
	redisClient *redis.Client
}

func NewRepository(redisClient *redis.Client) repository {
	return repository{redisClient: redisClient}
}

func (r repository) Set(key string, value string) {
	_ = r.redisClient.Set(key, value, 0)
}

func (r repository) Get(key string) string {
	value, _ := r.redisClient.Get(key).Result()
	return value
}
