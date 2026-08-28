package server

import (
	"encoding/json"
	"net/http"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/gorilla/mux"
)

func (app *Application) GetGithubConnections(w http.ResponseWriter, r *http.Request) {
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, msgUnauthorized, nil)
		return
	}

	// Single connection per user in current schema; wrap as array for extensibility.
	conn, err := app.Supabase.GetGithubConnection(claims.Subject)
	if err != nil {
		// No connection → empty list (not 404) so UI can show install CTA.
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]interface{}{})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode([]interface{}{conn})
}

func (app *Application) CreateGithubConnection(w http.ResponseWriter, r *http.Request) {
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, msgUnauthorized, nil)
		return
	}

	var req struct {
		InstallationId string `json:"installation_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "That request wasn't valid.", err)
		return
	}
	if req.InstallationId == "" {
		writeError(w, http.StatusBadRequest, "installation_id is required.", nil)
		return
	}

	// Idempotent: if already exists for this user, return existing.
	if existing, err := app.Supabase.GetGithubConnection(claims.Subject); err == nil && existing.Id != "" {
		// If same installation, return 200 with existing; if different, replace.
		if existing.InstallationId == req.InstallationId {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(existing)
			return
		}
		// Different installation for same user — delete old, create new.
		_ = app.Supabase.DeleteGithubConnection(existing.Id, claims.Subject)
	}

	if err := app.Supabase.CreateGithubConnection(claims.Subject, req.InstallationId); err != nil {
		writeError(w, http.StatusInternalServerError, "Couldn't save the GitHub connection.", err)
		return
	}

	conn, err := app.Supabase.GetGithubConnection(claims.Subject)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "GitHub connection was saved, but we couldn't confirm it. Please refresh.", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(conn)
}

func (app *Application) DeleteGithubConnection(w http.ResponseWriter, r *http.Request) {
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, msgUnauthorized, nil)
		return
	}
	connectionId := mux.Vars(r)["connection_id"]
	if connectionId == "" {
		writeError(w, http.StatusBadRequest, "A connection ID is required.", nil)
		return
	}

	if err := app.Supabase.DeleteGithubConnection(connectionId, claims.Subject); err != nil {
		writeError(w, http.StatusInternalServerError, "Couldn't remove the GitHub connection.", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (app *Application) GetGithubApp(w http.ResponseWriter, r *http.Request) {
	slug := app.Config.Server.GithubAppSlug
	installURL := ""
	if slug != "" {
		installURL = "https://github.com/apps/" + slug + "/installations/new"
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"slug":        slug,
		"install_url": installURL,
	})
}
