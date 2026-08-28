package server

import (
	"context"
	"fmt"
	"os"
)

// getInstallationToken exchanges a GitHub App installation ID for an installation access token.
// For local dev where GITHUB_APP_PRIVATE_KEY is not set, it returns empty token (public repos only).
// In production, implement JWT creation + POST https://api.github.com/app/installations/{id}/access_tokens
func (app *Application) getInstallationToken(ctx context.Context, installationId string) (string, error) {
	if installationId == "" {
		return "", fmt.Errorf("installation_id is required")
	}
	// If env var GITHUB_INSTALLATION_TOKEN is set (for local testing), use it.
	if tok := os.Getenv("GITHUB_INSTALLATION_TOKEN"); tok != "" {
		return tok, nil
	}
	// If GitHub App not configured, return empty - allows cloning public repos without auth.
	if os.Getenv("GITHUB_APP_PRIVATE_KEY") == "" {
		return "", nil
	}
	// TODO: implement JWT + token exchange. For now return error to surface misconfig.
	return "", fmt.Errorf("GitHub App token exchange not implemented - set GITHUB_INSTALLATION_TOKEN for local dev")
}
