package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/timmyjinks/tysoncloud/deploy"
	"github.com/timmyjinks/tysoncloud/store"
)

var invalidDatabaseId error = errors.New("database with id not found")
var invalidPort error = errors.New("no port found for engine")
var invalidEngine error = errors.New("no engine found")
var invalidStorageGB error = errors.New("no storage amount was specified")

func (app *Application) GetDatabase(w http.ResponseWriter, r *http.Request) {
	dataseId := mux.Vars(r)["database_id"]
	if dataseId == "" {
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

	database, err := app.Supabase.GetDatabase(dataseId, user.ID.String())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(DatabaseResponse{
		Id:        database.Id,
		ProjectId: database.ProjectId,
		Name:      database.Name,
		Engine:    database.Engine,
		Port:      database.Port,
		Storage:   database.StorageGB,
		CreatedAt: database.CreatedAt,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (app *Application) GetDatabases(w http.ResponseWriter, r *http.Request) {
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

	services, err := app.Supabase.GetDatabases(projectId, user.ID.String())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(ToDatabasesResponse(services)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (app *Application) CreateDatabase(w http.ResponseWriter, r *http.Request) {
	projectId := mux.Vars(r)["project_id"]

	var service DatabaseCreateRequest
	err := json.NewDecoder(r.Body).Decode(&service)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	port, err := getPort(service.Engine)
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

	res, err := app.Supabase.CreateDatabase(user.ID.String(), projectId, service.Name, service.Engine, port, service.StorageGB)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := app.Deploy.CreateDatabase(r.Context(), deploy.Database{
		Namespace: "proj-" + projectId,
		Name:      res.ResourceName,
		Engine:    service.Engine,
		StorageGB: service.StorageGB,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (app *Application) UpdateDatabase(w http.ResponseWriter, r *http.Request) {
	projectId := mux.Vars(r)["project_id"]
	if projectId == "" {
		http.Error(w, "project with id not found", http.StatusBadRequest)
		return
	}

	databaseId := mux.Vars(r)["database_id"]
	if databaseId == "" {
		http.Error(w, invalidDatabaseId.Error(), http.StatusBadRequest)
		return
	}

	var service DatabaseUpdateRequest
	err := json.NewDecoder(r.Body).Decode(&service)
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

	if service.Name == nil {
		http.Error(w, emptyName.Error(), http.StatusBadRequest)
		return
	}

	if service.Engine == nil {
		http.Error(w, invalidEngine.Error(), http.StatusBadRequest)
		return
	}

	if service.StorageGB == nil {
		http.Error(w, invalidStorageGB.Error(), http.StatusBadRequest)
		return
	}

	res, err := app.Supabase.UpdateDatabase(databaseId, user.ID.String(), *service.Name, *service.StorageGB)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := app.Deploy.CreateDatabase(r.Context(), deploy.Database{
		Namespace: "proj-" + projectId,
		Name:      res.ResourceName,
		Engine:    res.Engine,
		StorageGB: *service.StorageGB,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

}

func (app *Application) DeleteDatabase(w http.ResponseWriter, r *http.Request) {
	projectId := mux.Vars(r)["project_id"]
	if projectId == "" {
		http.Error(w, "project with id not found", http.StatusBadRequest)
		return
	}

	databaseId := mux.Vars(r)["database_id"]
	if databaseId == "" {
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

	if err := app.Supabase.DeleteDatabase(databaseId, user.ID.String()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := app.Deploy.DeleteDatabase(r.Context(), deploy.Database{
		Namespace: "proj-" + projectId,
		Name:      "db-" + databaseId,
	}); err != nil {
		return
	}

	w.WriteHeader(204)
}

func ToDatabasesResponse(databasesTable []store.DatabasesTable) []DatabaseResponse {
	var databases []DatabaseResponse = []DatabaseResponse{}
	for _, database := range databasesTable {
		databases = append(databases, DatabaseResponse{
			Id:        database.Id,
			ProjectId: database.ProjectId,
			Name:      database.Name,
			Engine:    database.Engine,
			Port:      database.Port,
			Storage:   database.StorageGB,
			CreatedAt: database.CreatedAt,
		})
	}
	return databases
}

func getPort(engine string) (int32, error) {
	switch engine {
	case "postgres":
		return 5432, nil
	default:
		return -1, invalidPort
	}
}
