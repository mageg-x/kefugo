package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"kefu-server/controllers"
	"kefu-server/utils/logger"
	"kefu-server/utils/response"
)

// rateLimitBucket 是每个来源 IP 对应的令牌桶状态。
type rateLimitBucket struct {
	tokens     float64
	lastRefill time.Time
	lastSeen   time.Time
}

var rateLimitState = struct {
	mu      sync.Mutex
	buckets map[string]*rateLimitBucket
}{
	buckets: make(map[string]*rateLimitBucket),
}

var customRateLimitState = struct {
	mu      sync.Mutex
	buckets map[string]*rateLimitBucket
}{
	buckets: make(map[string]*rateLimitBucket),
}

// RateLimitMiddleware 提供自定义参数的令牌桶限流中间件。
// rpm: 每分钟允许的请求数; burst: 突发上限。
func RateLimitMiddleware(rpm int, burst int) gin.HandlerFunc {
	if rpm <= 0 {
		rpm = 10
	}
	if burst <= 0 {
		burst = 5
	}
	refillPerSecond := float64(rpm) / 60.0
	return func(c *gin.Context) {
		key := strings.TrimSpace(c.ClientIP())
		if key == "" {
			key = "unknown"
		}
		now := time.Now()
		customRateLimitState.mu.Lock()
		bucket, ok := customRateLimitState.buckets[key]
		if !ok {
			bucket = &rateLimitBucket{
				tokens:     float64(burst),
				lastRefill: now,
				lastSeen:   now,
			}
			customRateLimitState.buckets[key] = bucket
		}
		elapsed := now.Sub(bucket.lastRefill).Seconds()
		if elapsed > 0 {
			bucket.tokens += elapsed * refillPerSecond
			if bucket.tokens > float64(burst) {
				bucket.tokens = float64(burst)
			}
			bucket.lastRefill = now
		}
		bucket.lastSeen = now
		if len(customRateLimitState.buckets) > 1024 {
			cutoff := now.Add(-10 * time.Minute)
			for ip, b := range customRateLimitState.buckets {
				if b == nil || b.lastSeen.Before(cutoff) {
					delete(customRateLimitState.buckets, ip)
				}
			}
		}
		allowed := bucket.tokens >= 1
		if allowed {
			bucket.tokens -= 1
		}
		customRateLimitState.mu.Unlock()
		if !allowed {
			logger.Errorf("custom rate limit exceeded ip=%s path=%s", key, c.Request.URL.Path)
			response.ResponseError(c, http.StatusTooManyRequests, response.ErrCodeSecurityRateLimitExceeded)
			c.Abort()
			return
		}
		c.Next()
	}
}

// RateLimit 提供基于令牌桶的接口限流中间件。
// 限流配置由系统设置控制，可动态开关与调整阈值。
func RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg := controllers.GetSystemSettingsForMiddleware()
		if !cfg.RateLimitEnabled {
			c.Next()
			return
		}

		rpm := cfg.RateLimitRPM
		burst := cfg.RateLimitBurst
		if rpm <= 0 {
			rpm = 120
		}
		if burst <= 0 {
			burst = 60
		}

		key := strings.TrimSpace(c.ClientIP())
		if key == "" {
			key = "unknown"
		}

		now := time.Now()
		refillPerSecond := float64(rpm) / 60.0

		rateLimitState.mu.Lock()
		bucket, ok := rateLimitState.buckets[key]
		if !ok {
			bucket = &rateLimitBucket{
				tokens:     float64(burst),
				lastRefill: now,
				lastSeen:   now,
			}
			rateLimitState.buckets[key] = bucket
		}

		elapsed := now.Sub(bucket.lastRefill).Seconds()
		if elapsed > 0 {
			bucket.tokens += elapsed * refillPerSecond
			if bucket.tokens > float64(burst) {
				bucket.tokens = float64(burst)
			}
			bucket.lastRefill = now
		}
		bucket.lastSeen = now

		// 清理长期未访问 IP，避免 map 持续膨胀。
		if len(rateLimitState.buckets) > 1024 {
			cutoff := now.Add(-10 * time.Minute)
			for ip, b := range rateLimitState.buckets {
				if b == nil || b.lastSeen.Before(cutoff) {
					delete(rateLimitState.buckets, ip)
				}
			}
		}

		allowed := bucket.tokens >= 1
		if allowed {
			bucket.tokens -= 1
		}
		rateLimitState.mu.Unlock()

		if !allowed {
			logger.Errorf("rate limit exceeded ip=%s path=%s", key, c.Request.URL.Path)
			response.ResponseError(c, http.StatusTooManyRequests, response.ErrCodeSecurityRateLimitExceeded)
			c.Abort()
			return
		}

		c.Next()
	}
}
