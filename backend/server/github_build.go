package server

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/timmyjinks/tysoncloud/deploy"
	"github.com/timmyjinks/tysoncloud/util"
)

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

// cloneRepo clones the repository at cloneURL into a temporary directory.
// If accessToken is non-empty, it is injected as https://x-access-token:<token>@host/path for private repos.
// Returns the temp dir path; caller must defer os.RemoveAll.
func cloneRepo(ctx context.Context, cloneURL, accessToken string) (string, error) {
	parent, err := os.MkdirTemp("", "gh-clone-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}
	// Remove parent so git clone can create it; we then clone into parent.
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

// buildImageWithRailpack builds a container image from buildContext using railpack.
// Local registry only: tags image as local/<serviceName>:<tag> via docker/railpack.
// For now this shells out to `railpack build` and `docker build`.
// If railpack is not available, falls back to plain `docker build`.
func buildImageWithRailpack(ctx context.Context, buildContext, imageTag string) (string, error) {
	if buildContext == "" {
		return "", fmt.Errorf("build context is empty")
	}
	if imageTag == "" {
		imageTag = "local/app:latest"
	}

	// Prefer: railpack build --context <dir> --tag <imageTag>
	// Fallback: docker build
	// We try railpack first; if binary missing, use docker.
	if _, err := exec.LookPath("railpack"); err == nil {
		cmd := exec.CommandContext(ctx, "railpack", "build", "--context", buildContext, "--tag", imageTag)
		out, err := cmd.CombinedOutput()
		if err != nil {
			slog.Error("railpack build failed, falling back to docker build", "err", err, "output", string(out))
		} else {
			return imageTag, nil
		}
	}

	// Fallback to docker build using Dockerfile generated or existing
	cmd := exec.CommandContext(ctx, "docker", "build", "-t", imageTag, buildContext)
	out, err := cmd.CombinedOutput()
	if err != nil {
		slog.Error("docker build failed", "err", err, "output", string(out))
		return "", fmt.Errorf("image build failed: %w", err)
	}
	return imageTag, nil
}

// cloneAndBuild clones the repo, resolves rootDir, builds image, and cleans up clone dir.
// Returns the built image tag.
func cloneAndBuild(ctx context.Context, cloneURL, accessToken, rootDir, imageTag string) (string, error) {
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

// deployGithubService wraps deploy.CreateService for github services.
func (app *Application) deployGithubService(ctx context.Context, svc deploy.Service) error {
	return app.Deploy.CreateService(ctx, svc)
}
