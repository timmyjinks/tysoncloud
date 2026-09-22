package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/clerk/clerk-sdk-go/v2"
	clerkjwt "github.com/clerk/clerk-sdk-go/v2/jwt"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
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

	branch := strings.TrimSpace(svc.Branch)
	if branch == "" {
		branch = "main"
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(GithubServiceResponse{
		Id:             svc.Id,
		ProjectId:      svc.ProjectId,
		Name:           svc.Name,
		Repo:           svc.RepoName,
		RepoId:         svc.RepoId,
		RootDir:        svc.RootDir,
		Branch:         branch,
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

	branch := strings.TrimSpace(req.Branch)
	if branch == "" {
		branch = "main"
	}
	sanitizedBranch, err := util.SanitizeBranch(branch)
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

	res, err := app.Supabase.CreateGithubService(userId, projectId, req.Name, connection.Id, req.Repo, req.RepoId, sanitizedRootDir, sanitizedBranch, req.Domain, req.Port)
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
		Repo:           res.RepoName,
		RepoId:         res.RepoId,
		RootDir:        res.RootDir,
		Branch:         sanitizedBranch,
		Port:           res.Port,
		Status:         res.Status,
		PublicDomain:   res.PublicDomain,
		InternalDomain: res.PrivateDomain,
		CreatedAt:      res.CreatedAt,
	})

	go func() {
		buildCtx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		defer cancel()

		logFn := func(line string) {
			app.Github.AppendBuildLog(res.Id, line)
		}
		emitState := func(to string) {
			line := `[state] ` + to + ` service=` + res.Id + ` repo=` + req.Repo + ` branch=` + sanitizedBranch + ` root_dir=` + sanitizedRootDir
			app.Github.AppendBuildLog(res.Id, line)
			slog.Info("github deploy state", "service_id", res.Id, "to", to, "repo", req.Repo, "branch", sanitizedBranch, "root_dir", sanitizedRootDir)
		}

		emitState("building")
		if _, statusErr := app.Supabase.UpdateGithubServiceStatus(res.Id, userId, "building"); statusErr != nil {
			slog.Warn("failed to mark github service building", "service_id", res.Id, "err", statusErr)
		}

		cloneURL := fmt.Sprintf("https://github.com/%s.git", req.Repo)
		accessToken, err := app.Github.GetInstallationToken(buildCtx, strconv.FormatInt(connection.InstallationId, 10))
		if err != nil {
			slog.Error("failed to get installation token", "service_id", res.Id, "err", err)
			logFn(`[build] failed to get installation token: ` + err.Error())
			emitState("failed")
			if _, statusErr := app.Supabase.UpdateGithubServiceStatus(res.Id, userId, "failed"); statusErr != nil {
				slog.Error("failed to mark github service failed after token error", "service_id", res.Id, "err", statusErr)
			}
			return
		}

		registryURL := app.Github.RegistryURL()
		imageTag := app.Github.RegistryTag(registryURL, res.ResourceName, "latest")

		image, err := app.Github.CloneAndBuildWithLogs(buildCtx, cloneURL, accessToken, sanitizedRootDir, sanitizedBranch, imageTag, logFn, util.ParseEnv(req.Env))
		if err != nil {
			slog.Error("failed to build github service image", "service_id", res.Id, "root_dir", sanitizedRootDir, "branch", sanitizedBranch, "err", err)
			logFn(`[state] building failed: ` + err.Error())
			emitState("failed")
			if _, statusErr := app.Supabase.UpdateGithubServiceStatus(res.Id, userId, "failed"); statusErr != nil {
				slog.Error("failed to mark github service failed after build error", "service_id", res.Id, "err", statusErr)
			}
			return
		}

		emitState("deploying")
		if _, statusErr := app.Supabase.UpdateGithubServiceStatus(res.Id, userId, "deploying"); statusErr != nil {
			slog.Warn("failed to mark github service deploying", "service_id", res.Id, "err", statusErr)
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
			logFn(`[state] deploying failed: ` + err.Error())
			emitState("failed")
			if _, statusErr := app.Supabase.UpdateGithubServiceStatus(res.Id, userId, "failed"); statusErr != nil {
				slog.Error("failed to mark github service failed after deploy error", "service_id", res.Id, "err", statusErr)
			}
			return
		}

		emitState("running")
		if _, err := app.Supabase.UpdateGithubServiceStatus(res.Id, userId, "running"); err != nil {
			slog.Error("failed to mark github service running after successful deploy", "service_id", res.Id, "err", err)
		}
		logFn(`[state] running image=` + image)
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
	var sanitizedBranch *string
	if req.Branch != nil {
		trimmed := strings.TrimSpace(*req.Branch)
		if trimmed == "" {
			writeError(w, http.StatusBadRequest, "Branch is required.", nil)
			return
		}
		sb, err := util.SanitizeBranch(trimmed)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error(), err)
			return
		}
		sanitizedBranch = &sb
	}

	userId := claims.Subject
	domainRequested := req.Domain != nil

	existing, err := app.Supabase.GetGithubService(githubServiceId, userId)
	if err != nil {
		writeError(w, http.StatusNotFound, "We couldn't find that service.", err)
		return
	}
	oldBranch := existing.Branch
	if strings.TrimSpace(oldBranch) == "" {
		oldBranch = "main"
	}

	res, err := app.Supabase.UpdateGithubService(githubServiceId, userId, *req.Name, req.Domain, *req.Port, sanitizedBranch)
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

	newBranch := oldBranch
	if sanitizedBranch != nil {
		newBranch = *sanitizedBranch
	}
	if res.Branch != "" {
		newBranch = res.Branch
	}

	if sanitizedBranch != nil && *sanitizedBranch != oldBranch {
		// Branch changed: rebuild from git asynchronously, like creation.
		connection, connErr := app.Supabase.GetGithubConnection(userId)
		if connErr != nil {
			writeError(w, http.StatusInternalServerError, "We saved your changes, but couldn't rebuild from the new branch. Use Redeploy to retry.", connErr)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(GithubServiceResponse{
			Id:             res.Id,
			ProjectId:      res.ProjectId,
			Name:           res.Name,
			Repo:           res.RepoName,
			RepoId:         res.RepoId,
			RootDir:        res.RootDir,
			Branch:         newBranch,
			Port:           res.Port,
			Status:         res.Status,
			PublicDomain:   res.PublicDomain,
			InternalDomain: res.PrivateDomain,
			CreatedAt:      res.CreatedAt,
		})

		go app.buildGithubServiceFromGit(userId, res, connection.InstallationId, newBranch, envStr)
		return
	}

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
		Repo:           res.RepoName,
		RepoId:         res.RepoId,
		RootDir:        res.RootDir,
		Branch:         newBranch,
		Port:           res.Port,
		Status:         res.Status,
		PublicDomain:   res.PublicDomain,
		InternalDomain: res.PrivateDomain,
		CreatedAt:      res.CreatedAt,
	})
}

func (app *Application) buildGithubServiceFromGit(userId string, svc store.GithubServicesTable, installationId int64, branch, envStr string) {
	buildCtx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	sanitizedRootDir := svc.RootDir
	if v, err := util.SanitizeRootDir(svc.RootDir); err == nil {
		sanitizedRootDir = v
	}
	if strings.TrimSpace(branch) == "" {
		branch = "main"
	}

	logFn := func(line string) {
		app.Github.AppendBuildLog(svc.Id, line)
	}
	emitState := func(to string) {
		line := `[state] ` + to + ` service=` + svc.Id + ` repo=` + svc.RepoName + ` branch=` + branch + ` root_dir=` + sanitizedRootDir
		app.Github.AppendBuildLog(svc.Id, line)
		slog.Info("github deploy state", "service_id", svc.Id, "to", to, "repo", svc.RepoName, "branch", branch, "root_dir", sanitizedRootDir)
	}

	emitState("building")
	if _, statusErr := app.Supabase.UpdateGithubServiceStatus(svc.Id, userId, "building"); statusErr != nil {
		slog.Warn("failed to mark github service building", "service_id", svc.Id, "err", statusErr)
	}

	cloneURL := fmt.Sprintf("https://github.com/%s.git", svc.RepoName)
	accessToken, err := app.Github.GetInstallationToken(buildCtx, strconv.FormatInt(installationId, 10))
	if err != nil {
		slog.Error("failed to get installation token", "service_id", svc.Id, "err", err)
		logFn(`[build] failed to get installation token: ` + err.Error())
		emitState("failed")
		if _, statusErr := app.Supabase.UpdateGithubServiceStatus(svc.Id, userId, "failed"); statusErr != nil {
			slog.Error("failed to mark github service failed after token error", "service_id", svc.Id, "err", statusErr)
		}
		return
	}

	registryURL := app.Github.RegistryURL()
	imageTag := app.Github.RegistryTag(registryURL, svc.ResourceName, "latest")

	// Resolve the effective env BEFORE the build so the image bakes the same
	// vars it runs with. Ephemeral — never stored beyond the K8s secret.
	buildEnv := util.ParseEnv(envStr)
	if strings.TrimSpace(envStr) == "" {
		if existingEnv, envErr := app.Deploy.GetServiceEnv(buildCtx, deploy.Service{
			Namespace: "proj-" + svc.ProjectId,
			Name:      svc.ResourceName,
		}); envErr == nil && len(existingEnv) > 0 {
			buildEnv = map[string][]byte{}
			for k, v := range existingEnv {
				buildEnv[k] = []byte(v)
			}
		}
	}

	image, err := app.Github.CloneAndBuildWithLogs(buildCtx, cloneURL, accessToken, sanitizedRootDir, branch, imageTag, logFn, buildEnv)
	if err != nil {
		slog.Error("failed to build github service image", "service_id", svc.Id, "branch", branch, "root_dir", sanitizedRootDir, "err", err)
		logFn(`[state] building failed: ` + err.Error())
		emitState("failed")
		if _, statusErr := app.Supabase.UpdateGithubServiceStatus(svc.Id, userId, "failed"); statusErr != nil {
			slog.Error("failed to mark github service failed after build error", "service_id", svc.Id, "err", statusErr)
		}
		return
	}

	emitState("deploying")
	if _, statusErr := app.Supabase.UpdateGithubServiceStatus(svc.Id, userId, "deploying"); statusErr != nil {
		slog.Warn("failed to mark github service deploying", "service_id", svc.Id, "err", statusErr)
	}

	env := buildEnv

	if err := app.Deploy.CreateService(buildCtx, deploy.Service{
		Namespace: "proj-" + svc.ProjectId,
		Name:      svc.ResourceName,
		Hostname:  svc.PublicDomain,
		Env:       env,
		Port:      svc.Port,
		Image:     image,
	}); err != nil {
		slog.Error("failed to deploy github service", "service_id", svc.Id, "err", err)
		logFn(`[state] deploying failed: ` + err.Error())
		emitState("failed")
		if _, statusErr := app.Supabase.UpdateGithubServiceStatus(svc.Id, userId, "failed"); statusErr != nil {
			slog.Error("failed to mark github service failed after deploy error", "service_id", svc.Id, "err", statusErr)
		}
		return
	}

	emitState("running")
	if _, err := app.Supabase.UpdateGithubServiceStatus(svc.Id, userId, "running"); err != nil {
		slog.Error("failed to mark github service running after successful deploy", "service_id", svc.Id, "err", err)
	}
	logFn(`[state] running image=` + image)
}

func (app *Application) RedeployGithubService(w http.ResponseWriter, r *http.Request) {
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
	userId := claims.Subject

	svc, err := app.Supabase.GetGithubService(githubServiceId, userId)
	if err != nil {
		writeError(w, http.StatusNotFound, "We couldn't find that service.", err)
		return
	}

	status := strings.ToLower(strings.TrimSpace(svc.Status))
	if status == "building" || status == "deploying" {
		writeError(w, http.StatusConflict, "That service is already deploying. Please wait for it to finish.", nil)
		return
	}

	connection, err := app.Supabase.GetGithubConnection(userId)
	if err != nil {
		writeError(w, http.StatusNotFound, "No GitHub connection found. Please install the GitHub App first.", err)
		return
	}

	branch := strings.TrimSpace(svc.Branch)
	if branch == "" {
		branch = "main"
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(GithubServiceResponse{
		Id:             svc.Id,
		ProjectId:      svc.ProjectId,
		Name:           svc.Name,
		Repo:           svc.RepoName,
		RepoId:         svc.RepoId,
		RootDir:        svc.RootDir,
		Branch:         branch,
		Port:           svc.Port,
		Status:         "building",
		PublicDomain:   svc.PublicDomain,
		InternalDomain: svc.PrivateDomain,
		CreatedAt:      svc.CreatedAt,
	})

	go app.buildGithubServiceFromGit(userId, svc, connection.InstallationId, branch, "")
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

	// Snapshot previews BEFORE the DB delete: the namespace is shared per
	// repo+PR, so remove only this service's K8s objects from it and delete
	// the namespace only when no sibling previews remain. No env values are
	// stored in preview_environments, so nothing secret to purge.
	previewsBefore, _ := app.Supabase.GetPreviewEnvironmentsByService(githubServiceId)
	svcBefore, _ := app.Supabase.GetGithubService(githubServiceId, claims.Subject)

	if err := app.Supabase.DeleteGithubService(githubServiceId, claims.Subject); err != nil {
		writeError(w, http.StatusInternalServerError, "Couldn't delete the service.", err)
		return
	}

	cleanupSharedPreviewObjects(r.Context(), app, previewsBefore)
	_ = app.Supabase.DeletePreviewEnvironmentsByService(githubServiceId)

	resourceName := "svc-" + githubServiceId
	if svcBefore.ResourceName != "" {
		resourceName = svcBefore.ResourceName
	}
	if err := app.Deploy.DeleteService(r.Context(), deploy.Service{
		Namespace: "proj-" + projectId,
		Name:      resourceName,
	}); err != nil {
		slog.Error("failed to clean up github service infrastructure", "service_id", githubServiceId, "err", err)
	}

	w.WriteHeader(204)
}

// cleanupSharedPreviewObjects removes one service's preview K8s objects from
// the shared per-repo+PR namespace and deletes the namespace itself only
// when no remaining previews reference it. Best-effort — a missing object
// just means that preview never deployed. Deploy.DeleteService is idempotent
// (ignores NotFound for Secret/PVC/HPA/Deployment/Service/HTTPRoute).
func cleanupSharedPreviewObjects(ctx context.Context, app *Application, previews []store.PreviewEnvironmentsTable) {
	for _, p := range previews {
		if strings.TrimSpace(p.Namespace) == "" || strings.TrimSpace(p.ResourceName) == "" {
			continue
		}
		if err := app.Deploy.DeleteService(ctx, deploy.Service{
			Namespace: p.Namespace,
			Name:      p.ResourceName,
		}); err != nil {
			slog.Warn("failed to clean up preview objects", "service_id", p.GithubServiceId, "namespace", p.Namespace, "name", p.ResourceName, "err", err)
		}
	}
	seen := map[string]store.PreviewEnvironmentsTable{}
	for _, p := range previews {
		if strings.TrimSpace(p.Namespace) == "" {
			continue
		}
		seen[p.Namespace] = p
	}
	for namespace, sample := range seen {
		remaining, err := app.Supabase.GetPreviewEnvironmentsByRepoPR(sample.RepoId, sample.PrNumber)
		if err != nil {
			slog.Warn("failed to check remaining previews, keeping namespace", "namespace", namespace, "err", err)
			continue
		}
		stillReferenced := false
		for _, r := range remaining {
			if strings.TrimSpace(r.Namespace) == namespace {
				stillReferenced = true
				break
			}
		}
		if stillReferenced {
			continue
		}
		if err := app.Deploy.DeleteProject(ctx, namespace); err != nil {
			slog.Warn("failed to clean up empty preview namespace", "namespace", namespace, "err", err)
		} else {
			slog.Info("preview cleanup: deleted empty namespace", "namespace", namespace)
		}
	}
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
		previewsBefore, _ := app.Supabase.GetPreviewEnvironmentsByService(id)
		svcBefore, _ := app.Supabase.GetGithubService(id, claims.Subject)
		if err := app.Supabase.DeleteGithubService(id, claims.Subject); err != nil {
			failed = append(failed, FailedDelete{Id: id, Error: "Couldn't delete the service."})
			continue
		}
		// Cascade preview objects for this service only (best-effort);
		// the shared namespace goes away only when its last preview does.
		cleanupSharedPreviewObjects(r.Context(), app, previewsBefore)
		_ = app.Supabase.DeletePreviewEnvironmentsByService(id)
		resourceName := "svc-" + id
		if svcBefore.ResourceName != "" {
			resourceName = svcBefore.ResourceName
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

func (app *Application) GetGithubServiceLogs(w http.ResponseWriter, r *http.Request) {
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
	token := r.URL.Query().Get("token")
	if token == "" {
		cookie, cookieErr := r.Cookie("__session")
		if cookieErr != nil {
			writeError(w, http.StatusUnauthorized, msgUnauthorized, cookieErr)
			return
		}
		token = cookie.Value
	}
	claims, err := clerkjwt.Verify(r.Context(), &clerkjwt.VerifyParams{Token: token})
	if err != nil {
		writeError(w, http.StatusUnauthorized, msgUnauthorized, err)
		return
	}
	svc, err := app.Supabase.GetGithubService(githubServiceId, claims.Subject)
	if err != nil {
		writeError(w, http.StatusNotFound, "We couldn't find that service.", err)
		return
	}
	allowedOrigins := parseAllowedOrigins(app.Config.Server.AllowedOrigins)
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true
			}
			return allowedOrigins[origin]
		},
	}
	ws, err := upgrader.Upgrade(w, r, http.Header{})
	if err != nil {
		slog.Error("github log stream upgrade failed", "err", err)
		return
	}
	defer ws.Close()
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	status := strings.ToLower(strings.TrimSpace(svc.Status))
	isPreDeploy := status == "pending" || status == "building" || status == "deploying" || status == ""
	if isPreDeploy {
		lines := make(chan string, 64)
		subID, ch, snap := app.Github.SubscribeBuildLogs(svc.Id)
		defer app.Github.UnsubscribeBuildLogs(svc.Id, subID)
		stateLine := `[state] ` + svc.Status + ` repo=` + svc.RepoName + ` branch=` + svc.Branch + ` root_dir=` + svc.RootDir + ` domain=` + svc.PublicDomain + ` port=` + strconv.FormatInt(int64(svc.Port), 10)
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
			Namespace: "proj-" + projectId,
			Name:      svc.ResourceName,
		}
		isDiagnostic := status == "pending" || status == "failed"
		var logErr error
		if isDiagnostic {
			logErr = app.Deploy.GetServiceDiagnosticLogs(ctx, svcRes, lines)
		} else {
			logErr = app.Deploy.GetServiceLogs(ctx, svcRes, lines)
		}
		if logErr != nil && ctx.Err() == nil {
			slog.Error("github log stream failed", "service_id", githubServiceId, "status", svc.Status, "err", logErr)
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

func ToGithubServicesResponse(tables []store.GithubServicesTable) []GithubServiceResponse {
	out := []GithubServiceResponse{}
	for _, t := range tables {
		branch := t.Branch
		if strings.TrimSpace(branch) == "" {
			branch = "main"
		}
		out = append(out, GithubServiceResponse{
			Id:             t.Id,
			ProjectId:      t.ProjectId,
			Name:           t.Name,
			Repo:           t.RepoName,
			RepoId:         t.RepoId,
			RootDir:        t.RootDir,
			Branch:         branch,
			Port:           t.Port,
			Status:         t.Status,
			PublicDomain:   t.PublicDomain,
			InternalDomain: t.PrivateDomain,
			CreatedAt:      t.CreatedAt,
		})
	}
	return out
}
