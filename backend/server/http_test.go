package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/gorilla/mux"
	"github.com/timmyjinks/tysoncloud/config"
	gh "github.com/timmyjinks/tysoncloud/github"
)

func authCookie(name, value string) *http.Cookie {
	return &http.Cookie{Name: name, Value: value}
}

func TestExtractTokenFromCookie(t *testing.T) {
	session := base64.StdEncoding.EncodeToString([]byte(`{"access_token":"jwt-123"}`))

	tests := []struct {
		name    string
		cookie  *http.Cookie
		want    string
		wantErr bool
	}{
		{"base64- prefixed", authCookie("sb-proj-auth-token.0", "base64-"+session), "jwt-123", false},
		{"raw base64", authCookie("sb-proj-auth-token.0", session), "jwt-123", false},
		{"no cookie", nil, "", true},
		{"wrong cookie name", authCookie("session", session), "", true},
		{"bad base64", authCookie("sb-proj-auth-token.0", "base64-!!!"), "", true},
		{"bad json", authCookie("sb-proj-auth-token.0", base64.StdEncoding.EncodeToString([]byte("nope"))), "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}
			got, err := extractTokenFromCookie(req)
			if (err != nil) != tt.wantErr || got != tt.want {
				t.Fatalf("got (%q, %v), want (%q, err=%v)", got, err, tt.want, tt.wantErr)
			}
		})
	}
}

func TestSupabaseAuthMiddlewareRejectsMissingCookie(t *testing.T) {
	app := &Application{}
	h := app.SupabaseAuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("next handler called without auth")
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestProtectedRoutesRequireAuth(t *testing.T) {
	router := Mount(&Application{})

	routes := []struct{ method, path string }{
		{http.MethodGet, "/projects"},
		{http.MethodPost, "/projects"},
		{http.MethodGet, "/projects/p1"},
		{http.MethodPut, "/projects/p1"},
		{http.MethodDelete, "/projects/p1"},
		{http.MethodPost, "/projects/p1/services"},
		{http.MethodDelete, "/projects/p1/services/s1"},
		{http.MethodPost, "/projects/p1/databases"},
		{http.MethodPost, "/projects/p1/github_services"},
		{http.MethodPost, "/projects/p1/config"},
		{http.MethodGet, "/github/connections"},
		{http.MethodGet, "/github/app"},
		{http.MethodPut, "/projects/p1/preview_environments/e1/services/s1"},
	}
	for _, rt := range routes {
		t.Run(rt.method+" "+rt.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, httptest.NewRequest(rt.method, rt.path, nil))
			if rec.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want 403 without Authorization", rec.Code)
			}
		})
	}
}

func TestPreflightRoute(t *testing.T) {
	router := Mount(&Application{Config: config.Config{Server: config.Server{AllowedOrigins: "https://app.example"}}})
	req := httptest.NewRequest(http.MethodOptions, "/projects/p1/services", nil)
	req.Header.Set("Origin", "https://app.example")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent || rec.Header().Get("Access-Control-Allow-Origin") != "https://app.example" {
		t.Fatalf("preflight = %d %v", rec.Code, rec.Header())
	}
}

func signBody(secret, body string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(body))
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestGithubWebhookSignatureAndPing(t *testing.T) {
	app := &Application{Github: gh.NewService(config.Github{WebhookSecret: "whsec"}, config.Registry{})}
	router := Mount(app)
	body := `{"zen":"hi"}`

	t.Run("bad signature", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/webhooks/github", strings.NewReader(body))
		req.Header.Set("X-GitHub-Event", "ping")
		req.Header.Set("X-Hub-Signature-256", signBody("wrong", body))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
	})

	t.Run("missing signature", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/webhooks/github", strings.NewReader(body))
		req.Header.Set("X-GitHub-Event", "ping")
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", rec.Code)
		}
	})

	t.Run("valid ping", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/webhooks/github", strings.NewReader(body))
		req.Header.Set("X-GitHub-Event", "ping")
		req.Header.Set("X-Hub-Signature-256", signBody("whsec", body))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "pong") {
			t.Fatalf("ping = %d %q", rec.Code, rec.Body.String())
		}
	})

	t.Run("unhandled event is acknowledged", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/webhooks/github", strings.NewReader(body))
		req.Header.Set("X-GitHub-Event", "star")
		req.Header.Set("X-Hub-Signature-256", signBody("whsec", body))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
	})

	t.Run("push for deleted branch is ignored", func(t *testing.T) {
		push := `{"ref":"refs/heads/feature","deleted":true,"repository":{"id":1}}`
		req := httptest.NewRequest(http.MethodPost, "/webhooks/github", strings.NewReader(push))
		req.Header.Set("X-GitHub-Event", "push")
		req.Header.Set("X-Hub-Signature-256", signBody("whsec", push))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
	})

	t.Run("push for tag is ignored", func(t *testing.T) {
		push := `{"ref":"refs/tags/v1.0.0","repository":{"id":1}}`
		req := httptest.NewRequest(http.MethodPost, "/webhooks/github", strings.NewReader(push))
		req.Header.Set("X-GitHub-Event", "push")
		req.Header.Set("X-Hub-Signature-256", signBody("whsec", push))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
	})

	t.Run("push without repo id is rejected", func(t *testing.T) {
		push := `{"ref":"refs/heads/main","repository":{}}`
		req := httptest.NewRequest(http.MethodPost, "/webhooks/github", strings.NewReader(push))
		req.Header.Set("X-GitHub-Event", "push")
		req.Header.Set("X-Hub-Signature-256", signBody("whsec", push))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})
}

func TestConfigProjectValidation(t *testing.T) {
	app := &Application{}

	call := func(t *testing.T, body string, withClaims bool) (*httptest.ResponseRecorder, errorResponse) {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, "/projects/p1/config", strings.NewReader(body))
		req = mux.SetURLVars(req, map[string]string{"project_id": "p1"})
		if withClaims {
			claims := &clerk.SessionClaims{}
			claims.Subject = "user_1"
			req = req.WithContext(clerk.ContextWithSessionClaims(req.Context(), claims))
		}
		rec := httptest.NewRecorder()
		app.ConfigProject(rec, req)
		return rec, decodeError(t, rec)
	}

	t.Run("bad json", func(t *testing.T) {
		rec, _ := call(t, "{", true)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d", rec.Code)
		}
	})

	t.Run("no session", func(t *testing.T) {
		rec, _ := call(t, `{"content":""}`, false)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d", rec.Code)
		}
	})

	t.Run("toml syntax error has line", func(t *testing.T) {
		rec, body := call(t, `{"content":"[[services]]\nname = \"web\"\nport = \n"}`, true)
		if rec.Code != http.StatusBadRequest || len(body.Issues) != 1 || body.Issues[0].Line != 3 {
			t.Fatalf("got %d %+v", rec.Code, body)
		}
	})

	t.Run("semantic issues", func(t *testing.T) {
		rec, body := call(t, `{"content":"[[services]]\nname = \"web\"\n"}`, true)
		if rec.Code != http.StatusBadRequest || len(body.Issues) != 2 {
			t.Fatalf("got %d %+v", rec.Code, body)
		}
	})
}
