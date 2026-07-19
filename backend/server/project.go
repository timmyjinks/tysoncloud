package server

import (
	"encoding/json"
	"errors"
	"net/http"

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

	client := ClientFromContext(r.Context())
	if client == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := client.Auth.GetUser()
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	project, err := app.Supabase.GetProject(projectId, user.ID.String())
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
	client := ClientFromContext(r.Context())
	if client == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := client.Auth.GetUser()
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	projects, err := app.Supabase.GetProjects(user.ID.String())
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
	var project ProjectCreateRequest
	err := json.NewDecoder(r.Body).Decode(&project)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	client := ClientFromContext(r.Context())
	if client == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := client.Auth.GetUser()
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if err := app.Supabase.CreateProject(user.ID.String(), project.Name); err != nil {
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

	client := ClientFromContext(r.Context())
	if client == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := client.Auth.GetUser()
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if project.Name == nil {
		http.Error(w, emptyName.Error(), http.StatusBadRequest)
		return
	}

	if err := app.Supabase.UpdateProject(projectId, user.ID.String(), *project.Name); err != nil {
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

	client := ClientFromContext(r.Context())
	if client == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	user, err := client.Auth.GetUser()
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if err := app.Supabase.DeleteProject(user.ID.String(), projectId); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(204)
}

func ToProjectsResponse(projectsTable []store.ProjectsTable) []ProjectResponse {
	var projects []ProjectResponse
	for _, project := range projectsTable {
		projects = append(projects, ProjectResponse{
			Id:   project.Id,
			Name: project.Name,
		})
	}
	return projects
}
