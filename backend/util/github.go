package util

import (
	"fmt"
	"regexp"
	"strings"
)

var githubRepoFullNameRegex = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,38})/[A-Za-z0-9._-]{1,100}$`)

func GithubCloneURL(fullName string) (string, error) {
	if !githubRepoFullNameRegex.MatchString(fullName) || fullName[len(fullName)-1] == '.' {
		return "", fmt.Errorf("invalid github repository name %q", fullName)
	}
	return "https://github.com/" + fullName + ".git", nil
}

func IsForkPR(headRepo, baseRepo string) bool {
	headRepo, baseRepo = strings.TrimSpace(headRepo), strings.TrimSpace(baseRepo)
	return headRepo == "" || baseRepo == "" || !strings.EqualFold(headRepo, baseRepo)
}
