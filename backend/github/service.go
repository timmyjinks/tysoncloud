package github

import (
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
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/timmyjinks/tysoncloud/config"
	"github.com/timmyjinks/tysoncloud/util"
)

type Service struct {
	cfg config.Github
}

func NewService(cfg config.Github) *Service {
	return &Service{cfg: cfg}
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
	cloneDir, err := cloneRepo(ctx, cloneURL, accessToken)
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(cloneDir)

	buildCtx, err := resolveBuildContext(cloneDir, rootDir)
	if err != nil {
		return "", err
	}

	image, err := buildImageWithRailpack(ctx, buildCtx, imageTag)
	if err != nil {
		return "", err
	}
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
	out, err := cmd.CombinedOutput()
	if err != nil {
		os.RemoveAll(parent)
		slog.Error("git clone failed", "err", err, "output", string(out))
		return "", fmt.Errorf("git clone failed: %w", err)
	}
	return parent, nil
}

func buildImageWithRailpack(ctx context.Context, buildContext, imageTag string) (string, error) {
	if buildContext == "" {
		return "", fmt.Errorf("build context is empty")
	}
	if imageTag == "" {
		imageTag = "local/app:latest"
	}

	if _, err := exec.LookPath("railpack"); err == nil {
		cmd := exec.CommandContext(ctx, "railpack", "build", "--context", buildContext, "--tag", imageTag)
		out, err := cmd.CombinedOutput()
		if err != nil {
			slog.Error("railpack build failed, falling back to docker build", "err", err, "output", string(out))
		} else {
			return imageTag, nil
		}
	}

	cmd := exec.CommandContext(ctx, "docker", "build", "-t", imageTag, buildContext)
	out, err := cmd.CombinedOutput()
	if err != nil {
		slog.Error("docker build failed", "err", err, "output", string(out))
		return "", fmt.Errorf("image build failed: %w", err)
	}
	return imageTag, nil
}
