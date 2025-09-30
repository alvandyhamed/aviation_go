package httpx

import (
	"net/http"
	"sync"
	"time"
)

// Token bucket ساده: 29 توکن/دقیقه، هر ~2.07 ثانیه 1 توکن جدید
type RateLimiter struct {
	mu     sync.Mutex
	tokens int
	max    int
	tick   *time.Ticker
}

func NewRateLimiter(maxPerMinute int) *RateLimiter {
	rl := &RateLimiter{
		max:    maxPerMinute,
		tokens: maxPerMinute,
		// هر دقیقه / max → فاصله‌ی شارژ
		tick: time.NewTicker(time.Minute / time.Duration(maxPerMinute)),
	}
	go func() {
		for range rl.tick.C {
			rl.mu.Lock()
			if rl.tokens < rl.max {
				rl.tokens++
			}
			rl.mu.Unlock()
		}
	}()
	return rl
}

func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if rl.tokens > 0 {
		rl.tokens--
		return true
	}
	return false
}

// میدل‌ویر
func LimitMiddleware(rl *RateLimiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.Allow() {
			w.Header().Set("Retry-After", "60")
			http.Error(w, `{"error":"rate limit: 29 req/min"}`, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
