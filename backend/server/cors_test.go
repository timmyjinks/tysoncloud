package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/timmyjinks/tysoncloud/config"
)

func TestCORSMiddleware(t *testing.T) {
	app := &Application{Config: config.Config{Server: config.Server{AllowedOrigins: "https://app.example"}}}
	called := false
	h := app.CORSMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusTeapot)
	}))

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
			called = false
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
			if tt.wantAllow != "" && w.Header().Get("Vary") != "Origin" {
				t.Fatalf("Vary = %q, want Origin", w.Header().Get("Vary"))
			}
			if !called || w.Code != http.StatusTeapot {
				t.Fatalf("next handler not called (code %d)", w.Code)
			}
		})
	}

	t.Run("preflight short-circuits", func(t *testing.T) {
		called = false
		r := httptest.NewRequest(http.MethodOptions, "/projects", nil)
		r.Header.Set("Origin", "https://app.example")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusNoContent || called {
			t.Fatalf("preflight code = %d, called = %v", w.Code, called)
		}
		if got := w.Header().Get("Access-Control-Allow-Methods"); got != corsAllowedMethods {
			t.Fatalf("Allow-Methods = %q", got)
		}
	})
}

func TestParseAllowedOrigins(t *testing.T) {
	if got := parseAllowedOrigins("  "); len(got) != len(defaultAllowedOrigins) {
		t.Fatalf("blank config should use defaults, got %v", got)
	}

	got := parseAllowedOrigins(" https://a.example , ,https://b.example")
	if len(got) != 2 || !got["https://a.example"] || !got["https://b.example"] {
		t.Fatalf("parsed = %v", got)
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
