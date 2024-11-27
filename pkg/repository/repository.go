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

func (r repository) Set(key string, value string) error {
	status := r.redisClient.Set(key, value, 0)
	return status.Err()
}

func (r repository) Get(key string) (string, error) {
	status := r.redisClient.Get(key)
	if status.Err() != nil {
		return "", status.Err()
	}

	return status.Result()
}
