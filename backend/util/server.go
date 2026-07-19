package util

import (
	"regexp"
)

var nameRegex = regexp.MustCompile(`^[A-Za-z-]+$`)
var envRegex = regexp.MustCompile(`\A(?:[A-Za-z_][A-Za-z0-9_]*=[^\n]*)*(?:\n[A-Za-z_][A-Za-z0-9_]*=[^\n]*)*\z`)

func validateName(name string) (bool, error) {
	if len(name) > 24 {
		return false, nil
	}

	return nameRegex.MatchString(name), nil
}

func validateEnv(env string) (bool, error) {
	if env == "" {
		return true, nil
	}

	return envRegex.MatchString(env), nil
}
