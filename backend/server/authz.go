package server

import (
	"net/http"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/gorilla/mux"
)

func (app *Application) RequireProjectOwner(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		projectId := mux.Vars(r)["project_id"]
		if projectId == "" {
			writeError(w, http.StatusBadRequest, "A project ID is required.", nil)
			return
		}

		claims, ok := clerk.SessionClaimsFromContext(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, msgUnauthorized, nil)
			return
		}

		if _, err := app.Supabase.GetProject(projectId, claims.Subject); err != nil {
			writeError(w, http.StatusNotFound, "We couldn't find that project.", err)
			return
		}

		next.ServeHTTP(w, r)
	})
}
