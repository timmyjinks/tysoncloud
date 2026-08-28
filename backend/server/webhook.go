package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/timmyjinks/tysoncloud/deploy"
	"github.com/timmyjinks/tysoncloud/util"
)

func verifyWebhookSignature(secret, signature string, body []byte) bool {
	if secret == "" {
		return true // skip if not configured (local dev)
	}
	if signature == "" {
		return false
	}
	// signature is "sha256=<hex>"
	hexSig := strings.TrimPrefix(signature, "sha256=")
	expectedMAC := hmac.New(sha256.New, []byte(secret))
	expectedMAC.Write(body)
	expected := hex.EncodeToString(expectedMAC.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(hexSig))
}

func (app *Application) GithubWebhook(w http.ResponseWriter, r *http.Request) {
	eventType := r.Header.Get("X-GitHub-Event")
	if eventType == "" {
		eventType = r.Header.Get("X-Github-Event")
	}

	// Read body for signature verification and payload parsing
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 2<<20))
	if err != nil {
		http.Error(w, "could not read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// HMAC verify if secret set (env GITHUB_WEBHOOK_SECRET)
	secret := app.Config.Server.GithubWebhookSecret
	if !verifyWebhookSignature(secret, r.Header.Get("X-Hub-Signature-256"), body) {
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	switch eventType {
	case "ping":
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"msg":"pong"}`))
		return

	case "push":
		var payload struct {
			Ref          string `json:"ref"`
			Deleted      bool   `json:"deleted"`
			HeadCommitID string `json:"after"`
			Repository   struct {
				Id       int64  `json:"id"`
				FullName string `json:"full_name"`
				CloneURL string `json:"clone_url"`
			} `json:"repository"`
			Installation struct {
				Id int64 `json:"id"`
			} `json:"installation"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if payload.Deleted {
			w.WriteHeader(http.StatusOK)
			return
		}
		if payload.Repository.Id == 0 {
			http.Error(w, "missing repository.id", http.StatusBadRequest)
			return
		}

		repoId := strconv.FormatInt(payload.Repository.Id, 10)
		installationId := ""
		if payload.Installation.Id != 0 {
			installationId = strconv.FormatInt(payload.Installation.Id, 10)
		}

		services, err := app.Supabase.GetGithubServicesByRepoId(repoId)
		if err != nil {
			slog.Error("failed to lookup github services by repo_id", "repo_id", repoId, "err", err)
			http.Error(w, "lookup failed", http.StatusInternalServerError)
			return
		}
		if len(services) == 0 {
			slog.Info("webhook push: no services for repo", "repo_id", repoId, "repo", payload.Repository.FullName)
			w.WriteHeader(http.StatusOK)
			return
		}

		// Authz: verify installation_id matches github_connections.InstallationId for each service
		var connectionInstallationId string
		if installationId != "" {
			conn, err := app.Supabase.GetGithubConnectionByInstallationId(installationId)
			if err != nil {
				slog.Warn("webhook push: unknown installation", "installation_id", installationId, "repo_id", repoId, "err", err)
				http.Error(w, "unknown installation", http.StatusForbidden)
				return
			}
			connectionInstallationId = conn.InstallationId
			// Verify at least one service belongs to this installation; otherwise authz fail
			authorized := false
			for _, svc := range services {
				if svc.GithubConnectionId == conn.Id {
					authorized = true
					break
				}
			}
			if !authorized {
				slog.Warn("webhook push: installation not authorized for repo services", "installation_id", installationId, "repo_id", repoId)
				http.Error(w, "installation not authorized for this repo", http.StatusForbidden)
				return
			}
		} else {
			slog.Warn("webhook push: missing installation.id, skipping authz check", "repo_id", repoId)
		}

		cloneURL := payload.Repository.CloneURL
		if cloneURL == "" && payload.Repository.FullName != "" {
			cloneURL = fmt.Sprintf("https://github.com/%s.git", payload.Repository.FullName)
		}
		// Get token for private repos if installation present
		accessToken := ""
		if installationId != "" {
			accessToken, _ = app.getInstallationToken(r.Context(), installationId)
			_ = connectionInstallationId // to avoid unused if not used above
		}

		// For each service, clone using its stored RootDir and redeploy
		// Run sequentially for local registry; for scale, use goroutines + WaitGroup.
		for _, svc := range services {
			// Double-check per-service authz if we have installation info
			if installationId != "" && connectionInstallationId != "" {
				// svc already verified belongs to this installation via above; but skip if not
				conn, _ := app.Supabase.GetGithubConnectionByInstallationId(installationId)
				if svc.GithubConnectionId != conn.Id {
					slog.Warn("skipping service not belonging to installation", "service_id", svc.Id, "installation_id", installationId)
					continue
				}
			}

			sanitizedRootDir, err := util.SanitizeRootDir(svc.RootDir)
			if err != nil {
				slog.Error("invalid root_dir stored for service, skipping deploy", "service_id", svc.Id, "root_dir", svc.RootDir, "err", err)
				continue
			}
			imageTag := fmt.Sprintf("local/%s:%s", svc.ResourceName, payload.HeadCommitID)
			if payload.HeadCommitID == "" {
				imageTag = fmt.Sprintf("local/%s:latest", svc.ResourceName)
			}

			builtImage, err := cloneAndBuild(r.Context(), cloneURL, accessToken, sanitizedRootDir, imageTag)
			if err != nil {
				slog.Error("webhook: build failed", "service_id", svc.Id, "root_dir", sanitizedRootDir, "err", err)
				if _, statusErr := app.Supabase.UpdateGithubServiceStatusById(svc.Id, "failed"); statusErr != nil {
					slog.Error("failed to mark service failed after webhook build error", "service_id", svc.Id, "err", statusErr)
				}
				continue
			}
			if builtImage == "" {
				builtImage = imageTag
			}

			// Preserve existing env on webhook redeploys (stored as k8s Secret).
			existingEnvStr, _ := app.Deploy.GetServiceEnv(r.Context(), deploy.Service{
				Namespace: "proj-" + svc.ProjectId,
				Name:      svc.ResourceName,
			})
			existingEnv := map[string][]byte{}
			for k, v := range existingEnvStr {
				existingEnv[k] = []byte(v)
			}
			if err := app.Deploy.CreateService(r.Context(), deploy.Service{
				Namespace: "proj-" + svc.ProjectId,
				Name:      svc.ResourceName,
				Hostname:  svc.PublicDomain,
				Port:      svc.Port,
				Image:     builtImage,
				Env:       existingEnv,
			}); err != nil {
				slog.Error("webhook: deploy failed", "service_id", svc.Id, "err", err)
				if _, statusErr := app.Supabase.UpdateGithubServiceStatusById(svc.Id, "failed"); statusErr != nil {
					slog.Error("failed to mark service failed after deploy error", "service_id", svc.Id, "err", statusErr)
				}
				continue
			}
			if _, err := app.Supabase.UpdateGithubServiceStatusById(svc.Id, "running"); err != nil {
				slog.Error("failed to mark service running after webhook deploy", "service_id", svc.Id, "err", err)
			}
			slog.Info("webhook push: deployed", "service_id", svc.Id, "root_dir", sanitizedRootDir, "image", builtImage)
		}

		w.WriteHeader(http.StatusOK)
		return

	case "installation":
		var payload struct {
			Action       string `json:"action"`
			Installation struct {
				Id int64 `json:"id"`
			} `json:"installation"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		installationId := strconv.FormatInt(payload.Installation.Id, 10)
		if installationId == "0" {
			http.Error(w, "missing installation.id", http.StatusBadRequest)
			return
		}
		switch payload.Action {
		case "created":
			// Webhook alone has no user_id; frontend should call POST /github/connections after install.
			// Here we just log; if you want to auto-create, need user mapping via state.
			slog.Info("github installation created", "installation_id", installationId)
		case "deleted":
			if err := app.Supabase.DeleteGithubConnectionByInstallationId(installationId); err != nil {
				slog.Error("failed to delete github connection on uninstall", "installation_id", installationId, "err", err)
				http.Error(w, "delete failed", http.StatusInternalServerError)
				return
			}
			slog.Info("github installation deleted", "installation_id", installationId)
		default:
			slog.Info("unhandled installation action", "action", payload.Action)
		}
		w.WriteHeader(http.StatusOK)
		return

	default:
		slog.Info("unhandled github event", "event", eventType)
		w.WriteHeader(http.StatusOK)
		return
	}
}
