package util

import (
	"fmt"
	"regexp"
	"strings"
)

var githubRepoFullNameRegex = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,38})/[A-Za-z0-9._-]{1,100}$`)

// GithubCloneURL builds the HTTPS clone URL for an "owner/repo" full name.
// Clone URLs are never taken from webhook payloads; they are always derived
// from a validated full name so builds can only ever target github.com.
func GithubCloneURL(fullName string) (string, error) {
	if !githubRepoFullNameRegex.MatchString(fullName) || fullName[len(fullName)-1] == '.' {
		return "", fmt.Errorf("invalid github repository name %q", fullName)
	}
	return "https://github.com/" + fullName + ".git", nil
}

// IsForkPR reports whether a PR's head lives outside the base repository.
// Fork PRs carry code from anyone on GitHub, so they must never be built
// with, or deployed next to, the base repo's secrets. An unknown base repo
// counts as a fork so callers fail closed.
func IsForkPR(headRepo, baseRepo string) bool {
	return baseRepo == "" || !strings.EqualFold(strings.TrimSpace(headRepo), strings.TrimSpace(baseRepo))
}
