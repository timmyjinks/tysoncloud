package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/timmyjinks/tysoncloud/config"
)

func TestCORSMiddleware(t *testing.T) {
	app := &Application{Config: config.Config{Server: config.Server{AllowedOrigins: "https://app.example"}}}
	h := app.CORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	tests := []struct {
		name      string
		origin    string
		wantAllow string
	}{
		{name: "allowed origin reflected", origin: "https://app.example", wantAllow: "https://app.example"},
		{name: "unknown origin not reflected", origin: "https://evil.example", wantAllow: ""},
		{name: "localhost not allowed unless configured", origin: "http://localhost:3000", wantAllow: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/projects", nil)
			r.Header.Set("Origin", tt.origin)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if got := w.Header().Get("Access-Control-Allow-Origin"); got != tt.wantAllow {
				t.Fatalf("Allow-Origin = %q, want %q", got, tt.wantAllow)
			}
			if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "" {
				t.Fatalf("Allow-Credentials = %q, want unset", got)
			}
		})
	}
}

func TestParseAllowedOriginsDefaultsExcludeLocalhost(t *testing.T) {
	origins := parseAllowedOrigins("")
	if origins["http://localhost:3000"] {
		t.Fatal("default allowed origins must not include localhost")
	}
	if !origins["https://tysoncloud.tysonjenkins.dev"] {
		t.Fatal("default allowed origins should include the production frontend")
	}
}
