package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var testGR = schema.GroupResource{Group: "apps", Resource: "deployments"}

func TestGetErrorMessage(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"already exists", apierrors.NewAlreadyExists(testGR, "web"), "That name is already taken. Please choose a different one."},
		{"forbidden", apierrors.NewForbidden(testGR, "web", errors.New("nope")), "You don't currently have permission to do this."},
		{"unauthorized", apierrors.NewUnauthorized("nope"), "You don't currently have permission to do this."},
		{"invalid", apierrors.NewInvalid(schema.GroupKind{Kind: "Deployment"}, "web", nil), "One of the provided values isn't allowed. Please review and try again."},
		{"timeout", apierrors.NewTimeoutError("slow", 1), "We're experiencing high demand right now. Please try again in a moment."},
		{"too many requests", apierrors.NewTooManyRequests("slow down", 1), "We're experiencing high demand right now. Please try again in a moment."},
		{"service unavailable", apierrors.NewServiceUnavailable("down"), "We're experiencing high demand right now. Please try again in a moment."},
		{"quota", apierrors.NewBadRequest("exceeded quota: compute-resources"), "There isn't enough capacity for this right now. Please try again shortly."},
		{"not found has no friendly message", apierrors.NewNotFound(testGR, "web"), ""},
		{"dial tcp", errors.New("dial tcp 10.0.0.1:443: i/o timeout"), "A connection problem occurred. Please try again in a moment."},
		{"connection refused", errors.New("Connection refused"), "A connection problem occurred. Please try again in a moment."},
		{"dns lookup", errors.New("lookup api.example: no such host"), "A connection problem occurred. Please try again in a moment."},
		{"unrelated", errors.New("something odd"), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getErrorMessage(tt.err); got != tt.want {
				t.Fatalf("getErrorMessage = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractCause(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"nil", nil, ""},
		{"k8s error", apierrors.NewAlreadyExists(testGR, "web"), "That name is already taken. Please choose a different one."},
		{"wrapped k8s error", fmt.Errorf("create: %w", apierrors.NewAlreadyExists(testGR, "web")), "That name is already taken. Please choose a different one."},
		{"network after failed:", errors.New("create_service failed: dial tcp 1.2.3.4:5432: connect: refused"), "A connection problem occurred. Please try again in a moment."},
		{"postgres after failed:", errors.New("create_service failed: duplicate key value violates unique constraint \"public_domain\""), "That domain is already taken. Please choose a different one."},
		{"postgres without prefix", errors.New("null value in column \"name\""), "A required value is missing. Please fill in all required fields."},
		{"unknown", errors.New("rpc failed: weird"), ""},
		{"empty after failed:", errors.New("rpc failed: "), ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractCause(tt.err); got != tt.want {
				t.Fatalf("extractCause = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWithCause(t *testing.T) {
	if got := withCause("Couldn't create.", errors.New("weird")); got != "Couldn't create." {
		t.Errorf("unknown cause changed message: %q", got)
	}
	got := withCause("Couldn't create.", errors.New("duplicate key value"))
	if got != "Couldn't create. That name is already taken. Please choose a different one." {
		t.Errorf("withCause = %q", got)
	}
}

func decodeError(t *testing.T, rec *httptest.ResponseRecorder) errorResponse {
	t.Helper()
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}
	var body errorResponse
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	return body
}

func TestWriteError(t *testing.T) {
	t.Run("4xx keeps public message only", func(t *testing.T) {
		rec := httptest.NewRecorder()
		writeError(rec, http.StatusBadRequest, msgBadRequest, errors.New("duplicate key value"))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d", rec.Code)
		}
		if body := decodeError(t, rec); body.Error != msgBadRequest {
			t.Fatalf("error = %q, want %q", body.Error, msgBadRequest)
		}
	})

	t.Run("5xx appends friendly cause", func(t *testing.T) {
		rec := httptest.NewRecorder()
		writeError(rec, http.StatusInternalServerError, msgServerError, errors.New("duplicate key value"))
		want := msgServerError + " That name is already taken. Please choose a different one."
		if body := decodeError(t, rec); body.Error != want {
			t.Fatalf("error = %q, want %q", body.Error, want)
		}
	})

	t.Run("5xx never leaks raw internal error", func(t *testing.T) {
		rec := httptest.NewRecorder()
		writeError(rec, http.StatusInternalServerError, msgServerError, errors.New("secret table xyz exploded"))
		if body := decodeError(t, rec); body.Error != msgServerError {
			t.Fatalf("error = %q, want %q", body.Error, msgServerError)
		}
	})
}

func TestWriteIssuesError(t *testing.T) {
	rec := httptest.NewRecorder()
	writeIssuesError(rec, http.StatusBadRequest, "bad config", []Issue{{Line: 3, Message: "oops"}}, nil)
	body := decodeError(t, rec)
	if rec.Code != http.StatusBadRequest || body.Error != "bad config" || len(body.Issues) != 1 || body.Issues[0].Line != 3 {
		t.Fatalf("got %d %+v", rec.Code, body)
	}
}

func TestDomainErrorClassifiers(t *testing.T) {
	tests := []struct {
		err                         error
		taken, duplicate, invalidDN bool
	}{
		{nil, false, false, false},
		// Also matches the validation check ("violates" + "domain"), so handlers
		// must test isDomainTakenError first to return 409 rather than 400.
		{errors.New(`duplicate key value violates unique constraint "services_public_domain_key"`), true, true, true},
		{errors.New(`duplicate key value violates unique constraint "services_name_key"`), false, true, false},
		{errors.New(`new row violates check constraint "public_domain_format"`), false, false, true},
		{errors.New("invalid domain"), false, false, true},
		{errors.New("unrelated"), false, false, false},
	}
	for _, tt := range tests {
		if got := isDomainTakenError(tt.err); got != tt.taken {
			t.Errorf("isDomainTakenError(%v) = %v, want %v", tt.err, got, tt.taken)
		}
		if got := isDuplicateKeyError(tt.err); got != tt.duplicate {
			t.Errorf("isDuplicateKeyError(%v) = %v, want %v", tt.err, got, tt.duplicate)
		}
		if got := isDomainValidationError(tt.err); got != tt.invalidDN {
			t.Errorf("isDomainValidationError(%v) = %v, want %v", tt.err, got, tt.invalidDN)
		}
	}
}

func TestDomainValidationMessage(t *testing.T) {
	if got := domainValidationMessage(errors.New("domain value too long")); got != "Custom domain must be 1-63 characters." {
		t.Errorf("length message = %q", got)
	}
	if got := domainValidationMessage(errors.New("domain violates check constraint")); got == "Custom domain must be 1-63 characters." {
		t.Errorf("format violation got length-only message")
	}
}

func TestWriteErrorRedactsInternalDetails(t *testing.T) {
	tests := []struct {
		name     string
		internal error
		want     string
	}{
		{
			name:     "raw internal error is not exposed",
			internal: errors.New(`rpc create_service failed: pq: relation "secret_table" at 10.43.0.5:5432`),
			want:     msgServerError,
		},
		{
			name:     "curated postgres message is appended",
			internal: errors.New("create failed: duplicate key value violates unique constraint"),
			want:     msgServerError + " That name is already taken. Please choose a different one.",
		},
		{
			name:     "network errors are generalized",
			internal: errors.New("dial tcp 10.43.208.127:5000: connect: connection refused"),
			want:     msgServerError + " A connection problem occurred. Please try again in a moment.",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			writeError(w, http.StatusInternalServerError, msgServerError, tt.internal)
			var body errorResponse
			if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body.Error != tt.want {
				t.Fatalf("error = %q, want %q", body.Error, tt.want)
			}
			if strings.Contains(body.Error, "10.43.") {
				t.Fatalf("response leaks internal address: %q", body.Error)
			}
		})
	}
}
