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

func (app *Application) withoutPreviewCopies(services []store.GithubServicesTable) []store.GithubServicesTable {
	sets := map[string]map[string]bool{}
	kept := make([]store.GithubServicesTable, 0, len(services))
	for _, svc := range services {
		set, ok := sets[svc.ProjectId]
		if !ok {
			var err error
			set, err = app.Supabase.GetPreviewCopyIDs(svc.ProjectId)
			if err != nil {
				slog.Warn("preview: copy lookup failed, keeping service", "service_id", svc.Id, "err", err)
				set = map[string]bool{}
			}
			sets[svc.ProjectId] = set
		}
		if set[svc.Id] {
			continue
		}
		kept = append(kept, svc)
	}
	return kept
}

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
			} `json:"repository"`
			Installation struct {
				Id int64 `json:"id"`
			} `json:"installation"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			http.Error(w, "invalid payload", http.StatusBadRequest)
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

		if payload.Installation.Id == 0 {
			slog.Warn("webhook push: missing installation.id", "repo_id", payload.Repository.Id)
			http.Error(w, "missing installation.id", http.StatusForbidden)
			return
		}

		repoId := payload.Repository.Id
		installationId := strconv.FormatInt(payload.Installation.Id, 10)

		services, err := app.Supabase.GetGithubServicesByRepoId(repoId)
		if err != nil {
			slog.Error("failed to lookup github services by repo_id", "repo_id", repoId, "err", err)
			http.Error(w, "lookup failed", http.StatusInternalServerError)
			return
		}
		services = app.withoutPreviewCopies(services)
		if len(services) == 0 {
			slog.Info("webhook push: no services for repo", "repo_id", repoId, "repo", payload.Repository.FullName)
			w.WriteHeader(http.StatusOK)
			return
		}

		conn, err := app.Supabase.GetGithubConnectionByInstallationId(installationId)
		if err != nil {
			slog.Warn("webhook push: unknown installation", "installation_id", installationId, "repo_id", repoId, "err", err)
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
			slog.Warn("webhook push: installation not authorized for repo services", "installation_id", installationId, "repo_id", repoId)
			http.Error(w, "installation not authorized for this repo", http.StatusForbidden)
			return
		}

		accessToken, err := app.Github.GetInstallationToken(r.Context(), installationId)
		if err != nil || accessToken == "" {
			slog.Error("webhook push: failed to mint installation token", "installation_id", installationId, "repo_id", repoId, "err", err)
			http.Error(w, "could not authenticate to github", http.StatusInternalServerError)
			return
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
			if svc.GithubConnectionId != conn.Id {
				slog.Warn("skipping service not belonging to installation", "service_id", svc.Id, "installation_id", installationId)
				continue
			}
			cloneURL, err := util.GithubCloneURL(svc.RepoName)
			if err != nil {
				slog.Error("invalid repo_name stored for service, skipping deploy", "service_id", svc.Id, "repo_name", svc.RepoName, "err", err)
				continue
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
			http.Error(w, "invalid payload", http.StatusBadRequest)
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
		var payload struct {
			Action      string `json:"action"`
			Number      int    `json:"number"`
			PullRequest struct {
				Head struct {
					Ref  string `json:"ref"`
					Sha  string `json:"sha"`
					Repo struct {
						FullName string `json:"full_name"`
					} `json:"repo"`
				} `json:"head"`
			} `json:"pull_request"`
			Repository struct {
				Id       int64  `json:"id"`
				FullName string `json:"full_name"`
			} `json:"repository"`
			Installation struct {
				Id int64 `json:"id"`
			} `json:"installation"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			http.Error(w, "invalid payload", http.StatusBadRequest)
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
		case "closed":
			prNumber := payload.Number
			repoFullName := payload.Repository.FullName
			previews, _ := app.Supabase.GetPreviewEnvironmentViewsByRepoPR(payload.Repository.Id, prNumber)
			go app.cleanupPRPreviews(installationId, payload.Repository.Id, repoFullName, prNumber, previews)
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
		services = app.withoutPreviewCopies(services)
		if len(services) == 0 {
			slog.Info("preview deploy: no services for repo", "repo_id", payload.Repository.Id, "repo", payload.Repository.FullName)
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

		headRepo := strings.TrimSpace(payload.PullRequest.Head.Repo.FullName)
		if headRepo == "" {
			headRepo = payload.Repository.FullName
		}
		headCloneURL, err := util.GithubCloneURL(headRepo)
		if err != nil {
			slog.Warn("preview deploy: invalid head repo", "repo_id", payload.Repository.Id, "head_repo", headRepo, "err", err)
			http.Error(w, "invalid head repository", http.StatusBadRequest)
			return
		}
		headRef := strings.TrimSpace(payload.PullRequest.Head.Ref)
		headSHA := strings.TrimSpace(payload.PullRequest.Head.Sha)

		accessToken, _ := app.Github.GetInstallationToken(r.Context(), installationId)

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
			go app.deployPRPreview(svc, conn.InstallationId, accessToken, payload.Repository.Id, headCloneURL, headRef, headSHA, repoFullName, prNumber)
		}
		previewNamespace := util.PreviewNamespaceForRepo(payload.Repository.Id, prNumber)
		copied := map[string]bool{}
		for _, svc := range services {
			if svc.GithubConnectionId != conn.Id || copied[svc.ProjectId] {
				continue
			}
			copied[svc.ProjectId] = true
			go app.copyProjectToPreview(svc.ProjectId, payload.Repository.Id, prNumber, previewNamespace)
		}
		w.WriteHeader(http.StatusOK)
		return

	case "issue_comment":
		var cpayload struct {
			Action string `json:"action"`
			Issue  struct {
				Number      int `json:"number"`
				PullRequest *struct {
					URL string `json:"url"`
				} `json:"pull_request"`
			} `json:"issue"`
			Comment struct {
				Id   int64  `json:"id"`
				Body string `json:"body"`
			} `json:"comment"`
			Sender struct {
				Type string `json:"type"`
			} `json:"sender"`
			Repository struct {
				Id       int64  `json:"id"`
				FullName string `json:"full_name"`
			} `json:"repository"`
			Installation struct {
				Id int64 `json:"id"`
			} `json:"installation"`
		}
		if err := json.Unmarshal(body, &cpayload); err != nil {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}
		if cpayload.Action != "edited" || cpayload.Issue.PullRequest == nil {
			w.WriteHeader(http.StatusOK)
			return
		}
		if cpayload.Sender.Type == "Bot" {
			w.WriteHeader(http.StatusOK)
			return
		}
		cInstallationId := strconv.FormatInt(cpayload.Installation.Id, 10)
		if cInstallationId == "0" || cpayload.Repository.Id == 0 || cpayload.Issue.Number <= 0 {
			http.Error(w, "missing installation/repository/issue", http.StatusBadRequest)
			return
		}
		ctoken, _ := app.Github.GetInstallationToken(r.Context(), cInstallationId)
		if ctoken == "" {
			slog.Warn("preview retry: no token", "installation_id", cInstallationId)
			http.Error(w, "no token", http.StatusInternalServerError)
			return
		}
		marker := previewMarker(cpayload.Repository.Id, cpayload.Issue.Number)
		ours := false
		if stored := app.Supabase.GetPreviewCommentIDByRepoPR(cpayload.Repository.Id, cpayload.Issue.Number); stored != 0 {
			ours = stored == cpayload.Comment.Id
		} else if found := app.Github.FindPRCommentByMarker(r.Context(), ctoken, cpayload.Repository.FullName, cpayload.Issue.Number, marker); found != 0 {
			app.Supabase.SetPreviewCommentIDByRepoPR(cpayload.Repository.Id, cpayload.Issue.Number, found)
			ours = found == cpayload.Comment.Id
		}
		if !ours {
			w.WriteHeader(http.StatusOK)
			return
		}
		if !strings.Contains(strings.ToLower(cpayload.Comment.Body), "- [x]") {
			w.WriteHeader(http.StatusOK)
			return
		}
		slog.Info("preview: retry checkbox checked", "repo", cpayload.Repository.FullName, "pr", cpayload.Issue.Number)
		cconn, cerr := app.Supabase.GetGithubConnectionByInstallationId(cInstallationId)
		if cerr != nil {
			slog.Warn("preview retry: unknown installation", "installation_id", cInstallationId, "err", cerr)
			http.Error(w, "unknown installation", http.StatusForbidden)
			return
		}
		cservices, cerr := app.Supabase.GetGithubServicesByRepoId(cpayload.Repository.Id)
		if cerr != nil {
			slog.Info("preview retry: lookup failed", "repo_id", cpayload.Repository.Id, "err", cerr)
			http.Error(w, "lookup failed", http.StatusInternalServerError)
			return
		}
		cservices = app.withoutPreviewCopies(cservices)
		if len(cservices) == 0 {
			slog.Info("preview retry: no services for repo", "repo_id", cpayload.Repository.Id)
			bg, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			app.postPreviewNoticeAsMain(bg, ctoken, cpayload.Repository.FullName, cpayload.Repository.Id, cpayload.Issue.Number)
			w.WriteHeader(http.StatusOK)
			return
		}
		cheadSHA, cheadRef, cheadRepo := app.fetchPRHead(r.Context(), ctoken, cpayload.Repository.FullName, cpayload.Issue.Number)
		if cheadRepo == "" {
			cheadRepo = cpayload.Repository.FullName
		}
		cheadCloneURL, cerr := util.GithubCloneURL(cheadRepo)
		if cerr != nil {
			slog.Warn("preview retry: invalid head repo", "repo_id", cpayload.Repository.Id, "head_repo", cheadRepo, "err", cerr)
			http.Error(w, "invalid head repository", http.StatusBadRequest)
			return
		}
		cpr := cpayload.Issue.Number
		crepoFull := cpayload.Repository.FullName
		crepoId := cpayload.Repository.Id
		for _, svc := range cservices {
			if svc.GithubConnectionId != cconn.Id {
				continue
			}
			go app.deployPRPreview(svc, cconn.InstallationId, ctoken, crepoId, cheadCloneURL, cheadRef, cheadSHA, crepoFull, cpr)
		}
		cpreviewNamespace := util.PreviewNamespaceForRepo(crepoId, cpr)
		ccopied := map[string]bool{}
		for _, svc := range cservices {
			if svc.GithubConnectionId != cconn.Id || ccopied[svc.ProjectId] {
				continue
			}
			ccopied[svc.ProjectId] = true
			go app.copyProjectToPreview(svc.ProjectId, crepoId, cpr, cpreviewNamespace)
		}
		w.WriteHeader(http.StatusOK)
		return

	default:
		slog.Info("unhandled github event", "event", eventType)
		w.WriteHeader(http.StatusOK)
		return
	}
}

const previewRetryBox = "\n\n- [ ] 🔄 Retry preview — check this box to rebuild"

var (
	previewErrMu sync.Mutex
	previewErrs  = map[string]string{}
)

func previewErrKey(repoId int64, prNumber int, serviceId string) string {
	return strconv.FormatInt(repoId, 10) + ":" + strconv.Itoa(prNumber) + ":" + serviceId
}

func recordPreviewError(repoId int64, prNumber int, serviceId, errText string) {
	if strings.TrimSpace(errText) == "" {
		return
	}
	previewErrMu.Lock()
	defer previewErrMu.Unlock()
	previewErrs[previewErrKey(repoId, prNumber, serviceId)] = truncateError(errText, 2000)
}

func clearPreviewError(repoId int64, prNumber int, serviceId string) {
	previewErrMu.Lock()
	defer previewErrMu.Unlock()
	delete(previewErrs, previewErrKey(repoId, prNumber, serviceId))
}

func snapshotPreviewErrors(repoId int64, prNumber int, views []store.PreviewEnvironmentView) map[string]string {
	previewErrMu.Lock()
	defer previewErrMu.Unlock()
	out := make(map[string]string, len(views))
	for i := range views {
		if msg, ok := previewErrs[previewErrKey(repoId, prNumber, views[i].Service.SourceServiceId)]; ok {
			out[views[i].Service.SourceServiceId] = msg
		}
	}
	return out
}

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

type previewCommentLine struct {
	ServiceID string
	Name      string
	Hostname  string
	Status    string
}

func renderPreviewMainComment(marker, repoFullName string, prNumber int, sha string, lines []previewCommentLine, errs map[string]string) string {
	var b strings.Builder
	b.WriteString(marker + "\n")
	b.WriteString(fmt.Sprintf("## TYSONCLOUD preview — PR #%d\n\n", prNumber))
	if len(lines) == 0 {
		b.WriteString("No preview services tracked yet.\n")
		b.WriteString(previewRetryBox)
		return b.String()
	}
	short := util.ShortSHA(sha)
	for _, l := range lines {
		name := l.Name
		host := strings.TrimSpace(l.Hostname)
		if _, failed := errs[l.ServiceID]; failed {
			fmt.Fprintf(&b, "- %s `%s` failed (`%s`).\n", previewStatusEmoji("failed"), name, short)
			if host != "" {
				fmt.Fprintf(&b, "  `%s`\n", host)
			}
			if msg := truncateError(errs[l.ServiceID], 500); msg != "" {
				b.WriteString("  ```\n  " + strings.ReplaceAll(msg, "\n", "\n  ") + "\n  ```\n")
			}
			continue
		}
		url := "https://" + host
		fmt.Fprintf(&b, "- %s `%s` (`%s`): %s\n", previewStatusEmoji(l.Status), name, short, url)
		if host != "" {
			fmt.Fprintf(&b, "  `%s`\n", host)
		}
	}
	b.WriteString(previewRetryBox)
	return b.String()
}

func renderPreviewUnlinkedComment(marker string, repoFullName string, prNumber int) string {
	return fmt.Sprintf("%s\n## TYSONCLOUD preview — PR #%d\n\nThis repo is not associated with any TYSONCLOUD project, so no preview was created. Create a GitHub service for this repo in a project to enable previews.%s", marker, prNumber, previewRetryBox)
}

func renderPreviewTornDownComment(marker string, prNumber int) string {
	return fmt.Sprintf("%s\n## TYSONCLOUD preview — PR #%d\n\nPreview environment(s) for PR #%d have been torn down.", marker, prNumber, prNumber)
}

func (app *Application) ensurePreviewMainComment(ctx context.Context, token, repoFullName string, repoId int64, prNumber int, initialBody string) int64 {
	marker := previewMarker(repoId, prNumber)
	if id := app.Supabase.GetPreviewCommentIDByRepoPR(repoId, prNumber); id != 0 {
		return id
	}
	if id := app.Github.FindPRCommentByMarker(ctx, token, repoFullName, prNumber, marker); id != 0 {
		app.Supabase.SetPreviewCommentIDByRepoPR(repoId, prNumber, id)
		return id
	}
	id, err := app.Github.PostPRComment(ctx, token, repoFullName, prNumber, initialBody)
	if err != nil {
		slog.Warn("preview: failed to post live comment", "repo", repoFullName, "pr", prNumber, "err", err)
		return 0
	}
	app.Supabase.SetPreviewCommentIDByRepoPR(repoId, prNumber, id)
	return id
}

func truncateError(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) > max {
		s = s[:max] + "…"
	}
	return s
}

func (app *Application) upsertPreviewMainComment(ctx context.Context, token, repoFullName string, repoId int64, prNumber int) {
	if token == "" || repoFullName == "" {
		return
	}
	lock := previewCommentLock(repoId, prNumber)
	lock.Lock()
	defer lock.Unlock()
	views, _ := app.Supabase.GetPreviewEnvironmentViewsByRepoPR(repoId, prNumber)
	errs := snapshotPreviewErrors(repoId, prNumber, views)
	headSHA, _, _ := app.fetchPRHead(ctx, token, repoFullName, prNumber)
	lines := make([]previewCommentLine, 0, len(views))
	for _, v := range views {
		src, err := app.Supabase.GetPreviewEnvironmentSource(v.Service.SourceType, v.Service.SourceServiceId)
		if err != nil {
			slog.Warn("preview comment: source lookup failed", "env_id", v.Environment.Id, "source", v.Service.SourceServiceId, "err", err)
			continue
		}
		lines = append(lines, previewCommentLine{
			ServiceID: v.Service.SourceServiceId,
			Name:      src.Name,
			Hostname:  util.PreviewHostname(v.Service.SourceServiceId, prNumber),
			Status:    src.Status,
		})
	}
	marker := previewMarker(repoId, prNumber)
	body := renderPreviewMainComment(marker, repoFullName, prNumber, headSHA, lines, errs)
	app.patchPreviewMainComment(ctx, token, repoFullName, repoId, prNumber, marker, body)
}

func (app *Application) upsertPreviewMainCommentWithError(ctx context.Context, token, repoFullName string, repoId int64, prNumber int, serviceId, errText string) {
	if token == "" || repoFullName == "" {
		return
	}
	recordPreviewError(repoId, prNumber, serviceId, errText)
	app.upsertPreviewMainComment(ctx, token, repoFullName, repoId, prNumber)
}

func (app *Application) patchPreviewMainComment(ctx context.Context, token, repoFullName string, repoId int64, prNumber int, marker, body string) {
	commentID := app.Supabase.GetPreviewCommentIDByRepoPR(repoId, prNumber)
	if commentID == 0 {
		commentID = app.Github.FindPRCommentByMarker(ctx, token, repoFullName, prNumber, marker)
		if commentID != 0 {
			app.Supabase.SetPreviewCommentIDByRepoPR(repoId, prNumber, commentID)
		}
	}
	if commentID == 0 {
		id, err := app.Github.PostPRComment(ctx, token, repoFullName, prNumber, body)
		if err != nil {
			slog.Warn("preview: failed to post live comment", "repo", repoFullName, "pr", prNumber, "err", err)
			return
		}
		app.Supabase.SetPreviewCommentIDByRepoPR(repoId, prNumber, id)
		return
	}
	if err := app.Github.UpdatePRComment(ctx, token, repoFullName, commentID, body); err != nil {
		if strings.Contains(err.Error(), "404") {
			id, perr := app.Github.PostPRComment(ctx, token, repoFullName, prNumber, body)
			if perr != nil {
				slog.Warn("preview: failed to recreate live comment", "repo", repoFullName, "pr", prNumber, "err", perr)
				return
			}
			app.Supabase.SetPreviewCommentIDByRepoPR(repoId, prNumber, id)
			return
		}
		slog.Warn("preview: failed to update live comment", "repo", repoFullName, "pr", prNumber, "err", err)
	}
}

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

func (app *Application) deployPRPreview(svc store.GithubServicesTable, installationId int64, accessToken string, repoId int64, headCloneURL, headRef, headSHA, repoFullName string, prNumber int) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	ownerId, err := app.Supabase.GetProjectOwnerId(svc.ProjectId)
	if err != nil {
		slog.Error("preview: owner lookup failed", "service_id", svc.Id, "pr", prNumber, "err", err)
		return
	}
	copy, err := app.Supabase.EnsureGithubServiceCopy(ownerId, svc, prNumber)
	if err != nil {
		slog.Error("preview: copy failed", "service_id", svc.Id, "pr", prNumber, "err", err)
		return
	}

	previewNamespace := util.PreviewNamespaceForRepo(repoId, prNumber)
	previewName := util.PreviewResourceName(copy.ResourceName, prNumber)
	hostname := util.PreviewHostname(copy.Id, prNumber)
	previewURL := "https://" + hostname
	shortSHA := util.ShortSHA(headSHA)
	previewLogID := fmt.Sprintf("%s-pr-%d", copy.Id, prNumber)

	logFn := func(line string) {
		app.Github.AppendBuildLog(previewLogID, line)
	}
	commentToken := accessToken
	if commentToken == "" {
		commentToken, _ = app.Github.GetInstallationToken(ctx, strconv.FormatInt(installationId, 10))
	}
	refreshMain := func() {
		tok := commentToken
		if tok == "" {
			tok, _ = app.Github.GetInstallationToken(ctx, strconv.FormatInt(installationId, 10))
		}
		app.upsertPreviewMainComment(ctx, tok, repoFullName, repoId, prNumber)
	}
	refreshMainWithError := func(errText string) {
		tok := commentToken
		if tok == "" {
			tok, _ = app.Github.GetInstallationToken(ctx, strconv.FormatInt(installationId, 10))
		}
		app.upsertPreviewMainCommentWithError(ctx, tok, repoFullName, repoId, prNumber, copy.Id, errText)
	}
	setStatus := func(status string) {
		if _, serr := app.Supabase.UpdateGithubServiceStatusById(copy.Id, status); serr != nil {
			slog.Warn("preview: copy status update failed", "copy_id", copy.Id, "status", status, "err", serr)
		}
	}
	track := func(status string) {
		if strings.ToLower(strings.TrimSpace(status)) != "failed" {
			clearPreviewError(repoId, prNumber, copy.Id)
		}
		env, terr := app.Supabase.EnsurePreviewEnvironment(svc.ProjectId, fmt.Sprintf("pr-%d", prNumber), prNumber, previewNamespace, 0)
		if terr != nil {
			slog.Warn("preview: failed to ensure preview env", "service_id", svc.Id, "pr", prNumber, "err", terr)
			return
		}
		if _, terr := app.Supabase.EnsurePreviewEnvironmentService(env.Id, "github_service", copy.Id); terr != nil {
			slog.Warn("preview: failed to track preview environment", "copy_id", copy.Id, "pr", prNumber, "err", terr)
		}
	}

	sanitizedRootDir, err := util.SanitizeRootDir(svc.RootDir)
	if err != nil {
		slog.Error("preview: invalid root_dir, skipping", "service_id", svc.Id, "root_dir", svc.RootDir, "err", err)
		logFn(`[state] preview failed: invalid root_dir: ` + err.Error())
		setStatus("failed")
		track("failed")
		refreshMainWithError(err.Error())
		return
	}

	imageTag := app.Github.RegistryTag(app.Github.RegistryURL(), svc.ResourceName, fmt.Sprintf("pr-%d-%s", prNumber, shortSHA))

	track("building")
	setStatus("building")
	refreshMain()
	logFn(`[state] building preview pr=` + strconv.Itoa(prNumber) + ` sha=` + shortSHA + ` service=` + svc.Name + ` root_dir=` + sanitizedRootDir + ` namespace=` + previewNamespace + ` port=` + strconv.FormatInt(int64(copy.Port), 10))
	slog.Info("preview deploy: building", "copy_id", copy.Id, "pr", prNumber, "sha", shortSHA, "root_dir", sanitizedRootDir, "namespace", previewNamespace, "port", copy.Port)

	token := accessToken
	if token == "" {
		token, _ = app.Github.GetInstallationToken(ctx, strconv.FormatInt(installationId, 10))
	}

	baseEnvStr, envErr := app.Deploy.GetServiceEnv(ctx, deploy.Service{
		Namespace: "proj-" + svc.ProjectId,
		Name:      svc.ResourceName,
	})
	if envErr != nil {
		slog.Error("preview: parent env lookup failed", "service_id", svc.Id, "pr", prNumber, "err", envErr)
		logFn(`[state] preview env lookup failed: ` + envErr.Error())
		setStatus("failed")
		track("failed")
		refreshMainWithError(envErr.Error())
		return
	}
	// Preserve preview edits: overlay live preview env on top of base.
	// GetSecret returns empty map (not error) when missing = first deploy.
	liveEnvStr, _ := app.Deploy.GetServiceEnv(ctx, deploy.Service{
		Namespace: previewNamespace,
		Name:      previewName,
	})
	existingEnv := util.MergePreviewServiceEnv(baseEnvStr, liveEnvStr, previewURL)
	slog.Info("preview deploy: env inherited", "copy_id", copy.Id, "pr", prNumber, "env_keys", len(existingEnv), "port", copy.Port)

	builtImage, err := app.Github.CloneAndBuildPRWithLogs(ctx, headCloneURL, token, sanitizedRootDir, headRef, headSHA, imageTag, logFn, existingEnv)
	if err != nil {
		if github.IsInfraBuildError(err) {
			slog.Error("preview: build failed (infra unavailable)", "copy_id", copy.Id, "pr", prNumber, "err", err)
		} else {
			slog.Error("preview: build failed", "copy_id", copy.Id, "pr", prNumber, "err", err)
		}
		logFn(`[state] preview build failed: ` + err.Error())
		setStatus("failed")
		track("failed")
		refreshMainWithError(err.Error())
		return
	}
	if builtImage == "" {
		builtImage = imageTag
	}

	if err := app.Deploy.CreateProject(ctx, previewNamespace); err != nil {
		slog.Error("preview: namespace setup failed", "copy_id", copy.Id, "pr", prNumber, "namespace", previewNamespace, "err", err)
		logFn(`[state] preview namespace failed: ` + err.Error())
		setStatus("failed")
		track("failed")
		refreshMainWithError(err.Error())
		return
	}

	track("deploying")
	setStatus("deploying")
	refreshMain()
	logFn(`[state] deploying preview image=` + builtImage + ` namespace=` + previewNamespace + ` port=` + strconv.FormatInt(int64(copy.Port), 10))
	if err := app.Deploy.CreateService(ctx, deploy.Service{
		Namespace: previewNamespace,
		Name:      previewName,
		Hostname:  hostname,
		Port:      copy.Port,
		Image:     builtImage,
		Env:       existingEnv,
	}); err != nil {
		slog.Error("preview: deploy failed", "copy_id", copy.Id, "pr", prNumber, "err", err)
		logFn(`[state] preview deploy failed: ` + err.Error())
		setStatus("failed")
		track("failed")
		refreshMainWithError(err.Error())
		return
	}

	track("running")
	setStatus("running")
	refreshMain()
	logFn(`[state] preview running url=` + previewURL + ` image=` + builtImage)
	slog.Info("preview deploy: running", "copy_id", copy.Id, "pr", prNumber, "preview", previewName, "namespace", previewNamespace, "url", previewURL, "image", builtImage)
}

func (app *Application) copyProjectToPreview(projectId string, excludeRepoId int64, prNumber int, previewNamespace string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	ownerId, err := app.Supabase.GetProjectOwnerId(projectId)
	if err != nil {
		slog.Warn("preview copy: owner lookup failed", "project_id", projectId, "pr", prNumber, "err", err)
		return
	}
	if err := app.Deploy.CreateProject(ctx, previewNamespace); err != nil {
		slog.Warn("preview copy: namespace setup failed", "project_id", projectId, "pr", prNumber, "namespace", previewNamespace, "err", err)
		return
	}
	env, err := app.Supabase.EnsurePreviewEnvironment(projectId, fmt.Sprintf("pr-%d", prNumber), prNumber, previewNamespace, 0)
	if err != nil {
		slog.Warn("preview copy: failed to ensure preview env", "project_id", projectId, "pr", prNumber, "err", err)
		return
	}
	// Lists below include copy rows too — skip them so we only copy prod.
	copies, err := app.Supabase.GetPreviewCopyIDs(projectId)
	if err != nil {
		slog.Warn("preview copy: copy lookup failed", "project_id", projectId, "pr", prNumber, "err", err)
		copies = map[string]bool{}
	}
	track := func(sourceType, sourceID string) error {
		_, err := app.Supabase.EnsurePreviewEnvironmentService(env.Id, sourceType, sourceID)
		if err != nil {
			slog.Warn("preview copy: failed to track preview service", "project_id", projectId, "pr", prNumber, "source", sourceID, "err", err)
		}
		return err
	}
	removeCopy := func(sourceType, id string) {
		_ = app.Supabase.DeletePreviewEnvironmentServicesBySource(id)
		if derr := app.Supabase.DeletePreviewCopy(ownerId, sourceType, id); derr != nil {
			slog.Warn("preview copy: failed to remove untracked copy", "project_id", projectId, "pr", prNumber, "source", id, "err", derr)
		}
	}
	copyEnv := func(prodNamespace, prodName, previewNamespace, previewName, hostname string) map[string][]byte {
		baseEnv, err := app.Deploy.GetServiceEnv(ctx, deploy.Service{Namespace: prodNamespace, Name: prodName})
		if err != nil {
			slog.Warn("preview copy: parent env lookup failed, copying without env", "namespace", prodNamespace, "name", prodName, "err", err)
			baseEnv = map[string]string{}
		}
		liveEnv, _ := app.Deploy.GetServiceEnv(ctx, deploy.Service{Namespace: previewNamespace, Name: previewName})
		return util.MergePreviewServiceEnv(baseEnv, liveEnv, "https://"+hostname)
	}

	if services, err := app.Supabase.GetServicesByProjectId(projectId); err != nil {
		slog.Warn("preview copy: services lookup failed", "project_id", projectId, "pr", prNumber, "err", err)
	} else {
		for _, svc := range services {
			if copies[svc.Id] {
				continue
			}
			copy, err := app.Supabase.EnsureServiceCopy(ownerId, svc, prNumber)
			if err != nil {
				slog.Warn("preview copy: service copy failed", "service_id", svc.Id, "pr", prNumber, "err", err)
				continue
			}
			previewName := util.PreviewResourceName(copy.ResourceName, prNumber)
			hostname := util.PreviewHostname(copy.Id, prNumber)
			_, _ = app.Supabase.UpdateServiceStatusById(copy.Id, "deploying")
			if err := app.Deploy.CreateService(ctx, deploy.Service{
				Namespace: previewNamespace,
				Name:      previewName,
				Hostname:  hostname,
				Port:      copy.Port,
				Image:     svc.Image,
				Env:       copyEnv("proj-"+projectId, svc.ResourceName, previewNamespace, previewName, hostname),
			}); err != nil {
				slog.Warn("preview copy: service deploy failed", "copy_id", copy.Id, "pr", prNumber, "err", err)
				_, _ = app.Supabase.UpdateServiceStatusById(copy.Id, "failed")
				continue
			}
			if err := track("service", copy.Id); err != nil {
				removeCopy("service", copy.Id)
				continue
			}
			_, _ = app.Supabase.UpdateServiceStatusById(copy.Id, "running")
		}
	}

	if siblings, err := app.Supabase.GetGithubServicesByProjectId(projectId); err != nil {
		slog.Warn("preview copy: github services lookup failed", "project_id", projectId, "pr", prNumber, "err", err)
	} else {
		for _, g := range siblings {
			if g.RepoId == excludeRepoId || copies[g.Id] {
				continue
			}
			copy, err := app.Supabase.EnsureGithubServiceCopy(ownerId, g, prNumber)
			if err != nil {
				slog.Warn("preview copy: github service copy failed", "service_id", g.Id, "pr", prNumber, "err", err)
				continue
			}
			previewName := util.PreviewResourceName(copy.ResourceName, prNumber)
			hostname := util.PreviewHostname(copy.Id, prNumber)
			_, _ = app.Supabase.UpdateGithubServiceStatusById(copy.Id, "deploying")
			if err := app.Deploy.CreateService(ctx, deploy.Service{
				Namespace: previewNamespace,
				Name:      previewName,
				Hostname:  hostname,
				Port:      copy.Port,
				Image:     app.Github.RegistryTag(app.Github.RegistryURL(), copy.ResourceName, "latest"),
				Env:       copyEnv("proj-"+projectId, g.ResourceName, previewNamespace, previewName, hostname),
			}); err != nil {
				slog.Warn("preview copy: github service deploy failed", "copy_id", copy.Id, "pr", prNumber, "err", err)
				_, _ = app.Supabase.UpdateGithubServiceStatusById(copy.Id, "failed")
				continue
			}
			if err := track("github_service", copy.Id); err != nil {
				removeCopy("github_service", copy.Id)
				continue
			}
			_, _ = app.Supabase.UpdateGithubServiceStatusById(copy.Id, "running")
		}
	}

	if dbs, err := app.Supabase.GetDatabasesByProjectId(projectId); err != nil {
		slog.Warn("preview copy: databases lookup failed", "project_id", projectId, "pr", prNumber, "err", err)
	} else {
		for _, db := range dbs {
			if copies[db.Id] {
				continue
			}
			copy, err := app.Supabase.EnsureDatabaseCopy(ownerId, db, prNumber)
			if err != nil {
				slog.Warn("preview copy: database copy failed", "database_id", db.Id, "pr", prNumber, "err", err)
				continue
			}
			previewName := util.PreviewResourceName(copy.ResourceName, prNumber)
			if err := app.Deploy.CreateDatabase(ctx, deploy.Database{
				Namespace: previewNamespace,
				Name:      previewName,
				Engine:    copy.Engine,
				StorageGB: copy.StorageGB,
			}); err != nil {
				slog.Warn("preview copy: database setup failed", "copy_id", copy.Id, "pr", prNumber, "err", err)
				continue
			}
			if err := track("database", copy.Id); err != nil {
				removeCopy("database", copy.Id)
				continue
			}
		}
	}
}

func (app *Application) cleanupPRPreviews(installationId string, repoId int64, repoFullName string, prNumber int, previews []store.PreviewEnvironmentView) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	namespaces := map[string]bool{}
	for _, v := range previews {
		if strings.TrimSpace(v.Environment.Namespace) == "" {
			continue
		}
		namespaces[v.Environment.Namespace] = true
	}
	if len(namespaces) == 0 {
		namespaces[util.PreviewNamespaceForRepo(repoId, prNumber)] = true
	}
	for namespace := range namespaces {
		if err := app.Deploy.DeleteProject(ctx, namespace); err != nil {
			slog.Warn("preview cleanup: delete failed", "repo_id", repoId, "pr", prNumber, "namespace", namespace, "err", err)
			continue
		}
		slog.Info("preview cleanup: deleted", "repo_id", repoId, "pr", prNumber, "namespace", namespace)
	}
	seenEnvs := map[string]bool{}
	owners := map[string]string{}
	for _, v := range previews {
		if v.Environment.Id == "" || seenEnvs[v.Environment.Id] {
			continue
		}
		seenEnvs[v.Environment.Id] = true
		ownerId, ok := owners[v.Environment.ProjectId]
		if !ok {
			var oerr error
			ownerId, oerr = app.Supabase.GetProjectOwnerId(v.Environment.ProjectId)
			if oerr != nil {
				slog.Warn("preview cleanup: owner lookup failed, keeping copy rows", "env_id", v.Environment.Id, "pr", prNumber, "err", oerr)
				ownerId = ""
			}
			owners[v.Environment.ProjectId] = ownerId
		}
		children, cerr := app.Supabase.GetPreviewEnvironmentServices(v.Environment.Id)
		if cerr == nil {
			for _, c := range children {
				_ = app.Supabase.DeletePreviewEnvironmentServicesBySource(c.SourceServiceId)
				if ownerId != "" {
					if derr := app.Supabase.DeletePreviewCopy(ownerId, c.SourceType, c.SourceServiceId); derr != nil {
						slog.Warn("preview cleanup: copy delete failed", "source", c.SourceServiceId, "pr", prNumber, "err", derr)
					}
				}
			}
		}
		if err := app.Supabase.DeletePreviewEnvironment(v.Environment.Id); err != nil {
			slog.Warn("preview cleanup: env delete failed", "env_id", v.Environment.Id, "pr", prNumber, "err", err)
		}
	}

	if token, err := app.Github.GetInstallationToken(ctx, installationId); err == nil && token != "" && repoFullName != "" {
		app.markPreviewTornDown(ctx, token, repoId, repoFullName, prNumber)
	}
}

func (app *Application) markPreviewTornDown(ctx context.Context, token string, repoId int64, repoFullName string, prNumber int) {
	if token == "" || repoFullName == "" {
		return
	}
	lock := previewCommentLock(repoId, prNumber)
	lock.Lock()
	defer lock.Unlock()
	marker := previewMarker(repoId, prNumber)
	commentID := app.Supabase.GetPreviewCommentIDByRepoPR(repoId, prNumber)
	if commentID == 0 {
		commentID = app.Github.FindPRCommentByMarker(ctx, token, repoFullName, prNumber, marker)
		if commentID == 0 {
			views, _ := app.Supabase.GetPreviewEnvironmentViewsByRepoPR(repoId, prNumber)
			if len(views) == 0 {
				return
			}
		} else {
			app.Supabase.SetPreviewCommentIDByRepoPR(repoId, prNumber, commentID)
		}
	}
	body := renderPreviewTornDownComment(marker, prNumber)
	if commentID == 0 {
		id, err := app.Github.PostPRComment(ctx, token, repoFullName, prNumber, body)
		if err != nil {
			slog.Warn("preview: failed to post torn-down comment", "repo", repoFullName, "pr", prNumber, "err", err)
			return
		}
		app.Supabase.SetPreviewCommentIDByRepoPR(repoId, prNumber, id)
		return
	}
	if err := app.Github.UpdatePRComment(ctx, token, repoFullName, commentID, body); err != nil {
		slog.Warn("preview: failed to update torn-down comment", "repo", repoFullName, "pr", prNumber, "err", err)
	}
}

func (app *Application) fetchPRHead(ctx context.Context, installationToken, repoFullName string, prNumber int) (sha, ref, headRepo string) {
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
				FullName string `json:"full_name"`
			} `json:"repo"`
		} `json:"head"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", "", ""
	}
	return strings.TrimSpace(out.Head.Sha), strings.TrimSpace(out.Head.Ref), strings.TrimSpace(out.Head.Repo.FullName)
}
