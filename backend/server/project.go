package server

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/BurntSushi/toml"
	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/gorilla/mux"
	"github.com/timmyjinks/tysoncloud/deploy"
	"github.com/timmyjinks/tysoncloud/store"
)

func (app *Application) GetProject(w http.ResponseWriter, r *http.Request) {
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

	project, err := app.Supabase.GetProject(projectId, claims.Subject)
	if err != nil {
		writeError(w, http.StatusNotFound, "We couldn't find that project.", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(ProjectResponse{Id: project.Id, Name: project.Name}); err != nil {
		writeError(w, http.StatusInternalServerError, msgServerError, err)
		return
	}
}

func (app *Application) GetProjects(w http.ResponseWriter, r *http.Request) {
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, msgUnauthorized, nil)
		return
	}

	projects, err := app.Supabase.GetProjects(claims.Subject)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Couldn't load your projects. Please try again.", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(ToProjectsResponse(projects)); err != nil {
		writeError(w, http.StatusInternalServerError, msgServerError, err)
		return
	}
}

func (app *Application) CreateProject(w http.ResponseWriter, r *http.Request) {
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, msgUnauthorized, nil)
		return
	}

	var project ProjectCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&project); err != nil {
		writeError(w, http.StatusBadRequest, "That project request wasn't valid.", err)
		return
	}

	if project.Name == "" {
		writeError(w, http.StatusBadRequest, "Project name is required.", nil)
		return
	}

	res, err := app.Supabase.CreateProject(claims.Subject, project.Name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Couldn't create the project. Please try again.", err)
		return
	}

	if err := app.Deploy.CreateProject(r.Context(), res.Namespace); err != nil {
		writeError(w, http.StatusInternalServerError, "The project was created, but we couldn't finish setting up its infrastructure. Please try again or contact support.", err)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (app *Application) UpdateProject(w http.ResponseWriter, r *http.Request) {
	projectId := mux.Vars(r)["project_id"]
	if projectId == "" {
		writeError(w, http.StatusBadRequest, "A project ID is required.", nil)
		return
	}

	var project ProjectUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&project); err != nil {
		writeError(w, http.StatusBadRequest, "That project request wasn't valid.", err)
		return
	}

	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, msgUnauthorized, nil)
		return
	}

	if project.Name == nil || *project.Name == "" {
		writeError(w, http.StatusBadRequest, "Project name is required.", nil)
		return
	}

	if err := app.Supabase.UpdateProject(projectId, claims.Subject, *project.Name); err != nil {
		writeError(w, http.StatusInternalServerError, "Couldn't save the project name. Please try again.", err)
		return
	}
}

func (app *Application) DeleteProject(w http.ResponseWriter, r *http.Request) {
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

	if err := app.Supabase.DeleteProject(claims.Subject, projectId); err != nil {
		writeError(w, http.StatusInternalServerError, "Couldn't delete the project. Please try again.", err)
		return
	}

	if err := app.Deploy.DeleteProject(r.Context(), "proj-"+projectId); err != nil {
		slog.Error("failed to clean up project namespace", "project_id", projectId, "err", err)
	}

	w.WriteHeader(204)
}

func (app *Application) ConfigProject(w http.ResponseWriter, r *http.Request) {
	projectId := mux.Vars(r)["project_id"]
	if projectId == "" {
		writeError(w, http.StatusBadRequest, "A project ID is required.", nil)
		return
	}

	var data ProjectConfigRequest

	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, msgUnauthorized, nil)
		return
	}
	userId := claims.Subject

	var config Config

	if _, err := toml.Decode(data.Content, &config); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := ValidateToml(config); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var serviceTables []store.ServicesTable
	for _, service := range config.Services {
		res, err := app.Supabase.CreateService(userId, projectId, service.Name, service.Image, int32(service.Port))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Couldn't create the service. Please try again.", err)
			return
		}
		serviceTables = append(serviceTables, res)
	}

	var databaseTables []store.DatabasesTable
	for _, database := range config.Databases {
		port, err := getPort(database.Engine)
		if err != nil {
			return
		}

		res, err := app.Supabase.CreateDatabase(userId, projectId, database.Name, database.Engine, port, int32(database.StorageGB))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Couldn't create the service. Please try again.", err)
			return
		}
		databaseTables = append(databaseTables, res)
	}

	services, databases := ToProjectData(serviceTables, databaseTables)

	if err := app.Deploy.BatchCreateServices(r.Context(), services); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for i, service := range config.Services {
		if service.Volume != nil {
			if _, err := app.Supabase.CreateVolume(serviceTables[i].Id, userId, service.Volume.MountPath, int32(service.Volume.StorageGB)); err != nil {
				writeError(w, http.StatusInternalServerError, "Couldn't attach the volume. Please try again.", err)
				return
			}

			app.Deploy.AttachVolume(r.Context(), deploy.Service{
				Namespace: "proj-" + projectId,
				Name:      serviceTables[i].ResourceName,
			}, deploy.Volume{
				MountPath: service.Volume.MountPath,
				StorageGB: int32(service.Volume.StorageGB),
			})

		}
	}

	if err := app.Deploy.BatchCreateDatabases(r.Context(), databases); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, service := range serviceTables {
		if err := app.Cloudflare.CreateRecord(r.Context(), "tc-"+service.Id); err != nil {
			writeError(w, http.StatusInternalServerError, "The service deployed, but we couldn't set up its domain. Please try again or contact support.", err)
			return
		}

		if err := app.Cloudflare.CreateRoute(r.Context(), "tc-"+service.Id); err != nil {
			writeError(w, http.StatusInternalServerError, "The service deployed, but we couldn't finish routing its domain. Please try again or contact support.", err)
			return
		}
	}
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

func ValidateToml(config Config) error {
	for _, service := range config.Services {
		if service.Name == "" {
			return emptyName
		}
		if service.Image == "" {
			return emptyImage
		}
		if service.Port == 0 {
			return errors.New("empty port")
		}
	}

	for _, database := range config.Databases {
		if database.Name == "" {
			return emptyName
		}
		if database.Engine == "" {
			return emptyImage
		}
		if database.StorageGB == 0 {
			return errors.New("empty storage_gb")
		}
	}

	return nil
}

func ToProjectData(services []store.ServicesTable, databases []store.DatabasesTable) ([]deploy.Service, []deploy.Database) {
	var servicesData []deploy.Service = []deploy.Service{}
	var databasesData []deploy.Database = []deploy.Database{}

	for _, service := range services {
		servicesData = append(servicesData, deploy.Service{
			Namespace: "proj-" + service.ProjectId,
			Name:      service.ResourceName,
			Hostname:  service.PublicDomain,
			Image:     service.Image,
			Port:      int32(service.Port),
			Env:       map[string][]byte{},
		})
	}

	for _, database := range databases {
		databasesData = append(databasesData, deploy.Database{
			Namespace: "proj-" + database.ProjectId,
			Name:      database.ResourceName,
			Engine:    database.Engine,
			StorageGB: int32(database.StorageGB),
		})
	}

	return servicesData, databasesData
}
