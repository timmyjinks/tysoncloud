package store

import (
	"fmt"
	"strings"
)

func rpcError(op string, pgErr PostgrestError) error {
	return fmt.Errorf("%s failed: %s", op, pgErr.Message)
}

func GetPostgresErrorMessage(msg string) (string, bool) {
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "duplicate key value") && (strings.Contains(lower, "public_domain") || strings.Contains(lower, "domain")):
		return "That domain is already taken. Please choose a different one.", true
	case strings.Contains(lower, "duplicate key value"):
		return "That name is already taken. Please choose a different one.", true
	case strings.Contains(lower, "violates check constraint") && strings.Contains(lower, "domain"):
		return "Custom domain must be 1-63 characters, lowercase letters, numbers, and hyphens only, and cannot start or end with a hyphen.", true
	case strings.Contains(lower, "violates check constraint"):
		return "One of the provided values isn't allowed. Please review and try again.", true
	case strings.Contains(lower, "violates foreign key constraint"):
		return "This resource is linked to another resource, so it can't be changed right now.", true
	case strings.Contains(lower, "null value in column"):
		return "A required value is missing. Please fill in all required fields.", true
	case strings.Contains(lower, "value too long"):
		return "One of the values is too long. Please shorten it and try again.", true
	case strings.Contains(lower, "out of range"):
		return "One of the values is too large. Please lower it and try again.", true
	case strings.Contains(lower, "permission denied"), strings.Contains(lower, "row-level security"):
		return "You don't currently have permission to do this.", true
	}
	return "", false
}