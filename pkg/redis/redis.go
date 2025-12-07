package redis

import (
    "context"
    "fmt"
    "os"
    "time"

    "github.com/redis/go-redis/v9"
)

var ctx = context.Background()

type RedisClient struct {
    client *redis.Client
}

func Connect() (*RedisClient, error) {
    host := getEnv("REDIS_HOST", "localhost")
    port := getEnv("REDIS_PORT", "6379")
    password := getEnv("REDIS_PASSWORD", "")
    
    client := redis.NewClient(&redis.Options{
        Addr:     fmt.Sprintf("%s:%s", host, port),
        Password: password,
        DB:       0,
    })
    
    _, err := client.Ping(ctx).Result()
    if err != nil {
        return nil, fmt.Errorf("failed to connect to Redis: %w", err)
    }
    
    return &RedisClient{client: client}, nil
}

func (r *RedisClient) Close() error {
    return r.client.Close()
}

func (r *RedisClient) CheckRateLimit(key string, limit int, window time.Duration) (bool, int, error) {
    current, err := r.client.Get(ctx, key).Int()
    if err != nil && err != redis.Nil {
        return true, 0, err
    }
    
    if current >= limit {
        ttl, _ := r.client.TTL(ctx, key).Result()
        return false, int(ttl.Seconds()), nil
    }
    
    pipe := r.client.Pipeline()
    pipe.Incr(ctx, key)
    pipe.Expire(ctx, key, window)
    _, err = pipe.Exec(ctx)
    
    return true, 0, err
}

func (r *RedisClient) Set(key string, value interface{}, expiration time.Duration) error {
    return r.client.Set(ctx, key, value, expiration).Err()
}

func (r *RedisClient) Get(key string) (string, error) {
    return r.client.Get(ctx, key).Result()
}

func (r *RedisClient) Delete(key string) error {
    return r.client.Del(ctx, key).Err()
}

func getEnv(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}
