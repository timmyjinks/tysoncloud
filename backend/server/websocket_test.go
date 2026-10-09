package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWsSessionToken(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		cookie   string
		query    string
		want     string
	}{
		{name: "subprotocol", protocol: "bearer, eyJhbGciOi.abc.def", want: "eyJhbGciOi.abc.def"},
		{name: "cookie fallback", cookie: "cookie-token", want: "cookie-token"},
		{name: "subprotocol wins over cookie", protocol: "bearer, proto-token", cookie: "cookie-token", want: "proto-token"},
		{name: "bearer without token", protocol: "bearer", want: ""},
		{name: "query string ignored", query: "?token=leaky", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/logs"+tt.query, nil)
			if tt.protocol != "" {
				r.Header.Set("Sec-WebSocket-Protocol", tt.protocol)
			}
			if tt.cookie != "" {
				r.AddCookie(&http.Cookie{Name: "__session", Value: tt.cookie})
			}
			if got := wsSessionToken(r); got != tt.want {
				t.Fatalf("wsSessionToken() = %q, want %q", got, tt.want)
			}
		})
	}
}
