package server

import (
	"net/http"

	"github.com/gorilla/mux"
)

func (s *Application) registerRoutes(
	r *mux.Router,
) error {

	r.Handle("/projects/{project_id}", s.SupabaseAuthMiddleware(http.HandlerFunc(s.GetProject))).Methods("GET")
	r.Handle("/projects", s.SupabaseAuthMiddleware(http.HandlerFunc(s.GetProjects))).Methods("GET")
	r.Handle("/projects", s.SupabaseAuthMiddleware(http.HandlerFunc(s.CreateProject))).Methods("POST")
	r.Handle("/projects/{project_id}", s.SupabaseAuthMiddleware(http.HandlerFunc(s.UpdateProject))).Methods("PUT")
	r.Handle("/projects/{project_id}", s.SupabaseAuthMiddleware(http.HandlerFunc(s.DeleteProject))).Methods("DELETE")

	r.HandleFunc("/projects/{project_id}/services", nil).Methods("GET")
	r.HandleFunc("/projects/{project_id}/services", nil).Methods("POST")
	r.HandleFunc("/projects/{project_id}/services", nil).Methods("PUT")
	r.HandleFunc("/projects/{project_id}/services", nil).Methods("DELETE")

	r.HandleFunc("/services/{service_id}/logs", nil).Methods("GET")
	r.HandleFunc("/tasks/{task_id}", s.HandleTaskWS)

	return nil
}
