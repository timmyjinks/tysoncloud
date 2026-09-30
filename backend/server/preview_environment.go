package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/gorilla/mux"
	"github.com/timmyjinks/tysoncloud/deploy"
	"github.com/timmyjinks/tysoncloud/store"
	"github.com/timmyjinks/tysoncloud/util"
)

const PreviewURLKey = "PREVIEW_URL"

func toPreviewResponse(src store.PreviewSource, env store.PreviewEnvironment, child store.PreviewEnvironmentService, hostname string, liveEnv map[string]string, status string) PreviewEnvironmentResponse {
	previewURL := ""
	if hostname != "" {
		previewURL = "https://" + hostname
	}
	if liveEnv == nil {
		liveEnv = map[string]string{}
	}
	prURL := ""
	if src.RepoName != "" {
		prURL = "https://github.com/" + src.RepoName + "/pull/" + strconv.Itoa(env.Pr)
	}
	return PreviewEnvironmentResponse{
		Id:              strconv.FormatInt(child.Id, 10),
		PreviewEnvId:    env.Id,
		EnvName:         env.Name,
		SourceType:      child.SourceType,
		SourceServiceId: child.SourceServiceId,
		GithubServiceId: child.SourceServiceId,
		ServiceName:     src.Name,
		RepoName:        src.RepoName,
		Engine:          src.Engine,
		Pr:              env.Pr,
		PrURL:           prURL,
		Branch:          src.Branch,
		Hostname:        hostname,
		URL:             previewURL,
		Env:             liveEnv,
		PreviewURL:      previewURL,
		Port:            src.Port,
		Status:          status,
		CreatedAt:       child.CreatedAt,
	}
}

func previewSecretName(src store.PreviewSource, previewName string) string {
	if src.Type == "database" {
		return previewName + "-app"
	}
	return previewName
}

// mergePreviewEnv preserves preview edits across redeploys: start from prod,
// overlay live preview keys, force PREVIEW_URL. First deploy (live==nil/empty)
// is just prod + PREVIEW_URL.
func mergePreviewEnv(prod, live map[string]string, previewURL string) map[string][]byte {
	merged := map[string][]byte{}
	for k, v := range prod {
		merged[k] = []byte(v)
	}
	for k, v := range live {
		if k == PreviewURLKey {
			continue
		}
		merged[k] = []byte(v)
	}
	if previewURL != "" {
		merged[PreviewURLKey] = []byte(previewURL)
	}
	return merged
}

func (app *Application) resolvePreviewEnv(ctx context.Context, env store.PreviewEnvironment, src store.PreviewSource, hostname string) (map[string]string, error) {
	previewName := util.PreviewResourceName(src.ResourceName, env.Pr)
	if live, err := app.Deploy.GetServiceEnv(ctx, deploy.Service{
		Namespace: env.Namespace,
		Name:      previewSecretName(src, previewName),
	}); err == nil {
		effective := map[string]string{}
		for k, v := range live {
			effective[k] = v
		}
		if hostname != "" {
			effective[PreviewURLKey] = "https://" + hostname
		}
		return effective, nil
	}
	secretName := src.ResourceName
	if src.Type == "database" {
		secretName = src.ResourceName + "-app"
	}
	parent, err := app.Deploy.GetServiceEnv(ctx, deploy.Service{
		Namespace: "proj-" + src.ProjectId,
		Name:      secretName,
	})
	if err != nil {
		return nil, err
	}
	effective := map[string]string{}
	for k, v := range parent {
		effective[k] = v
	}
	if hostname != "" {
		effective[PreviewURLKey] = "https://" + hostname
	}
	return effective, nil
}

func (app *Application) buildPreviewDetail(ctx context.Context, env store.PreviewEnvironment) PreviewEnvListItem {
	children, err := app.Supabase.GetPreviewEnvironmentServices(env.Id)
	if err != nil {
		slog.Warn("preview env children lookup failed", "env_id", env.Id, "err", err)
		children = nil
	}
	services := make([]PreviewEnvironmentResponse, 0, len(children))
	for _, c := range children {
		src, err := app.Supabase.GetPreviewSource(c.SourceType, c.SourceServiceId)
		if err != nil {
			slog.Warn("preview source lookup failed", "env_id", env.Id, "source", c.SourceServiceId, "err", err)
			continue
		}
		hostname := util.PreviewHostname(c.SourceServiceId, env.Pr)
		effective, err := app.resolvePreviewEnv(ctx, env, src, hostname)
		if err != nil {
			effective = map[string]string{}
		}
		services = append(services, toPreviewResponse(src, env, c, hostname, effective, src.Status))
	}
	return PreviewEnvListItem{
		Id:        env.Id,
		ProjectId: env.ProjectId,
		Name:      env.Name,
		Pr:        env.Pr,
		Namespace: env.Namespace,
		Services:  services,
		CreatedAt: env.CreatedAt,
	}
}

func (app *Application) visiblePreviewEnv(projectId, envId string) (store.PreviewEnvironment, bool) {
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

func filterPreviewCopies[T any](items []T, idOf func(T) string, copies map[string]bool) []T {
	kept := make([]T, 0, len(items))
	for _, it := range items {
		if copies[idOf(it)] {
			continue
		}
		kept = append(kept, it)
	}
	return kept
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
	out := make([]PreviewEnvSummary, 0, len(envs))
	for _, env := range envs {
		out = append(out, PreviewEnvSummary{
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
	env, ok := app.visiblePreviewEnv(projectId, envId)
	if !ok {
		writeError(w, http.StatusNotFound, "We couldn't find that preview environment.", nil)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(app.buildPreviewDetail(r.Context(), env)); err != nil {
		writeError(w, http.StatusInternalServerError, msgServerError, err)
	}
}

func (app *Application) UpdatePreviewEnvironmentService(w http.ResponseWriter, r *http.Request) {
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, msgUnauthorized, nil)
		return
	}
	vars := mux.Vars(r)
	projectId := vars["project_id"]
	envId := vars["preview_environment_id"]
	sourceId := vars["source_service_id"]
	if projectId == "" || envId == "" || sourceId == "" {
		writeError(w, http.StatusBadRequest, "A project ID, environment ID and service ID are required.", nil)
		return
	}
	if _, err := app.Supabase.GetProject(projectId, claims.Subject); err != nil {
		writeError(w, http.StatusNotFound, "We couldn't find that project.", err)
		return
	}
	env, ok := app.visiblePreviewEnv(projectId, envId)
	if !ok {
		writeError(w, http.StatusNotFound, "We couldn't find that preview environment.", nil)
		return
	}

	var req PreviewServiceUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "That preview request wasn't valid.", err)
		return
	}
	if req.Env == nil && req.Port == nil {
		writeError(w, http.StatusBadRequest, "Nothing to update. Provide environment variables or a port.", nil)
		return
	}
	if req.Env != nil {
		if valid, err := util.ValidateEnv(*req.Env); err != nil || !valid {
			writeError(w, http.StatusBadRequest, "Environment variables must be one KEY=value pair per line.", err)
			return
		}
	}
	if req.Port != nil && (*req.Port <= 0 || *req.Port > 65535) {
		writeError(w, http.StatusBadRequest, "Port must be between 1 and 65535.", nil)
		return
	}

	children, err := app.Supabase.GetPreviewEnvironmentServices(env.Id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Couldn't load the preview environment.", err)
		return
	}
	var child *store.PreviewEnvironmentService
	for i := range children {
		if children[i].SourceServiceId == sourceId {
			child = &children[i]
			break
		}
	}
	if child == nil {
		writeError(w, http.StatusNotFound, "We couldn't find that service in this preview environment.", nil)
		return
	}
	src, err := app.Supabase.GetPreviewSource(child.SourceType, child.SourceServiceId)
	if err != nil {
		writeError(w, http.StatusNotFound, "We couldn't find that service in this preview environment.", err)
		return
	}
	if src.Type == "database" {
		writeError(w, http.StatusBadRequest, "Databases can't be edited in preview environments.", nil)
		return
	}

	ctx := r.Context()
	previewName := util.PreviewResourceName(src.ResourceName, env.Pr)
	hostname := util.PreviewHostname(child.SourceServiceId, env.Pr)
	previewURL := "https://" + hostname
	secretName := previewSecretName(src, previewName)

	current, err := app.Deploy.GetServiceEnv(ctx, deploy.Service{
		Namespace: env.Namespace,
		Name:      secretName,
	})
	if err != nil {
		current, err = app.resolvePreviewEnv(ctx, env, src, hostname)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Couldn't load the preview's current configuration.", err)
			return
		}
	}

	if req.Env != nil {
		next := map[string]string{}
		for k, v := range util.ParseEnv(*req.Env) {
			next[k] = string(v)
		}
		next[PreviewURLKey] = previewURL
		current = next
	}

	port := src.Port
	if req.Port != nil {
		port = *req.Port
	}
	// Same as normal services: deploy with the image from the source of
	// truth (prod), not the stale live deployment — the image change is
	// what rolls the pods. Port stays preview-owned.
	image := ""
	if child.SourceType == "service" {
		baseName := strings.TrimSuffix(src.Name, fmt.Sprintf("-pr-%d", env.Pr))
		if prods, perr := app.Supabase.GetProductionServicesByProjectId(src.ProjectId); perr == nil {
			for _, p := range prods {
				if p.Name == baseName {
					image = p.Image
					break
				}
			}
		}
		if image != "" {
			if copyRow, cerr := app.Supabase.GetServiceById(child.SourceServiceId); cerr == nil && copyRow.Image != image {
				if updated, uerr := app.Supabase.UpdateServiceImageById(child.SourceServiceId, image); uerr == nil {
					_ = updated
				} else {
					slog.Warn("preview: copy row image update failed", "source", child.SourceServiceId, "err", uerr)
				}
			}
		} else if copyRow, cerr := app.Supabase.GetServiceById(child.SourceServiceId); cerr == nil {
			image = copyRow.Image
		}
	} else {
		// Github copies have no image column — the live deployment runs
		// the PR-built tag, which is the newest image. Keep it.
		image, err = app.Deploy.GetServiceImage(ctx, env.Namespace, previewName)
	}
	if image == "" {
		writeError(w, http.StatusInternalServerError, "Couldn't update the preview's port.", err)
		return
	}
	envBytes := map[string][]byte{}
	for k, v := range current {
		envBytes[k] = []byte(v)
	}
	envBytes[PreviewURLKey] = []byte(previewURL)
	if err := app.Deploy.CreateService(ctx, deploy.Service{
		Namespace: env.Namespace,
		Name:      previewName,
		Hostname:  hostname,
		Port:      port,
		Image:     image,
		Env:       envBytes,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "We saved your changes, but couldn't restart the preview. A refresh will show its current status.", err)
		return
	}
	if port != src.Port {
		src.Port = port
		var perr error
		if child.SourceType == "service" {
			_, perr = app.Supabase.UpdateServicePortById(child.SourceServiceId, port)
		} else {
			_, perr = app.Supabase.UpdateGithubServicePortById(child.SourceServiceId, port)
		}
		if perr != nil {
			slog.Warn("preview: copy row port update failed", "source", child.SourceServiceId, "port", port, "err", perr)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(toPreviewResponse(src, env, *child, hostname, current, src.Status)); err != nil {
		writeError(w, http.StatusInternalServerError, msgServerError, err)
	}
}
