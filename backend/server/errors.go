package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/timmyjinks/tysoncloud/store"
)

type Issue struct {
	Line    int    `json:"line,omitempty"`
	Message string `json:"message"`
}

type errorResponse struct {
	Error  string  `json:"error"`
	Issues []Issue `json:"issues,omitempty"`
}

func writeError(w http.ResponseWriter, status int, publicMessage string, internal error) {
	if internal != nil {
		slog.Error(publicMessage, "status", status, "err", internal)
	}
	if status >= http.StatusInternalServerError {
		publicMessage = withCause(publicMessage, internal)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(errorResponse{Error: publicMessage})
}

func writeIssuesError(w http.ResponseWriter, status int, publicMessage string, issues []Issue, internal error) {
	if internal != nil {
		slog.Error(publicMessage, "status", status, "err", internal)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(errorResponse{Error: publicMessage, Issues: issues})
}

var invalidPort error = errors.New("no port found for engine")

func getErrorMessage(internal error) string {
	var apiStatus apierrors.APIStatus
	if errors.As(internal, &apiStatus) {
		status := apiStatus.Status()
		switch {
		case apierrors.IsAlreadyExists(internal):
			return "That name is already taken. Please choose a different one."
		case apierrors.IsForbidden(internal), apierrors.IsUnauthorized(internal):
			return "You don't currently have permission to do this."
		case apierrors.IsInvalid(internal):
			return "One of the provided values isn't allowed. Please review and try again."
		case apierrors.IsTimeout(internal), apierrors.IsServerTimeout(internal), apierrors.IsTooManyRequests(internal), apierrors.IsServiceUnavailable(internal):
			return "We're experiencing high demand right now. Please try again in a moment."
		case strings.Contains(strings.ToLower(status.Message), "quota"):
			return "There isn't enough capacity for this right now. Please try again shortly."
		}
	}

	msg := strings.ToLower(strings.TrimSpace(internal.Error()))
	for _, prefix := range []string{"dial tcp", "dial udp", "connection refused", "connection reset", "i/o timeout", "no such host", "network is unreachable", "lookup "} {
		if strings.HasPrefix(msg, prefix) {
			return "A connection problem occurred. Please try again in a moment."
		}
	}

	return ""
}

func extractCause(internal error) string {
	if internal == nil {
		return ""
	}
	if message := getErrorMessage(internal); message != "" {
		return message
	}

	msg := strings.TrimSpace(internal.Error())
	if i := strings.Index(msg, "failed: "); i >= 0 {
		msg = strings.TrimSpace(msg[i+len("failed: "):])
	}
	if msg == "" {
		return ""
	}
	if message := getErrorMessage(errors.New(msg)); message != "" {
		return message
	}
	if message, ok := store.GetPostgresErrorMessage(msg); ok {
		return message
	}
	return ""
}

func withCause(publicMessage string, internal error) string {
	if cause := extractCause(internal); cause != "" {
		return publicMessage + " " + cause
	}
	return publicMessage
}

const (
	msgUnauthorized = "Your session has expired. Please sign in again."
	msgServerError  = "Something unexpected went wrong while completing your request."
	msgBadRequest   = "That request wasn't valid. Please check it and try again."
)

func isDomainTakenError(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "duplicate") && (strings.Contains(lower, "domain") || strings.Contains(lower, "public_domain"))
}

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "duplicate key value")
}

func isDomainValidationError(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "domain") && (strings.Contains(lower, "check constraint") || strings.Contains(lower, "invalid") || strings.Contains(lower, "violates"))
}

func domainValidationMessage(err error) string {
	lower := strings.ToLower(err.Error())
	if strings.Contains(lower, "length") || strings.Contains(lower, "too long") {
		return "Custom domain must be 1-63 characters."
	}
	return "Custom domain must be 1-63 characters, lowercase letters, numbers, and hyphens only, and cannot start or end with a hyphen."
}

