package server

import (
	"encoding/json"
	"log/slog"
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

	conn, err := app.Supabase.GetGithubConnection(claims.Subject)
	if err != nil {
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

	var req GithubConnectionCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "That request wasn't valid.", err)
		return
	}
	if req.InstallationId == 0 {
		writeError(w, http.StatusBadRequest, "installation_id is required.", nil)
		return
	}

	if existing, err := app.Supabase.GetGithubConnection(claims.Subject); err == nil && existing.Id != "" {
		if existing.InstallationId == req.InstallationId {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(existing)
			return
		}
		_ = app.Supabase.DeleteGithubConnection(existing.Id, claims.Subject)
	}

	if err := app.Supabase.CreateGithubConnection(claims.Subject, req.InstallationId); err != nil {
		slog.Error("CreateGithubConnection failed", "user_id", claims.Subject, "installation_id", req.InstallationId, "err", err)
		writeError(w, http.StatusInternalServerError, "Couldn't save the GitHub connection.", err)
		return
	}
	slog.Info("CreateGithubConnection success", "user_id", claims.Subject, "installation_id", req.InstallationId)

	conn, err := app.Supabase.GetGithubConnection(claims.Subject)
	if err != nil {
		slog.Error("GetGithubConnection after create failed", "user_id", claims.Subject, "err", err)
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
	slug := app.Config.Github.AppSlug
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
