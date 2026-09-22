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
	"sync"
	"time"

	"github.com/timmyjinks/tysoncloud/deploy"
	"github.com/timmyjinks/tysoncloud/github"
	"github.com/timmyjinks/tysoncloud/store"
	"github.com/timmyjinks/tysoncloud/util"
)

func (app *Application) GithubWebhook(w http.ResponseWriter, r *http.Request) {
	eventType := r.Header.Get("X-GitHub-Event")
	if eventType == "" {
		eventType = r.Header.Get("X-Github-Event")
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 2<<20))
	if err != nil {
		http.Error(w, "could not read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if !app.Github.VerifyWebhookSignature(r.Header.Get("X-Hub-Signature-256"), body) {
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
		pushedBranch := strings.TrimPrefix(payload.Ref, "refs/heads/")
		if pushedBranch == "" || strings.HasPrefix(payload.Ref, "refs/tags/") {
			slog.Info("webhook push: ignoring non-branch ref", "ref", payload.Ref, "repo_id", payload.Repository.Id, "repo", payload.Repository.FullName)
			w.WriteHeader(http.StatusOK)
			return
		}
		if payload.Repository.Id == 0 {
			http.Error(w, "missing repository.id", http.StatusBadRequest)
			return
		}

		repoId := payload.Repository.Id
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

		var connectionInstallationId string
		if installationId != "" {
			conn, err := app.Supabase.GetGithubConnectionByInstallationId(installationId)
			if err != nil {
				slog.Warn("webhook push: unknown installation", "installation_id", installationId, "repo_id", repoId, "err", err)
				http.Error(w, "unknown installation", http.StatusForbidden)
				return
			}
			connectionInstallationId = strconv.FormatInt(conn.InstallationId, 10)
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
		accessToken := ""
		if installationId != "" {
			accessToken, _ = app.Github.GetInstallationToken(r.Context(), installationId)
			_ = connectionInstallationId
		}

		for _, svc := range services {
			svcBranch := strings.TrimSpace(svc.Branch)
			if svcBranch == "" {
				svcBranch = "main"
			}
			if svcBranch != pushedBranch {
				slog.Info("webhook push: ignoring branch mismatch", "service_id", svc.Id, "service_branch", svcBranch, "pushed_branch", pushedBranch, "repo_id", repoId)
				continue
			}
			if installationId != "" && connectionInstallationId != "" {
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
			registryURLHook := app.Github.RegistryURL()
			tag := payload.HeadCommitID
			if tag == "" {
				tag = "latest"
			} else if len(tag) > 12 {
				tag = tag[:12]
			}
			imageTag := app.Github.RegistryTag(registryURLHook, svc.ResourceName, tag)

			logFn := func(line string) {
				app.Github.AppendBuildLog(svc.Id, line)
			}
			emitState := func(to string) {
				line := `[state] ` + to + ` commit=` + tag + ` repo=` + svc.RepoName + ` branch=` + pushedBranch + ` root_dir=` + sanitizedRootDir
				app.Github.AppendBuildLog(svc.Id, line)
				slog.Info("github deploy state", "service_id", svc.Id, "to", to, "commit", tag, "repo", svc.RepoName, "branch", pushedBranch, "root_dir", sanitizedRootDir)
			}
			emitState("building")
			if _, statusErr := app.Supabase.UpdateGithubServiceStatusById(svc.Id, "building"); statusErr != nil {
				slog.Warn("failed to mark service building", "service_id", svc.Id, "err", statusErr)
			}

			// Load the service env up front so the image bakes the same vars it
			// runs with (e.g. build-time API_URL). Ephemeral — never stored.
			pushEnvStr, _ := app.Deploy.GetServiceEnv(r.Context(), deploy.Service{
				Namespace: "proj-" + svc.ProjectId,
				Name:      svc.ResourceName,
			})
			pushEnv := map[string][]byte{}
			for k, v := range pushEnvStr {
				pushEnv[k] = []byte(v)
			}

			builtImage, err := app.Github.CloneAndBuildWithLogs(r.Context(), cloneURL, accessToken, sanitizedRootDir, pushedBranch, imageTag, logFn, pushEnv)
			if err != nil {
				if github.IsInfraBuildError(err) {
					slog.Error("webhook: build failed due to infra unavailable – will retry on next push", "service_id", svc.Id, "root_dir", sanitizedRootDir, "err", err, "hint", "ensure BuildKit is running: docker compose up -d buildkit registry or BUILDKIT_HOST=docker-container://buildkit")
				} else {
					slog.Error("webhook: build failed", "service_id", svc.Id, "root_dir", sanitizedRootDir, "err", err)
				}
				logFn(`[state] building failed: ` + err.Error())
				emitState("failed")
				if _, statusErr := app.Supabase.UpdateGithubServiceStatusById(svc.Id, "failed"); statusErr != nil {
					slog.Error("failed to mark service failed after webhook build error", "service_id", svc.Id, "err", statusErr)
				}
				if strings.Contains(strings.ToLower(err.Error()), "root_dir") {
					slog.Warn("webhook: root_dir not found, check repo path", "service_id", svc.Id, "root_dir", sanitizedRootDir)
				}
				continue
			}
			if builtImage == "" {
				builtImage = imageTag
			}

			emitState("deploying")
			if _, statusErr := app.Supabase.UpdateGithubServiceStatusById(svc.Id, "deploying"); statusErr != nil {
				slog.Warn("failed to mark service deploying", "service_id", svc.Id, "err", statusErr)
			}

			// Reuse the pre-build snapshot so runtime matches the baked image.
			existingEnv := pushEnv
			if err := app.Deploy.CreateService(r.Context(), deploy.Service{
				Namespace: "proj-" + svc.ProjectId,
				Name:      svc.ResourceName,
				Hostname:  svc.PublicDomain,
				Port:      svc.Port,
				Image:     builtImage,
				Env:       existingEnv,
			}); err != nil {
				slog.Error("webhook: deploy failed", "service_id", svc.Id, "err", err)
				logFn(`[state] deploying failed: ` + err.Error())
				emitState("failed")
				if _, statusErr := app.Supabase.UpdateGithubServiceStatusById(svc.Id, "failed"); statusErr != nil {
					slog.Error("failed to mark service failed after deploy error", "service_id", svc.Id, "err", statusErr)
				}
				continue
			}
			emitState("running")
			if _, err := app.Supabase.UpdateGithubServiceStatusById(svc.Id, "running"); err != nil {
				slog.Error("failed to mark service running after webhook deploy", "service_id", svc.Id, "err", err)
			}
			logFn(`[state] running image=` + builtImage)
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
	case "pull_request":
		// Preview-environment workflow: one shared namespace per repo+PR, with
		// every matched github_service deployed into it (monorepo support).
		//
		// Overview:
		//  1. Parse the PR payload (number + head SHA/ref + head repo clone URL).
		//  2. Look up which tysoncloud github_services map to this repo_id —
		//     one PR fans out to every matched service (monorepo support).
		//  3. Deploy each service's PR head into the SHARED namespace
		//     (`util.PreviewNamespaceForRepo(repoId, prNumber)` =
		//     `prev-r<repoId>-pr-<n>`, provisioned via Deploy.CreateProject /
		//     torn down via Deploy.DeleteProject — no `projects` rows are
		//     created), each under its own deterministic preview name/hostname
		//     so `synchronize` overwrites:
		//       namespace  = util.PreviewNamespaceForRepo(repoId, prNumber)
		//       name       = util.PreviewResourceName(svc.ResourceName, prNumber)
		//       hostname   = util.PreviewHostname(svc.Id, prNumber)
		//                  = tc-preview-<shortSvcUuid>-pr-<n>.tysonjenkins.dev
		//  4. Build the PR head with railpack (same path as `push` deploys),
		//     copy THAT service's env (from that service's `proj-<id>` namespace
		//     secret — never stored in the DB) plus PREVIEW_URL, then
		//     Deploy.CreateService the preview in the shared namespace with
		//     Port from the service DB row — i.e. the same Service+Secret+
		//     Deployment+HPA+HTTPRoute steps as a normal service deploy.
		//  5. Track basic info in `preview_environments` (connection, repo,
		//     PR, service, namespace, hostname, image, status — no envs) for
		//     cascade deletes and cleanup lookups.
		//  6. Post a PR comment with the preview URL(s); on failure post the
		//     error + `/preview retry` hint. Retry via `issue_comment` event.
		//
		// Concurrency (decided): last-write-wins. CreateService uses
		// server-side Apply, so concurrent `synchronize` builds safely
		// overwrite each other; no per-preview mutex in v1.
		// Image retention: preview tags (`pr-<n>-<sha>`) accumulate in the
		// registry with nothing tracking them (left as-is for v1).
		var payload struct {
			Action      string `json:"action"`
			Number      int    `json:"number"`
			PullRequest struct {
				Head struct {
					Ref  string `json:"ref"`
					Sha  string `json:"sha"`
					Repo struct {
						CloneURL string `json:"clone_url"`
						FullName string `json:"full_name"`
					} `json:"repo"`
				} `json:"head"`
			} `json:"pull_request"`
			Repository struct {
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
		installationId := strconv.FormatInt(payload.Installation.Id, 10)
		if installationId == "0" {
			http.Error(w, "missing installation.id", http.StatusBadRequest)
			return
		}
		if payload.Repository.Id == 0 {
			http.Error(w, "missing repository.id", http.StatusBadRequest)
			return
		}
		if payload.Number <= 0 {
			http.Error(w, "missing pull_request number", http.StatusBadRequest)
			return
		}

		switch payload.Action {
		case "opened", "synchronize", "reopened":
			// Fall through to async deploy below. Note: webhook must ack
			// fast (<10s), so all build/deploy work happens in goroutines
			// with a background context, not r.Context().
		case "closed":
			// Tear down previews: look up tracked rows (DB) and delete the
			// shared namespace once. Falls back to the deterministic
			// repo+PR namespace when the DB lookup fails. Best-effort;
			// always ack 200.
			services, err := app.Supabase.GetGithubServicesByRepoId(payload.Repository.Id)
			if err != nil {
				slog.Error("preview cleanup: lookup failed", "repo_id", payload.Repository.Id, "err", err)
				http.Error(w, "lookup failed", http.StatusInternalServerError)
				return
			}
			prNumber := payload.Number
			repoFullName := payload.Repository.FullName
			previews, perr := app.Supabase.GetPreviewEnvironmentsByRepoPR(payload.Repository.Id, prNumber)
			if perr != nil {
				slog.Warn("preview cleanup: preview lookup failed, falling back to services", "repo_id", payload.Repository.Id, "pr", prNumber, "err", perr)
			}
			go app.cleanupPRPreviews(installationId, payload.Repository.Id, repoFullName, prNumber, previews, services)
			w.WriteHeader(http.StatusOK)
			return
		default:
			slog.Info("unhandled pull_request action", "action", payload.Action, "pr", payload.Number)
			w.WriteHeader(http.StatusOK)
			return
		}

		services, err := app.Supabase.GetGithubServicesByRepoId(payload.Repository.Id)
		if err != nil {
			slog.Error("preview deploy: lookup failed", "repo_id", payload.Repository.Id, "err", err)
			http.Error(w, "lookup failed", http.StatusInternalServerError)
			return
		}
		if len(services) == 0 {
			slog.Info("preview deploy: no services for repo", "repo_id", payload.Repository.Id, "repo", payload.Repository.FullName)
			// Not linked to any tc project: main comment says so, skip creation.
			if token, _ := app.Github.GetInstallationToken(r.Context(), installationId); token != "" {
				bg, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				app.postPreviewNoticeAsMain(bg, token, payload.Repository.FullName, payload.Repository.Id, payload.Number)
			}
			w.WriteHeader(http.StatusOK)
			return
		}
		conn, err := app.Supabase.GetGithubConnectionByInstallationId(installationId)
		if err != nil {
			slog.Warn("preview deploy: unknown installation", "installation_id", installationId, "repo_id", payload.Repository.Id, "err", err)
			http.Error(w, "unknown installation", http.StatusForbidden)
			return
		}
		authorized := false
		for _, svc := range services {
			if svc.GithubConnectionId == conn.Id {
				authorized = true
				break
			}
		}
		if !authorized {
			slog.Warn("preview deploy: installation not authorized for repo services", "installation_id", installationId, "repo_id", payload.Repository.Id)
			if token, _ := app.Github.GetInstallationToken(r.Context(), installationId); token != "" {
				bg, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				app.postPreviewNoticeAsMain(bg, token, payload.Repository.FullName, payload.Repository.Id, payload.Number)
			}
			http.Error(w, "installation not authorized for this repo", http.StatusForbidden)
			return
		}

		// Resolve the clone source: prefer the PR head repo (handles forks —
		// head CloneURL points at the fork), fall back to the base repo URL.
		headCloneURL := strings.TrimSpace(payload.PullRequest.Head.Repo.CloneURL)
		if headCloneURL == "" {
			headCloneURL = strings.TrimSpace(payload.Repository.CloneURL)
		}
		if headCloneURL == "" && payload.Repository.FullName != "" {
			headCloneURL = fmt.Sprintf("https://github.com/%s.git", payload.Repository.FullName)
		}
		headRef := strings.TrimSpace(payload.PullRequest.Head.Ref)
		headSHA := strings.TrimSpace(payload.PullRequest.Head.Sha)

		// Mint the installation token synchronously (fast) so background
		// goroutines don't depend on the request context.
		accessToken, _ := app.Github.GetInstallationToken(r.Context(), installationId)

		// Fan out to every matched service (monorepo support). Each deploys
		// independently so one root_dir failure doesn't block the others.
		prNumber := payload.Number
		repoFullName := payload.Repository.FullName
		if repoFullName == "" {
			repoFullName = strings.TrimSpace(payload.PullRequest.Head.Repo.FullName)
		}
		for _, svc := range services {
			if svc.GithubConnectionId != conn.Id {
				slog.Warn("preview deploy: skipping service not belonging to installation", "service_id", svc.Id, "installation_id", installationId)
				continue
			}
			go app.deployPRPreview(svc, conn.Id, conn.InstallationId, accessToken, payload.Repository.Id, headCloneURL, headRef, headSHA, repoFullName, prNumber)
		}
		w.WriteHeader(http.StatusOK)
		return

	case "issue_comment":
		// Retry trigger: comment `/preview retry` on a PR re-runs the preview
		// build for that PR. Zero frontend — the PR comment itself is the
		// button. Requires subscribing to `Issue comment` events in the App.
		var cpayload struct {
			Action string `json:"action"`
			Issue  struct {
				Number      int `json:"number"`
				PullRequest *struct {
					URL string `json:"url"`
				} `json:"pull_request"`
			} `json:"issue"`
			Comment struct {
				Body string `json:"body"`
			} `json:"comment"`
			Sender struct {
				Type string `json:"type"`
			} `json:"sender"`
			Repository struct {
				Id       int64  `json:"id"`
				FullName string `json:"full_name"`
				CloneURL string `json:"clone_url"`
			} `json:"repository"`
			Installation struct {
				Id int64 `json:"id"`
			} `json:"installation"`
		}
		if err := json.Unmarshal(body, &cpayload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if cpayload.Action != "created" || cpayload.Issue.PullRequest == nil {
			w.WriteHeader(http.StatusOK)
			return
		}
		if cpayload.Sender.Type == "Bot" {
			w.WriteHeader(http.StatusOK)
			return
		}
		cmd := strings.ToLower(strings.TrimSpace(cpayload.Comment.Body))
		if !strings.Contains(cmd, "/preview retry") && !strings.Contains(cmd, "/preview rebuild") {
			w.WriteHeader(http.StatusOK)
			return
		}
		cInstallationId := strconv.FormatInt(cpayload.Installation.Id, 10)
		if cInstallationId == "0" || cpayload.Repository.Id == 0 || cpayload.Issue.Number <= 0 {
			http.Error(w, "missing installation/repository/issue", http.StatusBadRequest)
			return
		}
		cconn, cerr := app.Supabase.GetGithubConnectionByInstallationId(cInstallationId)
		if cerr != nil {
			slog.Warn("preview retry: unknown installation", "installation_id", cInstallationId, "err", cerr)
			http.Error(w, "unknown installation", http.StatusForbidden)
			return
		}
		cservices, cerr := app.Supabase.GetGithubServicesByRepoId(cpayload.Repository.Id)
		if cerr != nil || len(cservices) == 0 {
			slog.Info("preview retry: no services for repo", "repo_id", cpayload.Repository.Id)
			// Same as the pull_request path: main comment says not linked.
			if token, _ := app.Github.GetInstallationToken(r.Context(), cInstallationId); token != "" {
				bg, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				app.postPreviewNoticeAsMain(bg, token, cpayload.Repository.FullName, cpayload.Repository.Id, cpayload.Issue.Number)
			}
			w.WriteHeader(http.StatusOK)
			return
		}
		ctoken, _ := app.Github.GetInstallationToken(r.Context(), cInstallationId)
		if ctoken == "" {
			slog.Warn("preview retry: no token", "installation_id", cInstallationId)
			http.Error(w, "no token", http.StatusInternalServerError)
			return
		}
		// Transient ack reply (REST has no threaded replies — new top-level
		// comment). Rebuilds below edit the main comment through its states.
		ack := previewRetryAcks[cpayload.Issue.Number%len(previewRetryAcks)]
		if _, err := app.Github.PostPRComment(r.Context(), ctoken, cpayload.Repository.FullName, cpayload.Issue.Number, ack); err != nil {
			slog.Warn("preview retry: failed to post ack", "repo", cpayload.Repository.FullName, "pr", cpayload.Issue.Number, "err", err)
		}
		cheadSHA, cheadRef, cheadCloneURL := app.fetchPRHead(r.Context(), ctoken, cpayload.Repository.FullName, cpayload.Issue.Number)
		if cheadCloneURL == "" {
			cheadCloneURL = strings.TrimSpace(cpayload.Repository.CloneURL)
		}
		if cheadCloneURL == "" && cpayload.Repository.FullName != "" {
			cheadCloneURL = fmt.Sprintf("https://github.com/%s.git", cpayload.Repository.FullName)
		}
		cpr := cpayload.Issue.Number
		crepoFull := cpayload.Repository.FullName
		crepoId := cpayload.Repository.Id
		for _, svc := range cservices {
			if svc.GithubConnectionId != cconn.Id {
				continue
			}
			go app.deployPRPreview(svc, cconn.Id, cconn.InstallationId, ctoken, crepoId, cheadCloneURL, cheadRef, cheadSHA, crepoFull, cpr)
		}
		w.WriteHeader(http.StatusOK)
		return

	default:
		slog.Info("unhandled github event", "event", eventType)
		w.WriteHeader(http.StatusOK)
		return
	}
}

// previewRetryHint is appended to the live comment footer so the PR thread
// itself is the retry button (zero frontend): reply `/preview retry` to rebuild.
const previewRetryHint = "\n\nReply `/preview retry` to rebuild this preview."

// previewCommentMu serializes read-modify-write of the single live comment per
// PR (monorepo fan-out + concurrent synchronize would otherwise race PATCHes).
var (
	previewCommentMu    sync.Mutex
	previewCommentLocks = map[string]*sync.Mutex{}
)

func previewCommentLock(repoId int64, prNumber int) *sync.Mutex {
	key := strconv.FormatInt(repoId, 10) + ":" + strconv.Itoa(prNumber)
	previewCommentMu.Lock()
	defer previewCommentMu.Unlock()
	if m, ok := previewCommentLocks[key]; ok {
		return m
	}
	m := &sync.Mutex{}
	previewCommentLocks[key] = m
	return m
}

// previewMarker identifies the single live comment per PR for adoption.
func previewMarker(repoId int64, prNumber int) string {
	return fmt.Sprintf("<!-- tysoncloud-preview-pr:%d:%d -->", repoId, prNumber)
}

func previewStatusEmoji(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "running":
		return "✅"
	case "failed":
		return "❌"
	case "building", "deploying", "pending":
		return "🏗️"
	case "torn_down":
		return "🧹"
	default:
		return "🏗️"
	}
}

// renderPreviewMainComment builds the single live comment body for a PR from
// all tracked rows. Errors are truncated (never env values — none are stored).
func renderPreviewMainComment(marker, repoFullName string, prNumber int, rows []store.PreviewEnvironmentsTable) string {
	var b strings.Builder
	b.WriteString(marker + "\n")
	b.WriteString(fmt.Sprintf("## TYSONCLOUD preview — PR #%d\n\n", prNumber))
	if len(rows) == 0 {
		b.WriteString("No preview services tracked yet.\n")
		b.WriteString(previewRetryHint)
		return b.String()
	}
	for _, r := range rows {
		name := r.ResourceName
		if name == "" {
			name = r.Namespace
		}
		sha := util.ShortSHA(r.HeadSHA)
		switch strings.ToLower(strings.TrimSpace(r.Status)) {
		case "running":
			url := "https://" + r.Hostname
			fmt.Fprintf(&b, "- %s `%s` (`%s`): %s\n", previewStatusEmoji(r.Status), name, sha, url)
		case "failed":
			fmt.Fprintf(&b, "- %s `%s` failed (`%s`).\n", previewStatusEmoji(r.Status), name, sha)
		default:
			fmt.Fprintf(&b, "- %s `%s` %s (`%s`)…\n", previewStatusEmoji(r.Status), name, strings.ToLower(strings.TrimSpace(r.Status)), sha)
		}
	}
	b.WriteString(previewRetryHint)
	return b.String()
}

func renderPreviewUnlinkedComment(marker string, repoFullName string, prNumber int) string {
	return fmt.Sprintf("%s\n## TYSONCLOUD preview — PR #%d\n\nThis repo is not associated with any TYSONCLOUD project, so no preview was created. Create a GitHub service for this repo in a project to enable previews.%s", marker, prNumber, previewRetryHint)
}

func renderPreviewTornDownComment(marker string, prNumber int) string {
	return fmt.Sprintf("%s\n## TYSONCLOUD preview — PR #%d\n\nPreview environment(s) for PR #%d have been torn down.", marker, prNumber, prNumber)
}

// ensurePreviewMainComment returns the shared live comment id, creating it
// with initialBody when needed (adopting by marker when the DB row is missing,
// recreating on 404).
func (app *Application) ensurePreviewMainComment(ctx context.Context, token, repoFullName string, repoId int64, prNumber int, initialBody string) int64 {
	marker := previewMarker(repoId, prNumber)
	if id := app.Supabase.GetPreviewCommentID(repoId, prNumber); id != 0 {
		return id
	}
	if id := app.Github.FindPRCommentByMarker(ctx, token, repoFullName, prNumber, marker); id != 0 {
		app.Supabase.SetPreviewCommentID(repoId, prNumber, id)
		return id
	}
	id, err := app.Github.PostPRComment(ctx, token, repoFullName, prNumber, initialBody)
	if err != nil {
		slog.Warn("preview: failed to post live comment", "repo", repoFullName, "pr", prNumber, "err", err)
		return 0
	}
	app.Supabase.SetPreviewCommentID(repoId, prNumber, id)
	return id
}

// truncateError caps error text for the live comment. Never env values —
// only build/deploy error strings reach here.
func truncateError(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) > max {
		s = s[:max] + "…"
	}
	return s
}

// upsertPreviewMainComment re-renders the single live comment from DB rows and
// PATCHes it (creating first when needed). Best-effort: logs but never fails
// the deploy.
func (app *Application) upsertPreviewMainComment(ctx context.Context, token, repoFullName string, repoId int64, prNumber int) {
	if token == "" || repoFullName == "" {
		return
	}
	lock := previewCommentLock(repoId, prNumber)
	lock.Lock()
	defer lock.Unlock()
	rows, _ := app.Supabase.GetPreviewEnvironmentsByRepoPR(repoId, prNumber)
	marker := previewMarker(repoId, prNumber)
	body := renderPreviewMainComment(marker, repoFullName, prNumber, rows)
	app.patchPreviewMainComment(ctx, token, repoFullName, repoId, prNumber, marker, body)
}

// upsertPreviewMainCommentWithError renders the live comment from DB rows,
// appends the failure text directly (nothing stored in the DB), and PATCHes
// it. Every error lands in the PR thread.
func (app *Application) upsertPreviewMainCommentWithError(ctx context.Context, token, repoFullName string, repoId int64, prNumber int, errText string) {
	if token == "" || repoFullName == "" {
		return
	}
	lock := previewCommentLock(repoId, prNumber)
	lock.Lock()
	defer lock.Unlock()
	rows, _ := app.Supabase.GetPreviewEnvironmentsByRepoPR(repoId, prNumber)
	marker := previewMarker(repoId, prNumber)
	body := renderPreviewMainComment(marker, repoFullName, prNumber, rows)
	if msg := truncateError(errText, 500); msg != "" {
		body += "\n```\n" + msg + "\n```\n"
	}
	app.patchPreviewMainComment(ctx, token, repoFullName, repoId, prNumber, marker, body)
}

// patchPreviewMainComment creates-or-PATCHes the single live comment.
// Caller must hold the per-PR lock.
func (app *Application) patchPreviewMainComment(ctx context.Context, token, repoFullName string, repoId int64, prNumber int, marker, body string) {
	commentID := app.Supabase.GetPreviewCommentID(repoId, prNumber)
	if commentID == 0 {
		commentID = app.Github.FindPRCommentByMarker(ctx, token, repoFullName, prNumber, marker)
		if commentID != 0 {
			app.Supabase.SetPreviewCommentID(repoId, prNumber, commentID)
		}
	}
	if commentID == 0 {
		id, err := app.Github.PostPRComment(ctx, token, repoFullName, prNumber, body)
		if err != nil {
			slog.Warn("preview: failed to post live comment", "repo", repoFullName, "pr", prNumber, "err", err)
			return
		}
		app.Supabase.SetPreviewCommentID(repoId, prNumber, id)
		return
	}
	if err := app.Github.UpdatePRComment(ctx, token, repoFullName, commentID, body); err != nil {
		if strings.Contains(err.Error(), "404") {
			id, perr := app.Github.PostPRComment(ctx, token, repoFullName, prNumber, body)
			if perr != nil {
				slog.Warn("preview: failed to recreate live comment", "repo", repoFullName, "pr", prNumber, "err", perr)
				return
			}
			app.Supabase.SetPreviewCommentID(repoId, prNumber, id)
			return
		}
		slog.Warn("preview: failed to update live comment", "repo", repoFullName, "pr", prNumber, "err", err)
	}
}

// postPreviewNoticeAsMain posts the unlinked notice as the main comment
// (deduped by marker so pushes don't spam), skipping creation entirely.
func (app *Application) postPreviewNoticeAsMain(ctx context.Context, token, repoFullName string, repoId int64, prNumber int) {
	if token == "" || repoFullName == "" {
		return
	}
	lock := previewCommentLock(repoId, prNumber)
	lock.Lock()
	defer lock.Unlock()
	marker := previewMarker(repoId, prNumber)
	if id := app.Github.FindPRCommentByMarker(ctx, token, repoFullName, prNumber, marker); id != 0 {
		return
	}
	body := renderPreviewUnlinkedComment(marker, repoFullName, prNumber)
	if _, err := app.Github.PostPRComment(ctx, token, repoFullName, prNumber, body); err != nil {
		slog.Warn("preview: failed to post unlinked notice", "repo", repoFullName, "pr", prNumber, "err", err)
	}
}

var previewRetryAcks = []string{"Roger.", "Noted.", "Acknowledged."}

// deployPRPreview builds the PR head with railpack and deploys it as a
// preview into the SHARED per-repo+PR namespace, alongside the other
// services for that repo.
//
// Same steps as a normal service deploy (Deploy.CreateService:
// Service+Secret+Deployment+HPA+HTTPRoute) with Port taken from the service
// DB row. Env is inherited per-service: ALL keys are copied from THAT
// service's secret in the parent `proj-<projectId>` namespace, plus
// PREVIEW_URL=<this preview's public URL> is injected — never stored in the
// DB. Basic info (connection, repo, PR, service, namespace, hostname, image,
// status) is tracked in `preview_environments`. Runs in a background
// goroutine — never block the webhook on this.
func (app *Application) deployPRPreview(svc store.GithubServicesTable, connId string, installationId int64, accessToken string, repoId int64, headCloneURL, headRef, headSHA, repoFullName string, prNumber int) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	previewNamespace := util.PreviewNamespaceForRepo(repoId, prNumber)
	previewName := util.PreviewResourceName(svc.ResourceName, prNumber)
	hostname := util.PreviewHostname(svc.Id, prNumber)
	previewURL := "https://" + hostname
	shortSHA := util.ShortSHA(headSHA)
	previewLogID := fmt.Sprintf("%s-pr-%d", svc.Id, prNumber)

	logFn := func(line string) {
		app.Github.AppendBuildLog(previewLogID, line)
	}
	commentToken := accessToken
	if commentToken == "" {
		commentToken, _ = app.Github.GetInstallationToken(ctx, strconv.FormatInt(installationId, 10))
	}
	// Single live comment per PR: every state change re-renders it from DB.
	// Failures append the error text straight into the PATCH body (nothing
	// stored in the DB) so every error lands in the PR thread.
	refreshMain := func() {
		app.upsertPreviewMainComment(ctx, commentToken, repoFullName, repoId, prNumber)
	}
	refreshMainWithError := func(errText string) {
		app.upsertPreviewMainCommentWithError(ctx, commentToken, repoFullName, repoId, prNumber, errText)
	}
	track := func(status, image string) {
		_, terr := app.Supabase.UpsertPreviewEnvironment(store.PreviewEnvironmentsTable{
			GithubConnectionId: connId,
			GithubServiceId:    svc.Id,
			RepoId:             repoId,
			RepoName:           repoFullName,
			PrNumber:           prNumber,
			HeadSHA:            headSHA,
			HeadRef:            headRef,
			Namespace:          previewNamespace,
			ResourceName:       previewName,
			Hostname:           hostname,
			Image:              image,
			Status:             status,
		})
		if terr != nil {
			slog.Warn("preview: failed to track preview environment", "service_id", svc.Id, "pr", prNumber, "err", terr)
		}
	}

	sanitizedRootDir, err := util.SanitizeRootDir(svc.RootDir)
	if err != nil {
		slog.Error("preview: invalid root_dir, skipping", "service_id", svc.Id, "root_dir", svc.RootDir, "err", err)
		logFn(`[state] preview failed: invalid root_dir: ` + err.Error())
		track("failed", "")
		refreshMainWithError(err.Error())
		return
	}

	// QUESTION (image retention): tags accumulate as `pr-<n>-<sha>` with no
	// GC. If registry disk becomes an issue, consider overwriting a single
	// `pr-<n>` tag instead — trades traceability for bounded storage.
	imageTag := app.Github.RegistryTag(app.Github.RegistryURL(), svc.ResourceName, fmt.Sprintf("pr-%d-%s", prNumber, shortSHA))

	track("building", imageTag)
	refreshMain()
	logFn(`[state] building preview pr=` + strconv.Itoa(prNumber) + ` sha=` + shortSHA + ` service=` + svc.Name + ` root_dir=` + sanitizedRootDir + ` namespace=` + previewNamespace + ` port=` + strconv.FormatInt(int64(svc.Port), 10))
	slog.Info("preview deploy: building", "service_id", svc.Id, "pr", prNumber, "sha", shortSHA, "root_dir", sanitizedRootDir, "namespace", previewNamespace, "port", svc.Port)

	token := accessToken
	if token == "" {
		token, _ = app.Github.GetInstallationToken(ctx, strconv.FormatInt(installationId, 10))
	}

	// Inherit THAT service's env so the preview mirrors its prod config
	// (e.g. the frontend preview gets the frontend's env — never merged
	// across services). Loaded BEFORE the build so the image bakes the same
	// vars it runs with (e.g. build-time API_URL). Ephemeral: read from the
	// parent K8s secret on every build, never stored in the DB. PREVIEW_URL
	// is injected so the app can discover its own public URL (at build AND
	// runtime).
	existingEnvStr, envErr := app.Deploy.GetServiceEnv(ctx, deploy.Service{
		Namespace: "proj-" + svc.ProjectId,
		Name:      svc.ResourceName,
	})
	if envErr != nil {
		slog.Error("preview: parent env lookup failed", "service_id", svc.Id, "pr", prNumber, "err", envErr)
		logFn(`[state] preview env lookup failed: ` + envErr.Error())
		track("failed", imageTag)
		refreshMainWithError(envErr.Error())
		return
	}
	existingEnv := map[string][]byte{}
	for k, v := range existingEnvStr {
		existingEnv[k] = []byte(v)
	}
	existingEnv["PREVIEW_URL"] = []byte(previewURL)
	slog.Info("preview deploy: env inherited", "service_id", svc.Id, "pr", prNumber, "env_keys", len(existingEnv), "port", svc.Port)

	builtImage, err := app.Github.CloneAndBuildPRWithLogs(ctx, headCloneURL, token, sanitizedRootDir, headRef, headSHA, imageTag, logFn, existingEnv)
	if err != nil {
		if github.IsInfraBuildError(err) {
			slog.Error("preview: build failed (infra unavailable)", "service_id", svc.Id, "pr", prNumber, "err", err)
		} else {
			slog.Error("preview: build failed", "service_id", svc.Id, "pr", prNumber, "err", err)
		}
		logFn(`[state] preview build failed: ` + err.Error())
		track("failed", imageTag)
		refreshMainWithError(err.Error())
		return
	}
	if builtImage == "" {
		builtImage = imageTag
	}

	// Provision the shared per-repo+PR namespace (Deploy.CreateProject is
	// K8s-only: namespace + network policy, no `projects` row). Idempotent
	// under monorepo fan-out — concurrent goroutines Apply the same object.
	if err := app.Deploy.CreateProject(ctx, previewNamespace); err != nil {
		slog.Error("preview: namespace setup failed", "service_id", svc.Id, "pr", prNumber, "namespace", previewNamespace, "err", err)
		logFn(`[state] preview namespace failed: ` + err.Error())
		track("failed", builtImage)
		refreshMainWithError(err.Error())
		return
	}

	track("deploying", builtImage)
	refreshMain()
	logFn(`[state] deploying preview image=` + builtImage + ` namespace=` + previewNamespace + ` port=` + strconv.FormatInt(int64(svc.Port), 10))
	// Same steps as a normal service deploy; only the hostname (preview URL)
	// differs from prod. Port comes from the service DB row.
	if err := app.Deploy.CreateService(ctx, deploy.Service{
		Namespace: previewNamespace,
		Name:      previewName,
		Hostname:  hostname,
		Port:      svc.Port,
		Image:     builtImage,
		Env:       existingEnv,
	}); err != nil {
		slog.Error("preview: deploy failed", "service_id", svc.Id, "pr", prNumber, "err", err)
		logFn(`[state] preview deploy failed: ` + err.Error())
		track("failed", builtImage)
		refreshMainWithError(err.Error())
		return
	}

	track("running", builtImage)
	refreshMain()
	logFn(`[state] preview running url=` + previewURL + ` image=` + builtImage)
	slog.Info("preview deploy: running", "service_id", svc.Id, "pr", prNumber, "preview", previewName, "namespace", previewNamespace, "url", previewURL, "image", builtImage)
}

// cleanupPRPreviews deletes the SHARED preview namespace for a closed PR.
// Tracked rows are the source of namespaces (deduplicated — every row for
// the repo+PR shares one namespace); when the DB lookup failed, falls back
// to the deterministic repo+PR namespace. Each delete is best-effort. Rows
// are marked torn_down (kept for history). The live comment is edited to the
// torn-down state — or left untouched when nothing was ever tracked and no
// main comment exists (e.g. never-linked repos).
func (app *Application) cleanupPRPreviews(installationId string, repoId int64, repoFullName string, prNumber int, previews []store.PreviewEnvironmentsTable, services []store.GithubServicesTable) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	namespaces := map[string]bool{}
	for _, p := range previews {
		if strings.TrimSpace(p.Namespace) == "" {
			continue
		}
		namespaces[p.Namespace] = true
	}
	if len(namespaces) == 0 {
		// Deterministic fallback: one shared namespace per repo+PR.
		// `services` is only used to detect the never-linked case below
		// (via previews being empty); the namespace itself derives from
		// the repo, not any service.
		_ = services
		namespaces[util.PreviewNamespaceForRepo(repoId, prNumber)] = true
	}
	for namespace := range namespaces {
		if err := app.Deploy.DeleteProject(ctx, namespace); err != nil {
			slog.Warn("preview cleanup: delete failed", "repo_id", repoId, "pr", prNumber, "namespace", namespace, "err", err)
			continue
		}
		slog.Info("preview cleanup: deleted", "repo_id", repoId, "pr", prNumber, "namespace", namespace)
	}
	for _, p := range previews {
		if p.Id != "" {
			_ = app.Supabase.UpdatePreviewEnvironmentStatus(p.Id, "torn_down")
		}
	}

	// Best-effort teardown: edit the live comment (or skip when nothing was
	// ever tracked and no main comment exists — e.g. never-linked repos).
	if token, err := app.Github.GetInstallationToken(ctx, installationId); err == nil && token != "" && repoFullName != "" {
		app.markPreviewTornDown(ctx, token, repoId, repoFullName, prNumber)
	}
}

// markPreviewTornDown edits the live comment to the torn-down state. Skips
// silently when no rows were ever tracked and no main comment exists.
func (app *Application) markPreviewTornDown(ctx context.Context, token string, repoId int64, repoFullName string, prNumber int) {
	if token == "" || repoFullName == "" {
		return
	}
	lock := previewCommentLock(repoId, prNumber)
	lock.Lock()
	defer lock.Unlock()
	marker := previewMarker(repoId, prNumber)
	commentID := app.Supabase.GetPreviewCommentID(repoId, prNumber)
	if commentID == 0 {
		commentID = app.Github.FindPRCommentByMarker(ctx, token, repoFullName, prNumber, marker)
		if commentID == 0 {
			rows, _ := app.Supabase.GetPreviewEnvironmentsByRepoPR(repoId, prNumber)
			if len(rows) == 0 {
				return
			}
		} else {
			app.Supabase.SetPreviewCommentID(repoId, prNumber, commentID)
		}
	}
	body := renderPreviewTornDownComment(marker, prNumber)
	if commentID == 0 {
		id, err := app.Github.PostPRComment(ctx, token, repoFullName, prNumber, body)
		if err != nil {
			slog.Warn("preview: failed to post torn-down comment", "repo", repoFullName, "pr", prNumber, "err", err)
			return
		}
		app.Supabase.SetPreviewCommentID(repoId, prNumber, id)
		return
	}
	if err := app.Github.UpdatePRComment(ctx, token, repoFullName, commentID, body); err != nil {
		slog.Warn("preview: failed to update torn-down comment", "repo", repoFullName, "pr", prNumber, "err", err)
	}
}

// fetchPRHead resolves the current head SHA/ref/clone URL for a PR via the
// GitHub API (issue_comment payloads don't include PR head info).
func (app *Application) fetchPRHead(ctx context.Context, installationToken, repoFullName string, prNumber int) (sha, ref, cloneURL string) {
	if installationToken == "" || strings.TrimSpace(repoFullName) == "" || prNumber <= 0 {
		return "", "", ""
	}
	url := fmt.Sprintf("https://api.github.com/repos/%s/pulls/%d", strings.TrimSpace(repoFullName), prNumber)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", "", ""
	}
	req.Header.Set("Authorization", "Bearer "+installationToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Warn("preview retry: failed to fetch PR head", "repo", repoFullName, "pr", prNumber, "err", err)
		return "", "", ""
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		slog.Warn("preview retry: PR head lookup failed", "repo", repoFullName, "pr", prNumber, "status", resp.StatusCode, "body", string(body))
		return "", "", ""
	}
	var out struct {
		Head struct {
			Ref  string `json:"ref"`
			Sha  string `json:"sha"`
			Repo struct {
				CloneURL string `json:"clone_url"`
				FullName string `json:"full_name"`
			} `json:"repo"`
		} `json:"head"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", "", ""
	}
	return strings.TrimSpace(out.Head.Sha), strings.TrimSpace(out.Head.Ref), strings.TrimSpace(out.Head.Repo.CloneURL)
}
