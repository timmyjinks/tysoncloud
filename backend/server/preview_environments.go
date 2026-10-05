package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/gorilla/mux"
	"github.com/timmyjinks/tysoncloud/store"
	"github.com/timmyjinks/tysoncloud/util"
)

func (app *Application) buildPreviewEnvironmentServicesResponse(ctx context.Context, env store.PreviewEnvironment) PreviewEnvironmentServicesResponse {
	children, err := app.Supabase.GetPreviewEnvironmentServices(env.Id)
	if err != nil {
		slog.Warn("preview env children lookup failed", "env_id", env.Id, "err", err)
		children = nil
	}
	services := make([]PreviewEnvironmentServiceResponse, 0, len(children))
	for _, c := range children {
		src, err := app.Supabase.GetPreviewEnvironmentSource(c.SourceType, c.SourceServiceId)
		if err != nil {
			slog.Warn("preview source lookup failed", "env_id", env.Id, "source", c.SourceServiceId, "err", err)
			continue
		}
		hostname := util.PreviewHostname(c.SourceServiceId, env.Pr)
		effective, err := app.resolvePreviewEnvironmentServiceEnv(ctx, env, src, hostname)
		if err != nil {
			effective = map[string]string{}
		}
		services = append(services, toPreviewEnvironmentServiceResponse(src, env, c, hostname, effective, src.Status))
	}
	return PreviewEnvironmentServicesResponse{
		Id:        env.Id,
		ProjectId: env.ProjectId,
		Name:      env.Name,
		Pr:        env.Pr,
		Namespace: env.Namespace,
		Services:  services,
		CreatedAt: env.CreatedAt,
	}
}

func (app *Application) visiblePreviewEnvironment(projectId, envId string) (store.PreviewEnvironment, bool) {
	envs, _, err := app.Supabase.GetPreviewEnvironmentsForProject(projectId)
	if err != nil {
		return store.PreviewEnvironment{}, false
	}
	for _, env := range envs {
		if env.Id == envId {
			return env, true
		}
	}
	return store.PreviewEnvironment{}, false
}

func (app *Application) GetProjectPreviewEnvironments(w http.ResponseWriter, r *http.Request) {
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, msgUnauthorized, nil)
		return
	}
	projectId := mux.Vars(r)["project_id"]
	if projectId == "" {
		writeError(w, http.StatusBadRequest, "A project ID is required.", nil)
		return
	}
	if _, err := app.Supabase.GetProject(projectId, claims.Subject); err != nil {
		writeError(w, http.StatusNotFound, "We couldn't find that project.", err)
		return
	}
	envs, _, err := app.Supabase.GetPreviewEnvironmentsForProject(projectId)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Couldn't load the preview environments.", err)
		return
	}
	out := make([]PreviewEnvironmentResponse, 0, len(envs))
	for _, env := range envs {
		out = append(out, PreviewEnvironmentResponse{
			Id:        env.Id,
			ProjectId: env.ProjectId,
			Name:      env.Name,
			Pr:        env.Pr,
			Namespace: env.Namespace,
			CreatedAt: env.CreatedAt,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(out); err != nil {
		writeError(w, http.StatusInternalServerError, msgServerError, err)
	}
}

func (app *Application) GetProjectPreviewEnvironment(w http.ResponseWriter, r *http.Request) {
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, msgUnauthorized, nil)
		return
	}
	vars := mux.Vars(r)
	projectId := vars["project_id"]
	envId := vars["preview_environment_id"]
	if projectId == "" || envId == "" {
		writeError(w, http.StatusBadRequest, "A project ID and environment ID are required.", nil)
		return
	}
	if _, err := app.Supabase.GetProject(projectId, claims.Subject); err != nil {
		writeError(w, http.StatusNotFound, "We couldn't find that project.", err)
		return
	}
	env, ok := app.visiblePreviewEnvironment(projectId, envId)
	if !ok {
		writeError(w, http.StatusNotFound, "We couldn't find that preview environment.", nil)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(app.buildPreviewEnvironmentServicesResponse(r.Context(), env)); err != nil {
		writeError(w, http.StatusInternalServerError, msgServerError, err)
	}
}
