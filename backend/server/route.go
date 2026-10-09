package server

import (
	"net/http"
	"time"

	clerkhttp "github.com/clerk/clerk-sdk-go/v2/http"
	"github.com/gorilla/mux"
	"golang.org/x/time/rate"
)

func (s *Application) registerRoutes(
	r *mux.Router,
) error {

	// Per-user API budget and a tighter one for routes that start builds.
	// The webhook is deliberately not rate limited: a shared bucket checked
	// before signature verification would let unsigned junk starve real
	// GitHub deliveries; webhook builds are bounded by the build semaphore.
	apiLimit := newRateLimiter(rate.Limit(10), 40)
	buildLimit := newRateLimiter(rate.Every(20*time.Second), 5)

	authed := func(h http.HandlerFunc) http.Handler {
		return clerkhttp.RequireHeaderAuthorization()(apiLimit.perUser(h))
	}
	projectOwned := func(h http.HandlerFunc) http.Handler {
		return clerkhttp.RequireHeaderAuthorization()(apiLimit.perUser(s.RequireProjectOwner(h)))
	}
	projectOwnedBuild := func(h http.HandlerFunc) http.Handler {
		return clerkhttp.RequireHeaderAuthorization()(apiLimit.perUser(buildLimit.perUser(s.RequireProjectOwner(h))))
	}

	r.Use(s.CORSMiddleware)
	r.PathPrefix("/").Methods(http.MethodOptions).HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	r.Handle("/projects/{project_id}", projectOwned(s.GetProject)).Methods("GET")
	r.Handle("/projects", authed(s.GetProjects)).Methods("GET")
	r.Handle("/projects", authed(s.CreateProject)).Methods("POST")
	r.Handle("/projects/{project_id}", projectOwned(s.UpdateProject)).Methods("PUT")
	r.Handle("/projects/{project_id}", projectOwned(s.DeleteProject)).Methods("DELETE")

	r.Handle("/services/{service_id}", authed(s.GetService)).Methods("GET")
	r.Handle("/projects/{project_id}/services", projectOwned(s.GetServices)).Methods("GET")
	r.Handle("/projects/{project_id}/services", projectOwned(s.DeleteServices)).Methods("DELETE")
	r.HandleFunc("/projects/{project_id}/services/{service_id}/logs", s.GetServiceLogs).Methods("GET")
	r.Handle("/projects/{project_id}/services", projectOwned(s.CreateService)).Methods("POST")
	r.Handle("/projects/{project_id}/services/{service_id}", projectOwned(s.UpdateService)).Methods("PUT")
	r.Handle("/projects/{project_id}/services/{service_id}", projectOwned(s.DeleteService)).Methods("DELETE")

	r.Handle("/github_services/{github_service_id}", authed(s.GetGithubService)).Methods("GET")
	r.Handle("/projects/{project_id}/github_services", projectOwned(s.GetGithubServices)).Methods("GET")
	r.Handle("/projects/{project_id}/github_services", projectOwned(s.DeleteGithubServices)).Methods("DELETE")
	r.Handle("/projects/{project_id}/github_services", projectOwnedBuild(s.CreateGithubService)).Methods("POST")
	r.Handle("/projects/{project_id}/github_services/{github_service_id}", projectOwnedBuild(s.UpdateGithubService)).Methods("PUT")
	r.Handle("/projects/{project_id}/github_services/{github_service_id}/redeploy", projectOwnedBuild(s.RedeployGithubService)).Methods("POST")
	r.Handle("/projects/{project_id}/github_services/{github_service_id}", projectOwned(s.DeleteGithubService)).Methods("DELETE")
	r.HandleFunc("/projects/{project_id}/github_services/{github_service_id}/logs", s.GetGithubServiceLogs).Methods("GET")

	r.Handle("/projects/{project_id}/preview_environments", projectOwned(s.GetProjectPreviewEnvironments)).Methods("GET")
	r.Handle("/projects/{project_id}/preview_environments/{preview_environment_id}", projectOwned(s.GetProjectPreviewEnvironment)).Methods("GET")
	r.Handle("/projects/{project_id}/preview_environments/{preview_environment_id}/services/{source_service_id}", projectOwned(s.UpdatePreviewEnvironmentService)).Methods("PUT")
	r.Handle("/projects/{project_id}/preview_environments/{preview_environment_id}/services/{source_service_id}", projectOwned(s.DeletePreviewEnvironmentService)).Methods("DELETE")
	r.HandleFunc("/projects/{project_id}/preview_environments/{preview_environment_id}/services/{source_service_id}/logs", s.GetPreviewEnvironmentServiceLogs).Methods("GET")

	r.Handle("/services/{service_id}/volumes", authed(s.GetVolume)).Methods("GET")
	r.Handle("/projects/{project_id}/services/{service_id}/volumes", projectOwned(s.CreateVolume)).Methods("POST")
	r.Handle("/projects/{project_id}/services/{service_id}/volumes", projectOwned(s.DeleteVolume)).Methods("DELETE")

	r.Handle("/databases/{database_id}", authed(s.GetDatabase)).Methods("GET")
	r.Handle("/projects/{project_id}/databases", projectOwned(s.GetDatabases)).Methods("GET")
	r.Handle("/projects/{project_id}/databases", projectOwned(s.DeleteDatabases)).Methods("DELETE")
	r.Handle("/projects/{project_id}/databases", projectOwned(s.CreateDatabase)).Methods("POST")
	r.Handle("/projects/{project_id}/databases/{database_id}", projectOwned(s.UpdateDatabase)).Methods("PUT")
	r.Handle("/projects/{project_id}/databases/{database_id}", projectOwned(s.DeleteDatabase)).Methods("DELETE")

	r.Handle("/projects/{project_id}/config", projectOwned(s.ConfigProject)).Methods("POST")

	r.Handle("/github/installations/{installation_id}/repositories", authed(s.GithubRepos)).Methods("GET")

	r.Handle("/github/connections", authed(s.GetGithubConnections)).Methods("GET")
	r.Handle("/github/connections", authed(s.CreateGithubConnection)).Methods("POST")
	r.Handle("/github/connections/{connection_id}", authed(s.DeleteGithubConnection)).Methods("DELETE")

	r.Handle("/github/app", authed(s.GetGithubApp)).Methods("GET")

	r.HandleFunc("/webhooks/github", s.GithubWebhook).Methods("POST")

	return nil
}
