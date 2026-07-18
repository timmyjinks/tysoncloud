package util

import (
	"errors"
	"regexp"
)

func validateName(name string) (bool, error) {
	if len(name) > 24 {
		return false, nil
	}

	r, err := regexp.Compile(`^[A-Za-z-]*$`)
	if err != nil {
		return false, err
	}

	return r.MatchString(name), nil
}

func validateEnv(env string) (bool, error) {
	if env == "" {
		return true, nil
	}

	r, err := regexp.Compile(`\A(?:[A-Za-z_][A-Za-z0-9_]*=[^\n]*)*(?:\n[A-Za-z_][A-Za-z0-9_]*=[^\n]*)*\z`)
	if err != nil {
		return false, err
	}

	return r.MatchString(env), nil
}

func validateRequest(name, env string) error {
	nameOk, err := validateName(name)
	if err != nil {
		return err
	}

	if !nameOk {
		return errors.New("Invalid name, can only have [a-zA-Z-] and up to 24 characters")
	}

	envOk, err := validateEnv(env)
	if err != nil {
		return err
	}

	if !envOk {
		return errors.New("Invalid env, must follow key=value format")

	}

	return nil
}
