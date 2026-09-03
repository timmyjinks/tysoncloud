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

func (s *Service) CloneAndBuild(ctx context.Context, cloneURL, accessToken, rootDir, imageTag string) (string, error) {
	return s.CloneAndBuildWithLogs(ctx, cloneURL, accessToken, rootDir, imageTag, nil)
}

func (s *Service) CloneAndBuildWithLogs(ctx context.Context, cloneURL, accessToken, rootDir, imageTag string, logFn func(string)) (string, error) {
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

	emit(fmt.Sprintf("[state] building: cloning %s root_dir=%s", cloneURL, rootDir))
	cloneDir, err := cloneRepoWithLogs(ctx, cloneURL, accessToken, logFn)
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
	image, err := buildImageWithRailpackWithLogs(ctx, buildCtx, imageTag, logFn)
	if err != nil {
		return "", err
	}
	emit(fmt.Sprintf("[state] building: image built %s", image))
	return image, nil
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

func cloneRepo(ctx context.Context, cloneURL, accessToken string) (string, error) {
	return cloneRepoWithLogs(ctx, cloneURL, accessToken, nil)
}

func cloneRepoWithLogs(ctx context.Context, cloneURL, accessToken string, logFn func(string)) (string, error) {
	parent, err := os.MkdirTemp("", "gh-clone-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	os.RemoveAll(parent)

	url := cloneURL
	if accessToken != "" && strings.HasPrefix(cloneURL, "https://") {
		url = strings.Replace(cloneURL, "https://", "https://x-access-token:"+accessToken+"@", 1)
	}

	cmd := exec.CommandContext(ctx, "git", "clone", "--depth", "1", url, parent)
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
	return buildImageWithRailpackWithLogs(ctx, buildContext, imageTag, nil)
}

func buildImageWithRailpackWithLogs(ctx context.Context, buildContext, imageTag string, logFn func(string)) (string, error) {
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

	prepCmd := exec.CommandContext(ctx, "railpack", "prepare", buildContext, "--plan-out", planPath)
	prepCmd.Env = os.Environ()
	if out, err := runCmdWithLogs(prepCmd, logFn); err != nil {
		slog.Error("railpack prepare failed", "err", err, "output", out)
		return "", fmt.Errorf("railpack prepare failed: %w", err)
	}

	buildCmd := exec.CommandContext(ctx, "buildctl", "build",
		"--local", "context="+buildContext,
		"--local", "dockerfile="+planDir,
		"--frontend=gateway.v0",
		"--opt", "source=ghcr.io/railwayapp/railpack-frontend:latest",
		"--output", fmt.Sprintf("type=image,name=%s,push=true,registry.insecure=true", imageTag),
	)
	buildCmd.Env = os.Environ()
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
