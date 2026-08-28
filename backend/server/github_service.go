package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/gorilla/mux"
	"github.com/timmyjinks/tysoncloud/deploy"
	"github.com/timmyjinks/tysoncloud/store"
	"github.com/timmyjinks/tysoncloud/util"
)

func (app *Application) GithubRepos(w http.ResponseWriter, r *http.Request) {
	installationId := mux.Vars(r)["installation_id"]
	if installationId == "" {
		writeError(w, http.StatusBadRequest, "An installation ID is required.", nil)
		return
	}

	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, msgUnauthorized, nil)
		return
	}

	// Verify user owns this installation
	conn, err := app.Supabase.GetGithubConnectionByInstallationId(installationId)
	if err != nil {
		writeError(w, http.StatusNotFound, "We couldn't find that GitHub installation.", err)
		return
	}
	if conn.UserId != claims.Subject {
		writeError(w, http.StatusForbidden, "You don't have access to that installation.", nil)
		return
	}

	// NOTE: In production, exchange installationId for an installation access token via GitHub App JWT.
	// For local dev, attempt to fetch token; fallback returns 501 if not configured.
	token, err := app.getInstallationToken(r.Context(), installationId)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "GitHub installation token not configured.", err)
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), "GET", "https://api.github.com/installation/repositories", nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, msgServerError, err)
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		writeError(w, http.StatusBadGateway, "Couldn't reach GitHub.", err)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		slog.Error("failed to proxy github repos response", "err", err)
	}
}

func (app *Application) GetGithubService(w http.ResponseWriter, r *http.Request) {
	githubServiceId := mux.Vars(r)["github_service_id"]
	if githubServiceId == "" {
		writeError(w, http.StatusBadRequest, "A service ID is required.", nil)
		return
	}

	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, msgUnauthorized, nil)
		return
	}

	svc, err := app.Supabase.GetGithubService(githubServiceId, claims.Subject)
	if err != nil {
		writeError(w, http.StatusNotFound, "We couldn't find that service.", err)
		return
	}

	env, err := app.Deploy.GetServiceEnv(r.Context(), deploy.Service{
		Namespace: "proj-" + svc.ProjectId,
		Name:      svc.ResourceName,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Couldn't load the service's environment variables.", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(GithubServiceResponse{
		Id:             svc.Id,
		ProjectId:      svc.ProjectId,
		Name:           svc.Name,
		Repo:           svc.Repo,
		RepoId:         svc.RepoId,
		RootDir:        svc.RootDir,
		Port:           svc.Port,
		Status:         svc.Status,
		PublicDomain:   svc.PublicDomain,
		InternalDomain: svc.PrivateDomain,
		Env:            env,
		CreatedAt:      svc.CreatedAt,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, msgServerError, err)
		return
	}
}

func (app *Application) GetGithubServices(w http.ResponseWriter, r *http.Request) {
	projectId := mux.Vars(r)["project_id"]
	if projectId == "" {
		writeError(w, http.StatusBadRequest, "A project ID is required.", nil)
		return
	}

	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, msgUnauthorized, nil)
		return
	}

	services, err := app.Supabase.GetGithubServices(projectId, claims.Subject)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Couldn't load the project's services.", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(ToGithubServicesResponse(services)); err != nil {
		writeError(w, http.StatusInternalServerError, msgServerError, err)
		return
	}
}

func (app *Application) CreateGithubService(w http.ResponseWriter, r *http.Request) {
	projectId := mux.Vars(r)["project_id"]
	if projectId == "" {
		writeError(w, http.StatusBadRequest, "A project ID is required.", nil)
		return
	}

	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, msgUnauthorized, nil)
		return
	}
	userId := claims.Subject

	var req GithubServiceCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "That service request wasn't valid.", err)
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "Service name is required.", nil)
		return
	}
	if req.Repo == "" {
		writeError(w, http.StatusBadRequest, "Repository is required.", nil)
		return
	}
	if req.RepoId == "" {
		writeError(w, http.StatusBadRequest, "Repository ID is required.", nil)
		return
	}
	if req.Port <= 0 || req.Port > 65535 {
		writeError(w, http.StatusBadRequest, "Port must be between 1 and 65535.", nil)
		return
	}
	if ok, err := util.ValidateEnv(req.Env); err != nil || !ok {
		writeError(w, http.StatusBadRequest, "Environment variables must be one KEY=value pair per line.", err)
		return
	}
	if req.Domain != nil {
		normalized := util.NormalizeDomain(*req.Domain)
		if normalized == "" {
			req.Domain = nil
		} else {
			if ok, _ := util.ValidateDomainLabel(normalized); !ok {
				writeError(w, http.StatusBadRequest, "Custom domain must be 1-63 characters, lowercase letters, numbers, and hyphens only, and cannot start or end with a hyphen.", nil)
				return
			}
			req.Domain = &normalized
		}
	}

	sanitizedRootDir, err := util.SanitizeRootDir(req.RootDir)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), err)
		return
	}

	connection, err := app.Supabase.GetGithubConnection(userId)
	if err != nil {
		writeError(w, http.StatusNotFound, "No GitHub connection found. Please install the GitHub App first.", err)
		return
	}

	domainRequested := req.Domain != nil

	res, err := app.Supabase.CreateGithubService(userId, projectId, req.Name, connection.Id, req.Repo, req.RepoId, sanitizedRootDir, req.Domain, req.Port)
	if err != nil {
		if domainRequested && isDomainTakenError(err) {
			writeError(w, http.StatusConflict, "That domain is already taken. Please choose a different one.", err)
			return
		}
		if domainRequested && isDomainValidationError(err) {
			writeError(w, http.StatusBadRequest, domainValidationMessage(err), err)
			return
		}
		writeError(w, http.StatusInternalServerError, "Couldn't create the service.", err)
		return
	}

	// Build image using railpack with RootDir as build context.
	// Clone URL derived from repo (owner/repo). For private repos, inject installation token.
	cloneURL := fmt.Sprintf("https://github.com/%s.git", req.Repo)
	accessToken, _ := app.getInstallationToken(r.Context(), connection.InstallationId)
	imageTag := fmt.Sprintf("local/%s:latest", res.ResourceName)

	builtImage := ""
	if image, err := cloneAndBuild(r.Context(), cloneURL, accessToken, sanitizedRootDir, imageTag); err != nil {
		slog.Error("failed to build github service image", "service_id", res.Id, "root_dir", sanitizedRootDir, "err", err)
		// Mark failed but keep DB record; user can retry via webhook or update.
		if _, statusErr := app.Supabase.UpdateGithubServiceStatus(res.Id, userId, "failed"); statusErr != nil {
			slog.Error("failed to mark github service failed after build error", "service_id", res.Id, "err", statusErr)
		}
		writeError(w, http.StatusInternalServerError, "Your service was created, but we couldn't build its image. Check the root directory and try again.", err)
		return
	} else {
		builtImage = image
	}

	if builtImage == "" {
		builtImage = imageTag
	}

	if err := app.Deploy.CreateService(r.Context(), deploy.Service{
		Namespace: "proj-" + res.ProjectId,
		Name:      res.ResourceName,
		Hostname:  res.PublicDomain,
		Env:       util.ParseEnv(req.Env),
		Port:      req.Port,
		Image:     builtImage,
	}); err != nil {
		if _, statusErr := app.Supabase.UpdateGithubServiceStatus(res.Id, userId, "failed"); statusErr != nil {
			slog.Error("failed to mark github service failed after deploy error", "service_id", res.Id, "err", statusErr)
		}
		writeError(w, http.StatusInternalServerError, "Your service was created, but we couldn't start it. A refresh will show its current status.", err)
		return
	}

	if _, err := app.Supabase.UpdateGithubServiceStatus(res.Id, userId, "running"); err != nil {
		writeError(w, http.StatusInternalServerError, "Your service was started, but we couldn't confirm its status. A refresh will show where things stand.", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(GithubServiceResponse{
		Id:             res.Id,
		ProjectId:      res.ProjectId,
		Name:           res.Name,
		Repo:           res.Repo,
		RepoId:         res.RepoId,
		RootDir:        res.RootDir,
		Port:           res.Port,
		Status:         "running",
		PublicDomain:   res.PublicDomain,
		InternalDomain: res.PrivateDomain,
		CreatedAt:      res.CreatedAt,
	})
}

func (app *Application) UpdateGithubService(w http.ResponseWriter, r *http.Request) {
	projectId := mux.Vars(r)["project_id"]
	if projectId == "" {
		writeError(w, http.StatusBadRequest, "A project ID is required.", nil)
		return
	}
	githubServiceId := mux.Vars(r)["github_service_id"]
	if githubServiceId == "" {
		writeError(w, http.StatusBadRequest, "A service ID is required.", nil)
		return
	}

	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, msgUnauthorized, nil)
		return
	}

	var req GithubServiceUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "That service request wasn't valid.", err)
		return
	}

	if req.Name == nil || *req.Name == "" {
		writeError(w, http.StatusBadRequest, "Service name is required.", nil)
		return
	}
	if req.Port == nil || *req.Port <= 0 || *req.Port > 65535 {
		writeError(w, http.StatusBadRequest, "Port must be between 1 and 65535.", nil)
		return
	}
	if req.Env != nil {
		if ok, err := util.ValidateEnv(*req.Env); err != nil || !ok {
			writeError(w, http.StatusBadRequest, "Environment variables must be one KEY=value pair per line.", err)
			return
		}
	}
	if req.Domain != nil {
		normalized := util.NormalizeDomain(*req.Domain)
		if normalized == "" {
			req.Domain = nil
		} else {
			if ok, _ := util.ValidateDomainLabel(normalized); !ok {
				writeError(w, http.StatusBadRequest, "Custom domain must be 1-63 characters, lowercase letters, numbers, and hyphens only, and cannot start or end with a hyphen.", nil)
				return
			}
			req.Domain = &normalized
		}
	}

	userId := claims.Subject
	domainRequested := req.Domain != nil

	res, err := app.Supabase.UpdateGithubService(githubServiceId, userId, *req.Name, req.Domain, *req.Port)
	if err != nil {
		if domainRequested && isDomainTakenError(err) {
			writeError(w, http.StatusConflict, "That domain is already taken. Please choose a different one.", err)
			return
		}
		if domainRequested && isDomainValidationError(err) {
			writeError(w, http.StatusBadRequest, domainValidationMessage(err), err)
			return
		}
		writeError(w, http.StatusInternalServerError, "Couldn't save the service.", err)
		return
	}

	envStr := ""
	if req.Env != nil {
		envStr = *req.Env
	}
	// Re-deploy with existing image (or rebuilt image if desired). For now reuse current deploy.
	if err := app.Deploy.CreateService(r.Context(), deploy.Service{
		Namespace: "proj-" + projectId,
		Name:      res.ResourceName,
		Hostname:  res.PublicDomain,
		Env:       util.ParseEnv(envStr),
		Image:     fmt.Sprintf("local/%s:latest", res.ResourceName),
		Port:      *req.Port,
	}); err != nil {
		if _, statusErr := app.Supabase.UpdateGithubServiceStatus(res.Id, userId, "failed"); statusErr != nil {
			slog.Error("failed to mark github service failed after deploy error", "service_id", res.Id, "err", statusErr)
		}
		writeError(w, http.StatusInternalServerError, "We saved your changes, but couldn't restart your service. A refresh will show its current status.", err)
		return
	}

	if _, err := app.Supabase.UpdateGithubServiceStatus(res.Id, userId, "running"); err != nil {
		writeError(w, http.StatusInternalServerError, "Your service was restarted, but we couldn't confirm its status. A refresh will show where things stand.", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(GithubServiceResponse{
		Id:             res.Id,
		ProjectId:      res.ProjectId,
		Name:           res.Name,
		Repo:           res.Repo,
		RepoId:         res.RepoId,
		RootDir:        res.RootDir,
		Port:           res.Port,
		Status:         res.Status,
		PublicDomain:   res.PublicDomain,
		InternalDomain: res.PrivateDomain,
		CreatedAt:      res.CreatedAt,
	})
}

func (app *Application) DeleteGithubService(w http.ResponseWriter, r *http.Request) {
	projectId := mux.Vars(r)["project_id"]
	if projectId == "" {
		writeError(w, http.StatusBadRequest, "A project ID is required.", nil)
		return
	}
	githubServiceId := mux.Vars(r)["github_service_id"]
	if githubServiceId == "" {
		writeError(w, http.StatusBadRequest, "A service ID is required.", nil)
		return
	}

	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, msgUnauthorized, nil)
		return
	}

	if err := app.Supabase.DeleteGithubService(githubServiceId, claims.Subject); err != nil {
		writeError(w, http.StatusInternalServerError, "Couldn't delete the service.", err)
		return
	}

	if err := app.Deploy.DeleteService(r.Context(), deploy.Service{
		Namespace: "proj-" + projectId,
		Name:      "svc-" + githubServiceId,
	}); err != nil {
		slog.Error("failed to clean up github service infrastructure", "service_id", githubServiceId, "err", err)
	}

	w.WriteHeader(204)
}

func (app *Application) DeleteGithubServices(w http.ResponseWriter, r *http.Request) {
	projectId := mux.Vars(r)["project_id"]
	if projectId == "" {
		writeError(w, http.StatusBadRequest, "A project ID is required.", nil)
		return
	}

	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, msgUnauthorized, nil)
		return
	}

	var req BulkDeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "That deletion request wasn't valid.", err)
		return
	}

	if len(req.Ids) == 0 {
		writeError(w, http.StatusBadRequest, "At least one service ID is required.", nil)
		return
	}

	deleted := []string{}
	failed := []FailedDelete{}
	for _, id := range req.Ids {
		if err := app.Supabase.DeleteGithubService(id, claims.Subject); err != nil {
			failed = append(failed, FailedDelete{Id: id, Error: "Couldn't delete the service."})
			continue
		}
		if err := app.Deploy.DeleteService(r.Context(), deploy.Service{
			Namespace: "proj-" + projectId,
			Name:      "svc-" + id,
		}); err != nil {
			slog.Error("failed to clean up github service infrastructure", "service_id", id, "err", err)
		}
		deleted = append(deleted, id)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(BulkDeleteResponse{Deleted: deleted, Failed: failed}); err != nil {
		writeError(w, http.StatusInternalServerError, msgServerError, err)
		return
	}
}

func ToGithubServicesResponse(tables []store.GithubServicesTable) []GithubServiceResponse {
	out := []GithubServiceResponse{}
	for _, t := range tables {
		out = append(out, GithubServiceResponse{
			Id:             t.Id,
			ProjectId:      t.ProjectId,
			Name:           t.Name,
			Repo:           t.Repo,
			RepoId:         t.RepoId,
			RootDir:        t.RootDir,
			Port:           t.Port,
			Status:         t.Status,
			PublicDomain:   t.PublicDomain,
			InternalDomain: t.PrivateDomain,
			CreatedAt:      t.CreatedAt,
		})
	}
	return out
}
