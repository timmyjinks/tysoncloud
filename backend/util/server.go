package util

import (
	"fmt"
	"regexp"
	"strings"
)

var nameRegex = regexp.MustCompile(`^[A-Za-z-]+$`)
var envRegex = regexp.MustCompile(`\A(?:[A-Za-z_][A-Za-z0-9_]*=[^\n]*)*(?:\n[A-Za-z_][A-Za-z0-9_]*=[^\n]*)*\z`)
var domainLabelRegex = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)

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
