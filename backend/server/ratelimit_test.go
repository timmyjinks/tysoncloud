package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/time/rate"
)

func TestRateLimiterPerKey(t *testing.T) {
	rl := newRateLimiter(rate.Limit(0), 2)

	if !rl.allow("alice") || !rl.allow("alice") {
		t.Fatal("burst requests should be allowed")
	}
	if rl.allow("alice") {
		t.Fatal("request beyond burst should be denied")
	}
	if !rl.allow("bob") {
		t.Fatal("other keys must have their own bucket")
	}
}

func TestRateLimiterPerUserRequiresSession(t *testing.T) {
	rl := newRateLimiter(rate.Inf, 1)
	h := rl.perUser(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/projects", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}
