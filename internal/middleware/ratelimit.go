package middleware

import (
    "net/http"
    "time"

    "isp-saas.com/platform/pkg/redis"
)

type RateLimiter struct {
    redis  *redis.RedisClient
    limit  int
    window time.Duration
}

func NewRateLimiter(redisClient *redis.RedisClient, limit int, window time.Duration) *RateLimiter {
    return &RateLimiter{
        redis:  redisClient,
        limit:  limit,
        window: window,
    }
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if rl.redis == nil {
            next.ServeHTTP(w, r)
            return
        }

        ip := r.RemoteAddr
        if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
            ip = forwarded
        }

        key := "ratelimit:" + ip

        allowed, retryAfter, err := rl.redis.CheckRateLimit(key, rl.limit, rl.window)
        if err != nil {
            next.ServeHTTP(w, r)
            return
        }

        if !allowed {
            w.Header().Set("Content-Type", "application/json")
            w.Header().Set("Retry-After", string(rune(retryAfter)))
            w.WriteHeader(http.StatusTooManyRequests)
            w.Write([]byte(`{"success":false,"error":"Rate limit exceeded. Please try again later."}`))
            return
        }

        next.ServeHTTP(w, r)
    })
}
