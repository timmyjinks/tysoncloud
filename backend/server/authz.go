package server

import (
	"net/http"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/gorilla/mux"
)

// RequireProjectOwner rejects requests whose {project_id} route variable is
// not a project owned by the authenticated user. Handlers behind it can use
// the URL project_id to target Kubernetes namespaces. It must run after
// Clerk's header-auth middleware so session claims are in the context.
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
