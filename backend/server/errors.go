package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
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
	// For 5xx failures the caller only knows the generic story ("couldn't
	// create the service"). Surface the underlying reason too, so the user
	// isn't stuck with "please try again" and no idea why it failed.
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

// unusableCauses are internal errors that carry no user-actionable meaning.
var unusableCauses = []string{
	"context canceled",
	"context deadline exceeded",
	"json: ",
	"http: ",
}

// extractCause pulls a concise, user-facing reason out of an internal error,
// returning "" when there's nothing safe or useful to surface.
func extractCause(internal error) string {
	if internal == nil {
		return ""
	}
	msg := strings.TrimSpace(internal.Error())
	if msg == "" {
		return ""
	}

	// The store layer wraps RPC failures as "<op> failed: <detail>" —
	// surface the detail, not the wrapper.
	if i := strings.Index(msg, "failed: "); i >= 0 {
		msg = strings.TrimSpace(msg[i+len("failed: "):])
	}
	if msg == "" {
		return ""
	}

	for _, prefix := range unusableCauses {
		if strings.HasPrefix(msg, prefix) {
			return ""
		}
	}
	// URLs point at internal endpoints (Supabase/k8s) — don't leak them.
	if strings.Contains(msg, "://") {
		return ""
	}

	// Collapse to the first line and cap the length so an API/library
	// doesn't dump a wall of text into the UI.
	if i := strings.IndexByte(msg, '\n'); i >= 0 {
		msg = msg[:i]
	}
	if len(msg) > 200 {
		msg = msg[:200]
	}
	msg = strings.TrimRight(msg, " \t.")

	return msg
}

// withCause appends the underlying cause to a public message when one can be
// safely extracted, so users see why an action failed instead of a bare
// "try again".
func withCause(publicMessage string, internal error) string {
	if cause := extractCause(internal); cause != "" {
		return publicMessage + " " + cause
	}
	return publicMessage
}

const (
	msgUnauthorized = "Please sign in again."
	msgServerError  = "Something went wrong on our end. Please try again."
	msgBadRequest   = "That request wasn't valid."
)

func isDomainTakenError(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "duplicate") && (strings.Contains(lower, "domain") || strings.Contains(lower, "public_domain"))
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

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "duplicate key value")
}
