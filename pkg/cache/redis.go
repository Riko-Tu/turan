package cache

import (
	"TEFS-BE/pkg/log"
	"fmt"
	"github.com/go-redis/redis"
	"github.com/spf13/viper"
	"time"
)

var (
	redisClient *redis.Client
	RedisNilError = redis.Nil
)

// Setup redis connection
func Setup() {
	redisClient = redis.NewClient(&redis.Options{
		Addr:         viper.GetString("redis.host"),
		Password:     viper.GetString("redis.auth"),
		DB:           viper.GetInt("redis.db"),
		PoolSize:     viper.GetInt("redis.pool.max"),
		MinIdleConns: viper.GetInt("redis.pool.min"),
	})
	_, err := redisClient.Ping().Result()
	if err != nil {
		log.Fatal(fmt.Sprintf("Could not connected to redis : %s", err.Error()))
	}
	log.Info("Successfully connected to redis")
}

func GetRedis() *redis.Client {
	return redisClient
}

// Get from key
func Get(key string) (string, error) {
	return redisClient.Get(key).Result()
}

// Set value with key and expire time
func Set(key string, val string, expire int) error {
	return redisClient.Set(key, val, time.Duration(expire)).Err()
}

// Del delete key in redis
func Del(key string) error {
	return redisClient.Del(key).Err()
}

// HashGet from key
func HashGet(hk, key string) (string, error) {
	return redisClient.HGet(hk, key).Result()
}

// HashDel delete key in specify redis's hashtable
func HashDel(hk, key string) error {
	return redisClient.HDel(hk, key).Err()
}

// Increase
func Increase(key string) error {
	return redisClient.Incr(key).Err()
}

// Set ttl
func Expire(key string, dur time.Duration) error {
	return redisClient.Expire(key, dur).Err()
}
