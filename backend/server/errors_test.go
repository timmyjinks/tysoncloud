package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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
