package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/clerk/clerk-sdk-go/v2"
	clerkjwt "github.com/clerk/clerk-sdk-go/v2/jwt"
	"github.com/gorilla/mux"
	"github.com/timmyjinks/tysoncloud/deploy"
	"github.com/timmyjinks/tysoncloud/store"
	"github.com/timmyjinks/tysoncloud/util"
)

func toPreviewEnvironmentServiceResponse(src store.PreviewEnvironmentSource, env store.PreviewEnvironment, child store.PreviewEnvironmentService, hostname string, liveEnv map[string]string, status string) PreviewEnvironmentServiceResponse {
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
	return PreviewEnvironmentServiceResponse{
		Id:              child.Id,
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

func (app *Application) resolvePreviewEnvironmentServiceEnv(ctx context.Context, env store.PreviewEnvironment, src store.PreviewEnvironmentSource, hostname string) (map[string]string, error) {
	previewName := util.PreviewResourceName(src.ResourceName, env.Pr)
	if live, err := app.Deploy.GetServiceEnv(ctx, deploy.Service{
		Namespace: env.Namespace,
		Name:      util.PreviewSecretName(src.Type, previewName),
	}); err == nil {
		effective := map[string]string{}
		for k, v := range live {
			effective[k] = v
		}
		if hostname != "" {
			effective[util.PreviewURLKey] = "https://" + hostname
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
		effective[util.PreviewURLKey] = "https://" + hostname
	}
	return effective, nil
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
	env, ok := app.visiblePreviewEnvironment(projectId, envId)
	if !ok {
		writeError(w, http.StatusNotFound, "We couldn't find that preview environment.", nil)
		return
	}

	var req PreviewEnvironmentServiceUpdateRequest
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
	src, err := app.Supabase.GetPreviewEnvironmentSource(child.SourceType, child.SourceServiceId)
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
	secretName := util.PreviewSecretName(src.Type, previewName)

	current, err := app.Deploy.GetServiceEnv(ctx, deploy.Service{
		Namespace: env.Namespace,
		Name:      secretName,
	})
	if err != nil {
		current, err = app.resolvePreviewEnvironmentServiceEnv(ctx, env, src, hostname)
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
		next[util.PreviewURLKey] = previewURL
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
		// Github copies have no image column in the DB. Reconstruct the exact
		// PR-built tag from stored fields (prod resource name + live PR head
		// SHA) so a port/env edit never rolls the preview back to :latest
		// and never reads the live deployment image.
		image = app.previewGithubImage(ctx, claims.Subject, src, env)
		if image == "" {
			image = app.Github.RegistryTag(app.Github.RegistryURL(), src.ResourceName, "latest")
		}
	}
	if image == "" {
		writeError(w, http.StatusInternalServerError, "Couldn't update the preview's port.", nil)
		return
	}
	envBytes := map[string][]byte{}
	for k, v := range current {
		envBytes[k] = []byte(v)
	}
	envBytes[util.PreviewURLKey] = []byte(previewURL)
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
	if err := json.NewEncoder(w).Encode(toPreviewEnvironmentServiceResponse(src, env, *child, hostname, current, src.Status)); err != nil {
		writeError(w, http.StatusInternalServerError, msgServerError, err)
	}
}

// previewGithubImage reconstructs the PR-built registry tag for a github
// preview copy from stored fields: the prod service's resource name plus the
// live PR head SHA. Empty string when it can't be resolved; callers fall back
// to the :latest tag.
func (app *Application) previewGithubImage(ctx context.Context, userId string, src store.PreviewEnvironmentSource, env store.PreviewEnvironment) string {
	baseName := strings.TrimSuffix(src.Name, fmt.Sprintf("-pr-%d", env.Pr))
	prods, err := app.Supabase.GetProductionGithubServices(src.ProjectId, userId)
	if err != nil {
		return ""
	}
	prodResourceName := ""
	for _, p := range prods {
		if p.Name == baseName {
			prodResourceName = p.ResourceName
			break
		}
	}
	if prodResourceName == "" {
		return ""
	}
	conn, err := app.Supabase.GetGithubConnection(userId)
	if err != nil {
		return ""
	}
	token, err := app.Github.GetInstallationToken(ctx, strconv.FormatInt(conn.InstallationId, 10))
	if err != nil || token == "" {
		return ""
	}
	sha, _, _ := app.fetchPRHead(ctx, token, src.RepoName, env.Pr)
	if strings.TrimSpace(sha) == "" {
		return ""
	}
	return app.Github.RegistryTag(app.Github.RegistryURL(), prodResourceName, fmt.Sprintf("pr-%d-%s", env.Pr, util.ShortSHA(sha)))
}

// DeletePreviewEnvironmentService removes a single preview copy: its infra in
// the preview namespace, its environment mapping and its copy row. Prod
// handlers stay prod-only; preview deletes go here.
func (app *Application) DeletePreviewEnvironmentService(w http.ResponseWriter, r *http.Request) {
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
	env, ok := app.visiblePreviewEnvironment(projectId, envId)
	if !ok {
		writeError(w, http.StatusNotFound, "We couldn't find that preview environment.", nil)
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
	src, err := app.Supabase.GetPreviewEnvironmentSource(child.SourceType, child.SourceServiceId)
	if err != nil {
		writeError(w, http.StatusNotFound, "We couldn't find that service in this preview environment.", err)
		return
	}

	ctx := r.Context()
	previewName := util.PreviewResourceName(src.ResourceName, env.Pr)

	_ = app.Supabase.DeletePreviewEnvironmentServicesBySource(sourceId)
	if err := app.Supabase.DeletePreviewCopy(claims.Subject, child.SourceType, sourceId); err != nil {
		writeError(w, http.StatusInternalServerError, "Couldn't delete the preview service.", err)
		return
	}

	if child.SourceType == "database" {
		if err := app.Deploy.DeleteDatabase(ctx, deploy.Database{
			Namespace: env.Namespace,
			Name:      previewName,
			Engine:    src.Engine,
		}); err != nil {
			slog.Warn("preview: database infra cleanup failed", "source", sourceId, "namespace", env.Namespace, "err", err)
		}
	} else if err := app.Deploy.DeleteService(ctx, deploy.Service{
		Namespace: env.Namespace,
		Name:      previewName,
	}); err != nil {
		slog.Warn("preview: service infra cleanup failed", "source", sourceId, "namespace", env.Namespace, "err", err)
	}

	if remaining, err := app.Supabase.GetPreviewEnvironmentServices(env.Id); err == nil && len(remaining) == 0 {
		if err := app.Deploy.DeleteProject(ctx, env.Namespace); err != nil {
			slog.Warn("preview: empty namespace cleanup failed", "namespace", env.Namespace, "err", err)
		}
		if err := app.Supabase.DeletePreviewEnvironment(env.Id); err != nil {
			slog.Warn("preview: empty env cleanup failed", "env_id", env.Id, "err", err)
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetPreviewEnvironmentServiceLogs streams logs for a single preview copy:
// build logs while pre-deploy, otherwise the workload logs in the preview
// namespace. Prod log handlers stay prod-only; preview logs go here.
func (app *Application) GetPreviewEnvironmentServiceLogs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	projectId := vars["project_id"]
	envId := vars["preview_environment_id"]
	sourceId := vars["source_service_id"]
	if projectId == "" || envId == "" || sourceId == "" {
		writeError(w, http.StatusBadRequest, "A project ID, environment ID and service ID are required.", nil)
		return
	}

	token := wsSessionToken(r)
	if token == "" {
		writeError(w, http.StatusUnauthorized, msgUnauthorized, nil)
		return
	}
	claims, err := clerkjwt.Verify(r.Context(), &clerkjwt.VerifyParams{Token: token})
	if err != nil {
		writeError(w, http.StatusUnauthorized, msgUnauthorized, err)
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
	if child.SourceType == "database" {
		writeError(w, http.StatusBadRequest, "Databases don't have log streams.", nil)
		return
	}
	src, err := app.Supabase.GetPreviewEnvironmentSource(child.SourceType, child.SourceServiceId)
	if err != nil {
		writeError(w, http.StatusNotFound, "We couldn't find that service in this preview environment.", err)
		return
	}

	upgrader := app.newUpgrader()
	ws, err := upgrader.Upgrade(w, r, http.Header{})
	if err != nil {
		slog.Error("preview log stream upgrade failed", "err", err)
		return
	}
	defer ws.Close()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	previewName := util.PreviewResourceName(src.ResourceName, env.Pr)
	hostname := util.PreviewHostname(child.SourceServiceId, env.Pr)
	status := strings.ToLower(strings.TrimSpace(src.Status))

	if child.SourceType == "github_service" && (status == "pending" || status == "building" || status == "deploying" || status == "") {
		buildLogID := util.PreviewBuildLogID(child.SourceServiceId, env.Pr)
		lines := make(chan string, 64)
		subID, ch, snap := app.Github.SubscribeBuildLogs(buildLogID)
		defer app.Github.UnsubscribeBuildLogs(buildLogID, subID)
		stateLine := `[state] ` + src.Status + ` service=` + src.Name + ` pr=` + strconv.Itoa(env.Pr) + ` domain=` + hostname + ` port=` + strconv.FormatInt(int64(src.Port), 10)
		go func() {
			defer close(lines)
			select {
			case lines <- stateLine:
			case <-ctx.Done():
				return
			}
			for _, l := range snap {
				select {
				case lines <- l:
				case <-ctx.Done():
					return
				}
			}
			for {
				select {
				case <-ctx.Done():
					return
				case line, ok := <-ch:
					if !ok {
						return
					}
					select {
					case lines <- line:
					case <-ctx.Done():
						return
					}
				}
			}
		}()
		for {
			select {
			case <-ctx.Done():
				return
			case line, ok := <-lines:
				if !ok {
					return
				}
				_ = ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := ws.WriteJSON(struct {
					Message string `json:"message"`
				}{Message: line}); err != nil {
					cancel()
					return
				}
			}
		}
	}

	lines := make(chan string)
	go func() {
		defer close(lines)
		svcRes := deploy.Service{
			Namespace: env.Namespace,
			Name:      previewName,
		}
		isDiagnostic := status == "pending" || status == "failed"
		var logErr error
		if isDiagnostic {
			logErr = app.Deploy.GetServiceDiagnosticLogs(ctx, svcRes, lines)
		} else {
			logErr = app.Deploy.GetServiceLogs(ctx, svcRes, lines)
		}
		if logErr != nil && ctx.Err() == nil {
			slog.Error("preview log stream failed", "source", sourceId, "status", src.Status, "err", logErr)
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case line, ok := <-lines:
			if !ok {
				return
			}
			_ = ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := ws.WriteJSON(struct {
				Message string `json:"message"`
			}{Message: line}); err != nil {
				cancel()
				return
			}
		}
	}
}
