package router

import (
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimiter is an in-memory, per client IP, token bucket rate limiter.
//
// Client IPs come from gin's ClientIP, so X-Forwarded-For is only trusted
// from the proxies configured with TRUSTED_PROXIES.
type RateLimiter struct {
	mu        sync.Mutex
	clients   map[string]*limiterEntry
	every     time.Duration
	burst     int
	lastPrune time.Time
}

type limiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// NewRateLimiter allows `burst` requests at once, refilling one request
// every `every`. So (12s, 5) allows 5 attempts, then 5 per minute.
func NewRateLimiter(every time.Duration, burst int) *RateLimiter {
	return &RateLimiter{
		clients: map[string]*limiterEntry{},
		every:   every,
		burst:   burst,
	}
}

// Allow reports if the client identified by key may make a request now.
func (l *RateLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	// Forget clients that have been idle long enough to have a full bucket.
	if now.Sub(l.lastPrune) > time.Minute {
		idle := l.every * time.Duration(l.burst)
		for k, e := range l.clients {
			if now.Sub(e.lastSeen) > idle {
				delete(l.clients, k)
			}
		}
		l.lastPrune = now
	}
	e, ok := l.clients[key]
	if !ok {
		e = &limiterEntry{limiter: rate.NewLimiter(rate.Every(l.every), l.burst)}
		l.clients[key] = e
	}
	e.lastSeen = now
	return e.limiter.Allow()
}

// Middleware aborts with 429 when the client is over the limit.
func (l *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !l.Allow(c.ClientIP()) {
			slog.Warn("Rate limit exceeded", "ip", c.ClientIP(), "path", c.FullPath())
			c.AbortWithStatusJSON(http.StatusTooManyRequests, ErrorResponse{Error: "too many requests, try again later"})
			return
		}
		c.Next()
	}
}
