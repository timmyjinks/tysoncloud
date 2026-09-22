package github

import (
	"bufio"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/timmyjinks/tysoncloud/config"
	"github.com/timmyjinks/tysoncloud/util"
)

type Service struct {
	cfg  config.Github
	reg  config.Registry
	logs *BuildLogStore
}

func NewService(cfg config.Github, reg config.Registry) *Service {
	return &Service{cfg: cfg, reg: reg, logs: NewBuildLogStore()}
}

func (s *Service) AppendBuildLog(serviceID, line string) {
	if s.logs == nil {
		s.logs = NewBuildLogStore()
	}
	s.logs.Append(serviceID, line)
}

func (s *Service) SubscribeBuildLogs(serviceID string) (int, chan string, []string) {
	if s.logs == nil {
		s.logs = NewBuildLogStore()
	}
	return s.logs.Subscribe(serviceID)
}

func (s *Service) UnsubscribeBuildLogs(serviceID string, subID int) {
	if s.logs == nil {
		return
	}
	s.logs.Unsubscribe(serviceID, subID)
}

func (s *Service) RegistryURL() string {
	return s.reg.URL
}

func RegistryTag(registryURL, resourceName, tag string) string {
	if tag == "" {
		tag = "latest"
	}
	registryURL = strings.TrimSuffix(strings.TrimSpace(registryURL), "/")
	resourceName = strings.TrimSpace(resourceName)
	if registryURL == "" {
		return fmt.Sprintf("local/%s:%s", resourceName, tag)
	}
	return fmt.Sprintf("%s/%s:%s", registryURL, resourceName, tag)
}

func (s *Service) RegistryTag(registryURL, resourceName, tag string) string {
	return RegistryTag(registryURL, resourceName, tag)
}

func IsInfraBuildError(err error) bool {
	return isInfraBuildError(err)
}

func isInfraBuildError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	infraMarkers := []string{
		"buildkit_host", "buildkit host", "buildkit", "buildctl",
		"cannot connect to the docker daemon", "docker daemon", "is the docker daemon running",
		"no buildkit builder available", "registry.insecure", "connection refused", "no such host",
	}
	for _, m := range infraMarkers {
		if strings.Contains(msg, m) {
			return true
		}
	}
	return false
}

func (s *Service) GetInstallationToken(ctx context.Context, installationId string) (string, error) {
	if installationId == "" {
		return "", fmt.Errorf("installation_id is required")
	}
	if s.cfg.InstallationToken != "" {
		return s.cfg.InstallationToken, nil
	}
	if s.cfg.AppPrivateKey == "" || s.cfg.AppID == "" {
		return "", fmt.Errorf("github app not configured: set GITHUB_APP_ID and GITHUB_APP_PRIVATE_KEY")
	}
	jwtToken, err := s.generateAppJWT()
	if err != nil {
		return "", fmt.Errorf("generate app JWT: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("https://api.github.com/app/installations/%s/access_tokens", installationId), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+jwtToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("github token exchange failed %d: %s", resp.StatusCode, string(body))
	}
	var out struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", err
	}
	if out.Token == "" {
		return "", fmt.Errorf("empty token from github")
	}
	return out.Token, nil
}

func (s *Service) generateAppJWT() (string, error) {
	keyData := s.cfg.AppPrivateKey
	keyData = strings.ReplaceAll(keyData, "\\n", "\n")
	privKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(keyData))
	if err != nil {
		return "", err
	}
	now := time.Now()
	claims := jwt.MapClaims{
		"iat": now.Add(-60 * time.Second).Unix(),
		"exp": now.Add(9 * time.Minute).Unix(),
		"iss": s.cfg.AppID,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(privKey)
}

func (s *Service) VerifyWebhookSignature(signature string, body []byte) bool {
	if s.cfg.WebhookSecret == "" {
		return true
	}
	if signature == "" {
		return false
	}
	hexSig := strings.TrimPrefix(signature, "sha256=")
	expectedMAC := hmac.New(sha256.New, []byte(s.cfg.WebhookSecret))
	expectedMAC.Write(body)
	expected := hex.EncodeToString(expectedMAC.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(hexSig))
}

func (s *Service) CloneAndBuild(ctx context.Context, cloneURL, accessToken, rootDir, branch, imageTag string) (string, error) {
	return s.CloneAndBuildWithLogs(ctx, cloneURL, accessToken, rootDir, branch, imageTag, nil, nil)
}

// envKeyRegex gates which keys are forwarded to `railpack prepare --env`.
// Anything outside [A-Za-z_][A-Za-z0-9_]* is skipped (never logged either).
var envKeyRegex = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// railpackEnvArgs converts an env map into sorted `--env KEY=VALUE` arg pairs
// for `railpack prepare`. Sorted for reproducible plans/logs. Keys only ever
// appear in logs via the caller's env_keys line — never values.
func railpackEnvArgs(env map[string][]byte) []string {
	if len(env) == 0 {
		return nil
	}
	keys := make([]string, 0, len(env))
	for k := range env {
		if envKeyRegex.MatchString(k) {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	args := make([]string, 0, len(keys)*2)
	for _, k := range keys {
		args = append(args, "--env", k+"="+string(env[k]))
	}
	return args
}

// railpackEnvKeys returns the sorted, valid key names (for keys-only logging).
func railpackEnvKeys(env map[string][]byte) []string {
	if len(env) == 0 {
		return nil
	}
	keys := make([]string, 0, len(env))
	for k := range env {
		if envKeyRegex.MatchString(k) {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return keys
}

// buildctlSecretArgs converts an env map into `--secret id=KEY,env=KEY` arg
// pairs for `buildctl build` (railpack-frontend flow). Same sorted valid keys
// as railpackEnvArgs. The values are read from buildctl's own process env
// (env=KEY), so callers must export them on the command's Env.
func buildctlSecretArgs(env map[string][]byte) []string {
	keys := railpackEnvKeys(env)
	if len(keys) == 0 {
		return nil
	}
	args := make([]string, 0, len(keys)*2)
	for _, k := range keys {
		args = append(args, "--secret", "id="+k+",env="+k)
	}
	return args
}

// secretsHash mirrors upstream getSecretsHash: hex sha256 over sorted
// "KEY=VALUE\n" lines. Passed as build-arg:secrets-hash so changed secret
// values bust the layer cache (the frontend does NOT do this automatically).
func secretsHash(env map[string][]byte) string {
	keys := railpackEnvKeys(env)
	if len(keys) == 0 {
		return ""
	}
	h := sha256.New()
	for _, k := range keys {
		h.Write([]byte(k + "=" + string(env[k]) + "\n"))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// exportEnv returns base with every valid key of env appended as KEY=VALUE,
// replacing any existing entries for those keys (append wins in exec env).
func exportEnv(base []string, env map[string][]byte) []string {
	keys := railpackEnvKeys(env)
	if len(keys) == 0 {
		return base
	}
	replace := make(map[string]bool, len(keys))
	for _, k := range keys {
		replace[k+"="] = true
	}
	out := make([]string, 0, len(base)+len(keys))
	for _, kv := range base {
		if i := strings.IndexByte(kv, '='); i > 0 {
			if replace[kv[:i+1]] {
				continue
			}
		}
		out = append(out, kv)
	}
	for _, k := range keys {
		out = append(out, k+"="+string(env[k]))
	}
	return out
}

func (s *Service) CloneAndBuildWithLogs(ctx context.Context, cloneURL, accessToken, rootDir, branch, imageTag string, logFn func(string), env map[string][]byte) (string, error) {
	emit := func(msg string) {
		slog.Info(msg)
		if logFn != nil {
			logFn(msg)
		}
	}
	emitErr := func(msg string, err error, output string) {
		slog.Error(msg, "err", err, "output", output)
		if logFn != nil {
			if output != "" {
				for _, line := range strings.Split(strings.TrimRight(output, "\n"), "\n") {
					logFn(line)
				}
			}
			logFn(fmt.Sprintf("%s: %v", msg, err))
		}
	}

	emit(fmt.Sprintf("[state] building: cloning %s root_dir=%s branch=%s", cloneURL, rootDir, branch))
	cloneDir, err := cloneRepoWithLogs(ctx, cloneURL, accessToken, branch, logFn)
	if err != nil {
		emitErr("git clone failed", err, "")
		return "", err
	}
	defer os.RemoveAll(cloneDir)
	emit("[state] building: cloning succeeded")

	buildCtx, err := resolveBuildContext(cloneDir, rootDir)
	if err != nil {
		emitErr("resolve build context failed", err, "")
		return "", err
	}

	emit(fmt.Sprintf("[state] building: railpack prepare context=%s", buildCtx))
	image, err := buildImageWithRailpackWithLogs(ctx, buildCtx, imageTag, logFn, env)
	if err != nil {
		return "", err
	}
	emit(fmt.Sprintf("[state] building: image built %s", image))
	return image, nil
}

func (s *Service) CloneAndBuildPRWithLogs(ctx context.Context, cloneURL, accessToken, rootDir, headRef, headSHA, imageTag string, logFn func(string), env map[string][]byte) (string, error) {
	emit := func(msg string) {
		slog.Info(msg)
		if logFn != nil {
			logFn(msg)
		}
	}
	emitErr := func(msg string, err error, output string) {
		slog.Error(msg, "err", err, "output", output)
		if logFn != nil {
			if output != "" {
				for _, line := range strings.Split(strings.TrimRight(output, "\n"), "\n") {
					logFn(line)
				}
			}
			logFn(fmt.Sprintf("%s: %v", msg, err))
		}
	}

	// NOTE: cloneURL must be the PR *head* repo URL, not the base repo URL.
	// For same-repo PRs they are identical; for fork PRs the head URL points
	// at the fork. This is what makes fork previews work without extra code.
	// QUESTION: private forks may 404 with the installation token (the token
	// belongs to the base repo installation). If that becomes a problem we
	// should skip forks or fall back to checking out the merge ref instead.
	emit(fmt.Sprintf("[state] building: cloning PR head %s ref=%s sha=%s root_dir=%s", cloneURL, headRef, headSHA, rootDir))
	cloneDir, err := cloneRepoWithLogs(ctx, cloneURL, accessToken, headRef, logFn)
	if err != nil {
		emitErr("git clone failed", err, "")
		return "", err
	}
	defer os.RemoveAll(cloneDir)
	emit("[state] building: cloning succeeded")

	// Pin to the exact PR head SHA so a `synchronize` race (new push between
	// webhook delivery and clone) still builds what GitHub reported.
	if strings.TrimSpace(headSHA) != "" {
		fetch := exec.CommandContext(ctx, "git", "-C", cloneDir, "fetch", "origin", strings.TrimSpace(headSHA), "--depth", "1")
		fetch.Env = os.Environ()
		if out, ferr := runCmdWithLogs(fetch, logFn); ferr != nil {
			// Shallow single-branch clones often already contain the tip;
			// a failed fetch is non-fatal — try a direct checkout next.
			slog.Warn("git fetch sha failed, trying checkout", "sha", headSHA, "err", ferr, "output", out)
		}
		checkedOut := false
		for _, args := range [][]string{
			{"-C", cloneDir, "checkout", strings.TrimSpace(headSHA)},
			{"-C", cloneDir, "checkout", "FETCH_HEAD"},
		} {
			cmd := exec.CommandContext(ctx, "git", args...)
			cmd.Env = os.Environ()
			if _, cerr := runCmdWithLogs(cmd, logFn); cerr == nil {
				checkedOut = true
				break
			}
		}
		if !checkedOut {
			// Neither checkout worked — fall back to whatever the clone
			// gave us (the branch tip) rather than failing the preview.
			slog.Warn("git checkout sha failed, using branch tip", "sha", headSHA)
		}
	}

	buildCtx, err := resolveBuildContext(cloneDir, rootDir)
	if err != nil {
		emitErr("resolve build context failed", err, "")
		return "", err
	}

	emit(fmt.Sprintf("[state] building: railpack prepare context=%s", buildCtx))
	image, err := buildImageWithRailpackWithLogs(ctx, buildCtx, imageTag, logFn, env)
	if err != nil {
		return "", err
	}
	emit(fmt.Sprintf("[state] building: image built %s", image))
	return image, nil
}

// PostPRComment posts a comment on the PR via the issues API, which shares
// numbering with pull requests. Returns the created comment id so callers can
// PATCH it later for live updates. Best-effort: callers should log but not
// fail the deploy on comment errors.
func (s *Service) PostPRComment(ctx context.Context, installationToken, repoFullName string, prNumber int, body string) (int64, error) {
	if installationToken == "" || strings.TrimSpace(repoFullName) == "" || prNumber <= 0 {
		return 0, fmt.Errorf("installation token, repo and PR number are required")
	}
	url := fmt.Sprintf("https://api.github.com/repos/%s/issues/%d/comments", strings.TrimSpace(repoFullName), prNumber)
	payload, _ := json.Marshal(map[string]string{"body": body})
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+installationToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusCreated {
		return 0, fmt.Errorf("github comment failed %d: %s", resp.StatusCode, string(respBody))
	}
	var out struct {
		Id int64 `json:"id"`
	}
	_ = json.Unmarshal(respBody, &out)
	return out.Id, nil
}

// UpdatePRComment edits an existing issue/PR comment in place. Used for the
// single live preview comment per PR. A 404 means the comment was deleted —
// callers should recreate via PostPRComment.
func (s *Service) UpdatePRComment(ctx context.Context, installationToken, repoFullName string, commentID int64, body string) error {
	if installationToken == "" || strings.TrimSpace(repoFullName) == "" || commentID <= 0 {
		return fmt.Errorf("installation token, repo and comment id are required")
	}
	url := fmt.Sprintf("https://api.github.com/repos/%s/issues/comments/%d", strings.TrimSpace(repoFullName), commentID)
	payload, _ := json.Marshal(map[string]string{"body": body})
	req, err := http.NewRequestWithContext(ctx, "PATCH", url, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+installationToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("github comment update failed %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// FindPRCommentByMarker lists PR comments and returns the id of the first one
// containing marker (our hidden HTML marker). 0 = not found. Used to adopt an
// existing live comment when the DB row is missing.
func (s *Service) FindPRCommentByMarker(ctx context.Context, installationToken, repoFullName string, prNumber int, marker string) int64 {
	if installationToken == "" || strings.TrimSpace(repoFullName) == "" || prNumber <= 0 || marker == "" {
		return 0
	}
	url := fmt.Sprintf("https://api.github.com/repos/%s/issues/%d/comments?per_page=100", strings.TrimSpace(repoFullName), prNumber)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0
	}
	req.Header.Set("Authorization", "Bearer "+installationToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return 0
	}
	var comments []struct {
		Id   int64  `json:"id"`
		Body string `json:"body"`
	}
	if err := json.Unmarshal(body, &comments); err != nil {
		return 0
	}
	for _, c := range comments {
		if strings.Contains(c.Body, marker) {
			return c.Id
		}
	}
	return 0
}

func resolveBuildContext(cloneDir, rootDir string) (string, error) {
	sanitized, err := util.SanitizeRootDir(rootDir)
	if err != nil {
		return "", err
	}
	if sanitized == "." {
		return cloneDir, nil
	}
	ctx := filepath.Join(cloneDir, sanitized)
	fi, err := os.Stat(ctx)
	if err != nil {
		return "", fmt.Errorf("root_dir %q does not exist in repository: %w", rootDir, err)
	}
	if !fi.IsDir() {
		return "", fmt.Errorf("root_dir %q is not a directory", rootDir)
	}
	return ctx, nil
}

func cloneRepo(ctx context.Context, cloneURL, accessToken, branch string) (string, error) {
	return cloneRepoWithLogs(ctx, cloneURL, accessToken, branch, nil)
}

func cloneRepoWithLogs(ctx context.Context, cloneURL, accessToken, branch string, logFn func(string)) (string, error) {
	parent, err := os.MkdirTemp("", "gh-clone-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	os.RemoveAll(parent)

	url := cloneURL
	if accessToken != "" && strings.HasPrefix(cloneURL, "https://") {
		url = strings.Replace(cloneURL, "https://", "https://x-access-token:"+accessToken+"@", 1)
	}

	args := []string{"clone", "--depth", "1"}
	if strings.TrimSpace(branch) != "" {
		args = append(args, "--branch", strings.TrimSpace(branch))
	}
	args = append(args, url, parent)
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = os.Environ()
	out, err := runCmdWithLogs(cmd, logFn)
	if err != nil {
		os.RemoveAll(parent)
		slog.Error("git clone failed", "err", err, "output", out)
		return "", fmt.Errorf("git clone failed: %w", err)
	}
	return parent, nil
}

func buildImageWithRailpack(ctx context.Context, buildContext, imageTag string) (string, error) {
	return buildImageWithRailpackWithLogs(ctx, buildContext, imageTag, nil, nil)
}

func buildImageWithRailpackWithLogs(ctx context.Context, buildContext, imageTag string, logFn func(string), env map[string][]byte) (string, error) {
	if buildContext == "" {
		return "", fmt.Errorf("build context is empty")
	}
	if imageTag == "" {
		imageTag = "local/app:latest"
	}

	planDir, err := os.MkdirTemp("", "railpack-plan-*")
	if err != nil {
		return "", fmt.Errorf("create plan dir: %w", err)
	}
	defer os.RemoveAll(planDir)
	planPath := filepath.Join(planDir, "railpack-plan.json")

	// Bake app envs into the image plan (e.g. Vite API_URL). Keys only in
	// logs — never values.
	envArgs := railpackEnvArgs(env)
	if keys := railpackEnvKeys(env); len(keys) > 0 {
		msg := fmt.Sprintf("[state] building: railpack env_keys=%d [%s]", len(keys), strings.Join(keys, ","))
		slog.Info(msg)
		if logFn != nil {
			logFn(msg)
		}
	}
	prepArgs := append([]string{"prepare", buildContext, "--plan-out", planPath}, envArgs...)
	prepCmd := exec.CommandContext(ctx, "railpack", prepArgs...)
	prepCmd.Env = os.Environ()
	if out, err := runCmdWithLogs(prepCmd, logFn); err != nil {
		slog.Error("railpack prepare failed", "err", err, "output", out)
		return "", fmt.Errorf("railpack prepare failed: %w", err)
	}

	buildArgs := []string{
		"build",
		"--local", "context=" + buildContext,
		"--local", "dockerfile=" + planDir,
		"--frontend=gateway.v0",
		"--opt", "source=ghcr.io/railwayapp/railpack-frontend:latest",
	}
	// Secrets reach the frontend via BuildKit session secrets: values come
	// from buildctl's process env (env=KEY), ids only in argv. Keys-only in
	// logs — never values.
	if secretArgs := buildctlSecretArgs(env); len(secretArgs) > 0 {
		buildArgs = append(buildArgs, secretArgs...)
	}
	if hash := secretsHash(env); hash != "" {
		buildArgs = append(buildArgs, "--opt", "build-arg:secrets-hash="+hash)
	}
	buildArgs = append(buildArgs,
		"--output", fmt.Sprintf("type=image,name=%s,push=true,registry.insecure=true", imageTag),
	)
	buildCmd := exec.CommandContext(ctx, "buildctl", buildArgs...)
	buildCmd.Env = exportEnv(os.Environ(), env)
	if keys := railpackEnvKeys(env); len(keys) > 0 {
		msg := fmt.Sprintf("[state] building: buildctl secret_ids=%d [%s]", len(keys), strings.Join(keys, ","))
		slog.Info(msg)
		if logFn != nil {
			logFn(msg)
		}
	}
	if out, err := runCmdWithLogs(buildCmd, logFn); err != nil {
		slog.Error("buildctl build failed", "err", err, "output", out)
		return "", fmt.Errorf("buildctl build failed: %w", err)
	}
	return imageTag, nil
}

func runCmdWithLogs(cmd *exec.Cmd, logFn func(string)) (string, error) {
	var buf bytes.Buffer
	var mu sync.Mutex
	if logFn != nil {
		cmd.Stdout = &logWriter{fn: logFn, buf: &buf, mu: &mu}
		cmd.Stderr = &logWriter{fn: logFn, buf: &buf, mu: &mu}
	} else {
		cmd.Stdout = &buf
		cmd.Stderr = &buf
	}
	err := cmd.Run()
	return buf.String(), err
}

type logWriter struct {
	fn  func(string)
	buf *bytes.Buffer
	mu  *sync.Mutex
}

func (w *logWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	n, _ := w.buf.Write(p)
	s := string(p)
	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" && w.fn != nil {
			w.fn(line)
		}
	}
	if len(s) > 0 && !strings.HasSuffix(s, "\n") && !strings.Contains(s, "\n") {
		if w.fn != nil {
			w.fn(strings.TrimRight(s, "\r\n"))
		}
	}
	return n, nil
}

const maxBuildLogLines = 2000

type buildLogEntry struct {
	lines []string
	chans map[int]chan string
	next  int
	mu    sync.Mutex
}

type BuildLogStore struct {
	mu   sync.Mutex
	data map[string]*buildLogEntry
}

func NewBuildLogStore() *BuildLogStore {
	return &BuildLogStore{data: make(map[string]*buildLogEntry)}
}

func (s *BuildLogStore) ensure(id string) *buildLogEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.data[id]
	if !ok {
		e = &buildLogEntry{chans: make(map[int]chan string)}
		s.data[id] = e
	}
	return e
}

func (s *BuildLogStore) Append(serviceID, line string) {
	if line == "" {
		return
	}
	e := s.ensure(serviceID)
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.lines) >= maxBuildLogLines {
		e.lines = e.lines[1:]
	}
	e.lines = append(e.lines, line)
	for _, ch := range e.chans {
		select {
		case ch <- line:
		default:
		}
	}
}

func (s *BuildLogStore) Subscribe(serviceID string) (int, chan string, []string) {
	e := s.ensure(serviceID)
	e.mu.Lock()
	defer e.mu.Unlock()
	ch := make(chan string, 64)
	id := e.next
	e.next++
	e.chans[id] = ch
	snap := make([]string, len(e.lines))
	copy(snap, e.lines)
	return id, ch, snap
}

func (s *BuildLogStore) Unsubscribe(serviceID string, subID int) {
	s.mu.Lock()
	e, ok := s.data[serviceID]
	s.mu.Unlock()
	if !ok {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if ch, ok := e.chans[subID]; ok {
		close(ch)
		delete(e.chans, subID)
	}
}
