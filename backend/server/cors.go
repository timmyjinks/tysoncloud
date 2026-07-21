package server

import (
	"net/http"
	"strings"
)

const (
	corsAllowedMethods = "GET, POST, PUT, DELETE, OPTIONS"
	corsAllowedHeaders = "Content-Type, Authorization"
)

var defaultAllowedOrigins = []string{
	"https://status.tysonjenkins.dev",
	"https://tysoncloud.tysonjenkins.dev",
	"https://tysoncloud-test.tysonjenkins.dev",
	"http://localhost:3000",
}

func parseAllowedOrigins(raw string) map[string]bool {
	origins := map[string]bool{}

	if strings.TrimSpace(raw) == "" {
		for _, o := range defaultAllowedOrigins {
			origins[o] = true
		}
		return origins
	}

	for _, o := range strings.Split(raw, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			origins[o] = true
		}
	}
	return origins
}

func (app *Application) CORSMiddleware(next http.Handler) http.Handler {
	allowed := parseAllowedOrigins(app.Config.Server.AllowedOrigins)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && allowed[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Add("Vary", "Origin")
		}

		w.Header().Set("Access-Control-Allow-Methods", corsAllowedMethods)
		w.Header().Set("Access-Control-Allow-Headers", corsAllowedHeaders)

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
