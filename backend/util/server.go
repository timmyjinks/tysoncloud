package util

import (
	"regexp"
	"strings"
)

var nameRegex = regexp.MustCompile(`^[A-Za-z-]+$`)
var envRegex = regexp.MustCompile(`\A(?:[A-Za-z_][A-Za-z0-9_]*=[^\n]*)*(?:\n[A-Za-z_][A-Za-z0-9_]*=[^\n]*)*\z`)

func validateName(name string) (bool, error) {
	if len(name) > 24 {
		return false, nil
	}

	return nameRegex.MatchString(name), nil
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
