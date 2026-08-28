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
	r.Handle("/projects/{project_id}/services", clerkhttp.RequireHeaderAuthorization()(http.HandlerFunc(s.DeleteServices))).Methods("DELETE")
	r.HandleFunc("/projects/{project_id}/services/{service_id}/logs", s.GetServiceLogs).Methods("GET")
	r.Handle("/projects/{project_id}/services", clerkhttp.RequireHeaderAuthorization()(http.HandlerFunc(s.CreateService))).Methods("POST")
	r.Handle("/projects/{project_id}/services/{service_id}", clerkhttp.RequireHeaderAuthorization()(http.HandlerFunc(s.UpdateService))).Methods("PUT")
	r.Handle("/projects/{project_id}/services/{service_id}", clerkhttp.RequireHeaderAuthorization()(http.HandlerFunc(s.DeleteService))).Methods("DELETE")

	r.Handle("/github_services/{github_service_id}", clerkhttp.RequireHeaderAuthorization()(http.HandlerFunc(s.GetGithubService))).Methods("GET")
	r.Handle("/projects/{project_id}/github_services", clerkhttp.RequireHeaderAuthorization()(http.HandlerFunc(s.GetGithubServices))).Methods("GET")
	r.Handle("/projects/{project_id}/github_services", clerkhttp.RequireHeaderAuthorization()(http.HandlerFunc(s.DeleteGithubServices))).Methods("DELETE")
	r.Handle("/projects/{project_id}/github_services", clerkhttp.RequireHeaderAuthorization()(http.HandlerFunc(s.CreateGithubService))).Methods("POST")
	r.Handle("/projects/{project_id}/github_services/{github_service_id}", clerkhttp.RequireHeaderAuthorization()(http.HandlerFunc(s.UpdateGithubService))).Methods("PUT")
	r.Handle("/projects/{project_id}/github_services/{github_service_id}", clerkhttp.RequireHeaderAuthorization()(http.HandlerFunc(s.DeleteGithubService))).Methods("DELETE")

	r.Handle("/services/{service_id}/volumes", clerkhttp.RequireHeaderAuthorization()(http.HandlerFunc(s.GetVolume))).Methods("GET")
	r.Handle("/projects/{project_id}/services/{service_id}/volumes", clerkhttp.RequireHeaderAuthorization()(http.HandlerFunc(s.CreateVolume))).Methods("POST")
	r.Handle("/projects/{project_id}/services/{service_id}/volumes", clerkhttp.RequireHeaderAuthorization()(http.HandlerFunc(s.DeleteVolume))).Methods("DELETE")

	r.Handle("/databases/{database_id}", clerkhttp.RequireHeaderAuthorization()(http.HandlerFunc(s.GetDatabase))).Methods("GET")
	r.Handle("/projects/{project_id}/databases", clerkhttp.RequireHeaderAuthorization()(http.HandlerFunc(s.GetDatabases))).Methods("GET")
	r.Handle("/projects/{project_id}/databases", clerkhttp.RequireHeaderAuthorization()(http.HandlerFunc(s.DeleteDatabases))).Methods("DELETE")
	r.Handle("/projects/{project_id}/databases", clerkhttp.RequireHeaderAuthorization()(http.HandlerFunc(s.CreateDatabase))).Methods("POST")
	r.Handle("/projects/{project_id}/databases/{database_id}", clerkhttp.RequireHeaderAuthorization()(http.HandlerFunc(s.UpdateDatabase))).Methods("PUT")
	r.Handle("/projects/{project_id}/databases/{database_id}", clerkhttp.RequireHeaderAuthorization()(http.HandlerFunc(s.DeleteDatabase))).Methods("DELETE")

	r.Handle("/projects/{project_id}/config", clerkhttp.RequireHeaderAuthorization()(http.HandlerFunc(s.ConfigProject))).Methods("POST")

	r.Handle("/github/installations/{installation_id}/repositories", clerkhttp.RequireHeaderAuthorization()(http.HandlerFunc(s.GithubRepos))).Methods("GET")

	r.Handle("/github/connections", clerkhttp.RequireHeaderAuthorization()(http.HandlerFunc(s.GetGithubConnections))).Methods("GET")
	r.Handle("/github/connections", clerkhttp.RequireHeaderAuthorization()(http.HandlerFunc(s.CreateGithubConnection))).Methods("POST")
	r.Handle("/github/connections/{connection_id}", clerkhttp.RequireHeaderAuthorization()(http.HandlerFunc(s.DeleteGithubConnection))).Methods("DELETE")

	r.Handle("/github/app", clerkhttp.RequireHeaderAuthorization()(http.HandlerFunc(s.GetGithubApp))).Methods("GET")

	r.HandleFunc("/webhooks/github", s.GithubWebhook).Methods("POST")

	return nil
}
