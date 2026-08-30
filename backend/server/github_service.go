package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

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

	conn, err := app.Supabase.GetGithubConnectionByInstallationId(installationId)
	if err != nil {
		writeError(w, http.StatusNotFound, "We couldn't find that GitHub installation.", err)
		return
	}
	if conn.UserId != claims.Subject {
		writeError(w, http.StatusForbidden, "You don't have access to that installation.", nil)
		return
	}

	token, err := app.Github.GetInstallationToken(r.Context(), installationId)
	if err != nil {
		slog.Error("GetInstallationToken failed", "installation_id", installationId, "err", err)
		writeError(w, http.StatusInternalServerError, "GitHub installation token not configured.", err)
		return
	}
	if token == "" {
		slog.Error("GetInstallationToken empty", "installation_id", installationId)
		writeError(w, http.StatusInternalServerError, "GitHub App not configured.", nil)
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), "GET", "https://api.github.com/installation/repositories", nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, msgServerError, err)
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("GitHub repos request failed", "installation_id", installationId, "err", err)
		writeError(w, http.StatusBadGateway, "Couldn't reach GitHub.", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		body, _ := io.ReadAll(resp.Body)
		slog.Error("GitHub Bad credentials", "installation_id", installationId, "app_id", app.Config.Github.AppID, "status", resp.StatusCode, "body", string(body))
		writeError(w, http.StatusBadGateway, "GitHub Bad credentials - check GITHUB_APP_ID/PRIVATE_KEY and that installation 157404346 belongs to app tysoncloud", nil)
		return
	}

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
	if req.RepoId == 0 {
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
		if isDuplicateKeyError(err) {
			writeError(w, http.StatusConflict, "That name is already taken. Please choose a different one.", err)
			return
		}
		writeError(w, http.StatusInternalServerError, "Couldn't create the service.", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
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

	go func() {
		buildCtx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()

		cloneURL := fmt.Sprintf("https://github.com/%s.git", req.Repo)
		accessToken, err := app.Github.GetInstallationToken(buildCtx, strconv.FormatInt(connection.InstallationId, 10))
		if err != nil {
			slog.Error("failed to get installation token", "service_id", res.Id, "err", err)
			if _, statusErr := app.Supabase.UpdateGithubServiceStatus(res.Id, userId, "failed"); statusErr != nil {
				slog.Error("failed to mark github service failed after token error", "service_id", res.Id, "err", statusErr)
			}
			return
		}

		registryURL := app.Github.RegistryURL()
		imageTag := app.Github.RegistryTag(registryURL, res.ResourceName, "latest")

		image, err := app.Github.CloneAndBuild(buildCtx, cloneURL, accessToken, sanitizedRootDir, imageTag)
		if err != nil {
			slog.Error("failed to build github service image", "service_id", res.Id, "root_dir", sanitizedRootDir, "err", err)
			if _, statusErr := app.Supabase.UpdateGithubServiceStatus(res.Id, userId, "failed"); statusErr != nil {
				slog.Error("failed to mark github service failed after build error", "service_id", res.Id, "err", statusErr)
			}
			return
		}

		if err := app.Deploy.CreateService(buildCtx, deploy.Service{
			Namespace: "proj-" + res.ProjectId,
			Name:      res.ResourceName,
			Hostname:  res.PublicDomain,
			Env:       util.ParseEnv(req.Env),
			Port:      req.Port,
			Image:     image,
		}); err != nil {
			slog.Error("failed to deploy github service", "service_id", res.Id, "err", err)
			if _, statusErr := app.Supabase.UpdateGithubServiceStatus(res.Id, userId, "failed"); statusErr != nil {
				slog.Error("failed to mark github service failed after deploy error", "service_id", res.Id, "err", statusErr)
			}
			return
		}

		if _, err := app.Supabase.UpdateGithubServiceStatus(res.Id, userId, "running"); err != nil {
			slog.Error("failed to mark github service running after successful deploy", "service_id", res.Id, "err", err)
		}
	}()
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
		if isDuplicateKeyError(err) {
			writeError(w, http.StatusConflict, "That name is already taken. Please choose a different one.", err)
			return
		}
		writeError(w, http.StatusInternalServerError, "Couldn't save the service.", err)
		return
	}

	envStr := ""
	if req.Env != nil {
		envStr = *req.Env
	}
	// For updates we keep the current deployed image (registry tag). The image is managed by
	// CreateGithubService (latest) and webhook pushes (per-commit SHA). Update only restarts
	// with new env/domain/port, not a rebuild.
	registryURLUpdate := app.Github.RegistryURL()
	fallbackImage := app.Github.RegistryTag(registryURLUpdate, res.ResourceName, "latest")
	if err := app.Deploy.CreateService(r.Context(), deploy.Service{
		Namespace: "proj-" + projectId,
		Name:      res.ResourceName,
		Hostname:  res.PublicDomain,
		Env:       util.ParseEnv(envStr),
		Image:     fallbackImage,
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

	// Resolve actual k8s resource name if possible; fallback to svc-<id> convention on lookup failure.
	resourceName := "svc-" + githubServiceId
	if svc, err := app.Supabase.GetGithubServiceById(githubServiceId); err == nil && svc.ResourceName != "" {
		resourceName = svc.ResourceName
	}
	if err := app.Deploy.DeleteService(r.Context(), deploy.Service{
		Namespace: "proj-" + projectId,
		Name:      resourceName,
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
		resourceName := "svc-" + id
		if svc, err := app.Supabase.GetGithubServiceById(id); err == nil && svc.ResourceName != "" {
			resourceName = svc.ResourceName
		}
		if err := app.Deploy.DeleteService(r.Context(), deploy.Service{
			Namespace: "proj-" + projectId,
			Name:      resourceName,
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
