package server

import (
	"github.com/gorilla/mux"
)

func (s *Application) registerRoutes(
	r *mux.Router,
) error {

	r.HandleFunc("/projects", nil).Methods("GET")
	r.HandleFunc("/projects", nil).Methods("POST")
	r.HandleFunc("/projects", nil).Methods("PUT")
	r.HandleFunc("/projects", nil).Methods("DELETE")

	r.HandleFunc("/projects/{project_id}/services", nil).Methods("GET")
	r.HandleFunc("/projects/{project_id}/services", nil).Methods("POST")
	r.HandleFunc("/projects/{project_id}/services", nil).Methods("PUT")
	r.HandleFunc("/projects/{project_id}/services", nil).Methods("DELETE")

	r.HandleFunc("/projects/{project_id}/services/{service_id}/logs", nil).Methods("GET")
	r.HandleFunc("/tasks/{task_id}", s.HandleTaskWS)

	return nil
}
