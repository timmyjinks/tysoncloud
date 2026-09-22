package util

import (
	"fmt"
	"regexp"
	"strings"
)

var nameRegex = regexp.MustCompile(`^[A-Za-z-]+$`)
var envRegex = regexp.MustCompile(`\A(?:[A-Za-z_][A-Za-z0-9_]*=[^\n]*)*(?:\n[A-Za-z_][A-Za-z0-9_]*=[^\n]*)*\z`)
var domainLabelRegex = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)
var branchRegex = regexp.MustCompile(`^[A-Za-z0-9._-]+(?:/[A-Za-z0-9._-]+)*$`)

func validateName(name string) (bool, error) {
	if len(name) > 24 {
		return false, nil
	}

	return nameRegex.MatchString(name), nil
}

func ValidateDomainLabel(label string) (bool, error) {
	if len(label) == 0 || len(label) > 63 {
		return false, nil
	}
	return domainLabelRegex.MatchString(label), nil
}

// Preview helpers.
//
// Preview namespaces are shared per repo+PR: every github_service for the
// same repo deploys its PR head into ONE namespace
// (`PreviewNamespaceForRepo(repoId, prNumber)`) so monorepo services share
// networking. Names/hostnames stay per-service
// (`PreviewResourceName` / `PreviewHostname`) so `synchronize` rebuilds
// overwrite the same K8s objects (Apply is idempotent) and `closed` can
// clean up. Tracking rows live in the `preview_environments` table (see
// store/preview_environments.go) — K8s is the runtime, the DB is the index
// (used for cascade deletes and status/history). No env values are ever
// stored in the DB.

// ShortServiceID returns the first 8 hex chars of a service UUID with dashes
// removed, e.g. "a1b2c3d4". Used as the `<uuid>` in
// `tc-preview-<uuid>-pr-<n>.tysonjenkins.dev`. 8 chars keeps the first DNS
// label well under 63 chars while remaining unique per service.
func ShortServiceID(id string) string {
	v := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(id), "-", ""))
	// ResourceNames look like `svc-<uuid>`; strip a leading `svc` so the
	// hostname doesn't stutter (`tc-preview-svc...`).
	v = strings.TrimPrefix(v, "svc")
	v = strings.Trim(v, "-")
	if v == "" {
		return "svc"
	}
	// Keep only DNS-safe chars.
	var b strings.Builder
	for _, r := range v {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	s := b.String()
	if s == "" {
		return "svc"
	}
	if len(s) > 8 {
		s = s[:8]
	}
	return s
}

// PreviewResourceName derives the K8s object name for a PR preview from the
// parent service's ResourceName, e.g. `svc-a1b2c3d4-pr-42`. Truncated to 63
// chars (DNS-1123) with the `-pr-<n>` suffix preserved so deletes can always
// recompute the same name.
func PreviewResourceName(resourceName string, prNumber int) string {
	base := strings.ToLower(strings.TrimSpace(resourceName))
	// Sanitize anything outside [a-z0-9-] to `-`.
	base = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return '-'
	}, base)
	base = strings.Trim(base, "-")
	if base == "" {
		base = "svc"
	}
	suffix := fmt.Sprintf("-pr-%d", prNumber)
	// Leave room for the suffix within the 63-char limit.
	if len(base)+len(suffix) > 63 {
		base = strings.Trim(base[:63-len(suffix)], "-")
	}
	return base + suffix
}

// PreviewNamespaceForRepo derives the SHARED K8s namespace for a PR preview
// from the repo ID, e.g. `prev-r123456789-pr-42`. All services for the same
// repo deploy into this one namespace (provisioned via Deploy.CreateProject,
// torn down via Deploy.DeleteProject). Deterministic so `synchronize`
// rebuilds reuse it and `closed` can recompute it. Truncated to 63 chars
// (DNS-1123).
func PreviewNamespaceForRepo(repoID int64, prNumber int) string {
	ns := fmt.Sprintf("prev-r%d-pr-%d", repoID, prNumber)
	if len(ns) > 63 {
		ns = strings.Trim(ns[:63], "-")
	}
	return ns
}

// PreviewNamespace derives the isolated K8s namespace for a PR preview from
// the parent service ID, e.g. `prev-a1b2c3d4-pr-42`.
//
// Deprecated: previews now share one namespace per repo+PR — use
// PreviewNamespaceForRepo. Kept for reading pre-migration tracking rows.
func PreviewNamespace(serviceID string, prNumber int) string {
	base := fmt.Sprintf("prev-%s", ShortServiceID(serviceID))
	suffix := fmt.Sprintf("-pr-%d", prNumber)
	if len(base)+len(suffix) > 63 {
		base = strings.Trim(base[:63-len(suffix)], "-")
	}
	return base + suffix
}

// PreviewHostname returns the public URL host for a preview:
// `tc-preview-<shortSvcId>-pr-<n>.tysonjenkins.dev`.
func PreviewHostname(serviceID string, prNumber int) string {
	return fmt.Sprintf("tc-preview-%s-pr-%d.tysonjenkins.dev", ShortServiceID(serviceID), prNumber)
}

// ShortSHA truncates a commit SHA for image tags and log lines.
func ShortSHA(sha string) string {
	s := strings.TrimSpace(sha)
	if s == "" {
		return "latest"
	}
	if len(s) > 12 {
		s = s[:12]
	}
	return strings.ToLower(s)
}

func NormalizeDomain(raw string) string {
	v := strings.TrimSpace(strings.ToLower(raw))
	if v == "" {
		return ""
	}
	if strings.HasPrefix(v, "tc-") {
		v = strings.TrimPrefix(v, "tc-")
	}
	if idx := strings.Index(v, "."); idx != -1 {
		v = v[:idx]
	}
	return strings.TrimSpace(v)
}

func ValidateEnv(env string) (bool, error) {
	if env == "" {
		return true, nil
	}

	return envRegex.MatchString(env), nil
}

func ParseEnv(env string) map[string][]byte {
	result := map[string][]byte{}

	for _, line := range strings.Split(env, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		result[parts[0]] = []byte(parts[1])
	}

	return result
}

func SanitizeBranch(raw string) (string, error) {
	v := strings.TrimSpace(raw)
	if v == "" {
		return "", fmt.Errorf("branch is required")
	}
	if len(v) > 250 {
		return "", fmt.Errorf("branch must be 250 characters or fewer")
	}
	if strings.Contains(v, "..") || strings.Contains(v, " ") {
		return "", fmt.Errorf("branch %q is not valid", raw)
	}
	for _, bad := range []string{"~", "^", ":", "?", "*", "[", "\\", "@{", ".lock"} {
		if strings.Contains(v, bad) {
			return "", fmt.Errorf("branch %q contains invalid characters", raw)
		}
	}
	if !branchRegex.MatchString(v) {
		return "", fmt.Errorf("branch %q is not valid (use letters, numbers, '.', '_', '-' and '/' separators)", raw)
	}
	return v, nil
}

func SanitizeRootDir(raw string) (string, error) {
	v := strings.TrimSpace(raw)
	if v == "" {
		return ".", nil
	}
	clean := strings.ReplaceAll(v, "\\", "/")
	if strings.HasPrefix(clean, "/") {
		return "", fmt.Errorf("root_dir must be relative, got %q", raw)
	}
	parts := strings.Split(clean, "/")
	for _, p := range parts {
		if p == ".." {
			return "", fmt.Errorf("root_dir must not contain '..', got %q", raw)
		}
	}
	out := strings.Join(parts, "/")
	if out == "" || out == "." {
		return ".", nil
	}
	return out, nil
}
