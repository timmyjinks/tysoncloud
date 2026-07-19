package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/supabase-community/auth-go/types"
	"github.com/supabase-community/supabase-go"
)

type contextKey string

const supabaseClientKey contextKey = "supabaseClient"

func (app *Application) SupabaseAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userJWT, err := extractTokenFromCookie(r)
		if err != nil {
			http.Error(w, "missing or invalid session", http.StatusUnauthorized)
			return
		}

		client, err := supabase.NewClient(
			app.Config.Supabase.ProjectURL,
			app.Config.Supabase.APIKey,
			&supabase.ClientOptions{},
		)
		if err != nil {
			http.Error(w, "server error", http.StatusInternalServerError)
			return
		}

		client.UpdateAuthSession(types.Session{AccessToken: userJWT})

		ctx := context.WithValue(r.Context(), supabaseClientKey, client)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func ClientFromContext(ctx context.Context) *supabase.Client {
	client, _ := ctx.Value(supabaseClientKey).(*supabase.Client)
	return client
}

func extractTokenFromCookie(r *http.Request) (string, error) {
	cookie, err := r.Cookie("sb-127-auth-token.0")
	if err != nil {
		return "", err
	}

	val := cookie.Value

	// Supabase SSR cookies are often base64-encoded, sometimes prefixed with "base64-"
	val = strings.TrimPrefix(val, "base64-")

	decoded, err := base64.StdEncoding.DecodeString(val)
	if err != nil {
		return "", err
	}

	var session struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(decoded, &session); err != nil {
		return "", err
	}

	return session.AccessToken, nil
}
