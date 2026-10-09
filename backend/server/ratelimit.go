package server

import (
	"net/http"
	"sync"
	"time"

	"github.com/clerk/clerk-sdk-go/v2"
	"golang.org/x/time/rate"
)

const rateLimiterIdleTTL = 10 * time.Minute

// rateLimiter hands out one token bucket per key (a Clerk user id) and
// forgets keys idle for rateLimiterIdleTTL.
type rateLimiter struct {
	mu        sync.Mutex
	limit     rate.Limit
	burst     int
	buckets   map[string]*bucket
	lastSweep time.Time
}

type bucket struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func newRateLimiter(limit rate.Limit, burst int) *rateLimiter {
	return &rateLimiter{limit: limit, burst: burst, buckets: map[string]*bucket{}, lastSweep: time.Now()}
}

func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	if now.Sub(rl.lastSweep) > rateLimiterIdleTTL {
		for k, b := range rl.buckets {
			if now.Sub(b.lastSeen) > rateLimiterIdleTTL {
				delete(rl.buckets, k)
			}
		}
		rl.lastSweep = now
	}

	b, ok := rl.buckets[key]
	if !ok {
		b = &bucket{limiter: rate.NewLimiter(rl.limit, rl.burst)}
		rl.buckets[key] = b
	}
	b.lastSeen = now
	return b.limiter.Allow()
}

func writeRateLimited(w http.ResponseWriter) {
	w.Header().Set("Retry-After", "5")
	writeError(w, http.StatusTooManyRequests, "You're making requests too quickly. Please wait a moment and try again.", nil)
}

// perUser limits authenticated requests per Clerk user. It must run after
// Clerk's header-auth middleware.
func (rl *rateLimiter) perUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := clerk.SessionClaimsFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, msgUnauthorized, nil)
			return
		}
		if !rl.allow(claims.Subject) {
			writeRateLimited(w)
			return
		}
		next.ServeHTTP(w, r)
	})
}
