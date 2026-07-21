package server

import (
	"net/http"

	clerkhttp "github.com/clerk/clerk-sdk-go/v2/http"
	"github.com/gorilla/mux"
)

func (s *Application) registerRoutes(
	r *mux.Router,
) error {

	r.Use(s.CORSMiddleware)
	r.PathPrefix("/").Methods(http.MethodOptions).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	r.Handle("/projects/{project_id}", clerkhttp.RequireHeaderAuthorization()(http.HandlerFunc(s.GetProject))).Methods("GET")
	r.Handle("/projects", clerkhttp.RequireHeaderAuthorization()(http.HandlerFunc(s.GetProjects))).Methods("GET")
	r.Handle("/projects", clerkhttp.RequireHeaderAuthorization()(http.HandlerFunc(s.CreateProject))).Methods("POST")
	r.Handle("/projects/{project_id}", clerkhttp.RequireHeaderAuthorization()(http.HandlerFunc(s.UpdateProject))).Methods("PUT")
	r.Handle("/projects/{project_id}", clerkhttp.RequireHeaderAuthorization()(http.HandlerFunc(s.DeleteProject))).Methods("DELETE")

	r.Handle("/services/{service_id}", clerkhttp.RequireHeaderAuthorization()(http.HandlerFunc(s.GetService))).Methods("GET")
	r.Handle("/projects/{project_id}/services", clerkhttp.RequireHeaderAuthorization()(http.HandlerFunc(s.GetServices))).Methods("GET")
	r.Handle("/projects/{project_id}/services", clerkhttp.RequireHeaderAuthorization()(http.HandlerFunc(s.CreateService))).Methods("POST")
	r.Handle("/projects/{project_id}/services/{service_id}", s.SupabaseAuthMiddleware(http.HandlerFunc(s.UpdateService))).Methods("PUT")
	r.Handle("/projects/{project_id}/services/{service_id}", s.SupabaseAuthMiddleware(http.HandlerFunc(s.DeleteService))).Methods("DELETE")

	r.Handle("/services/{service_id}/volumes", s.SupabaseAuthMiddleware(http.HandlerFunc(s.GetVolume))).Methods("GET")
	r.Handle("/projects/{project_id}/services/{service_id}/volumes", s.SupabaseAuthMiddleware(http.HandlerFunc(s.CreateVolume))).Methods("POST")
	r.Handle("/projects/{project_id}/services/{service_id}/volumes", s.SupabaseAuthMiddleware(http.HandlerFunc(s.DeleteVolume))).Methods("DELETE")

	r.Handle("/databases/{database_id}", s.SupabaseAuthMiddleware(http.HandlerFunc(s.GetDatabase))).Methods("GET")
	r.Handle("/projects/{project_id}/databases", s.SupabaseAuthMiddleware(http.HandlerFunc(s.GetDatabases))).Methods("GET")
	r.Handle("/projects/{project_id}/databases", s.SupabaseAuthMiddleware(http.HandlerFunc(s.CreateDatabase))).Methods("POST")
	r.Handle("/projects/{project_id}/databases/{database_id}", s.SupabaseAuthMiddleware(http.HandlerFunc(s.UpdateDatabase))).Methods("PUT")
	r.Handle("/projects/{project_id}/databases/{database_id}", s.SupabaseAuthMiddleware(http.HandlerFunc(s.DeleteDatabase))).Methods("DELETE")

	r.HandleFunc("/services/{service_id}/logs", nil).Methods("GET")
	r.HandleFunc("/tasks/{task_id}", s.HandleTaskWS)

	return nil
}
