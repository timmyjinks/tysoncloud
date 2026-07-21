package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/gorilla/mux"
	"github.com/timmyjinks/tysoncloud/store"
)

var invalidProjectId error = errors.New("project with id not found")

func (app *Application) GetProject(w http.ResponseWriter, r *http.Request) {
	projectId := mux.Vars(r)["project_id"]
	if projectId == "" {
		http.Error(w, "project with id not found", http.StatusBadRequest)
		return
	}

	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userId := claims.Subject

	project, err := app.Supabase.GetProject(projectId, userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(ProjectResponse{Id: project.Id, Name: project.Name}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (app *Application) GetProjects(w http.ResponseWriter, r *http.Request) {
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userId := claims.Subject

	projects, err := app.Supabase.GetProjects(userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(ToProjectsResponse(projects)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (app *Application) CreateProject(w http.ResponseWriter, r *http.Request) {
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusBadRequest)
		return
	}

	userId := claims.Subject

	var project ProjectCreateRequest
	err := json.NewDecoder(r.Body).Decode(&project)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	res, err := app.Supabase.CreateProject(userId, project.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := app.Deploy.CreateProject(r.Context(), res.Namespace); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (app *Application) UpdateProject(w http.ResponseWriter, r *http.Request) {
	projectId := mux.Vars(r)["project_id"]
	if projectId == "" {
		http.Error(w, invalidProjectId.Error(), http.StatusBadRequest)
		return
	}

	var project ProjectUpdateRequest
	err := json.NewDecoder(r.Body).Decode(&project)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userId := claims.Subject

	if project.Name == nil {
		http.Error(w, emptyName.Error(), http.StatusBadRequest)
		return
	}

	if err := app.Supabase.UpdateProject(projectId, userId, *project.Name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

func (app *Application) DeleteProject(w http.ResponseWriter, r *http.Request) {
	projectId := mux.Vars(r)["project_id"]
	if projectId == "" {
		http.Error(w, "project with id not found", http.StatusBadRequest)
		return
	}

	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	userId := claims.Subject

	if err := app.Supabase.DeleteProject(userId, projectId); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := app.Deploy.DeleteProject(r.Context(), "proj-"+projectId); err != nil {
		slog.Error(err.Error())
	}

	w.WriteHeader(204)
}

func ToProjectsResponse(projectsTable []store.ProjectsTable) []ProjectResponse {
	var projects []ProjectResponse = []ProjectResponse{}
	for _, project := range projectsTable {
		projects = append(projects, ProjectResponse{
			Id:   project.Id,
			Name: project.Name,
		})
	}
	return projects
}
