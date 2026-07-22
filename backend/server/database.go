package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/clerk/clerk-sdk-go/v2"
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

	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusBadRequest)
		return
	}

	userId := claims.Subject

	database, err := app.Supabase.GetDatabase(dataseId, userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(DatabaseResponse{
		Id:             database.Id,
		ProjectId:      database.ProjectId,
		Name:           database.Name,
		Engine:         database.Engine,
		Port:           database.Port,
		Storage:        database.StorageGB,
		InternalDomain: database.InternalDomain,
		CreatedAt:      database.CreatedAt,
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

	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusBadRequest)
		return
	}

	userId := claims.Subject

	services, err := app.Supabase.GetDatabases(projectId, userId)
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

	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusBadRequest)
		return
	}

	userId := claims.Subject

	res, err := app.Supabase.CreateDatabase(userId, projectId, service.Name, service.Engine, port, service.StorageGB)
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

	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusBadRequest)
		return
	}

	userId := claims.Subject

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

	res, err := app.Supabase.UpdateDatabase(databaseId, userId, *service.Name, *service.StorageGB)
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

	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusBadRequest)
		return
	}

	userId := claims.Subject

	if err := app.Supabase.DeleteDatabase(databaseId, userId); err != nil {
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
			Id:             database.Id,
			ProjectId:      database.ProjectId,
			Name:           database.Name,
			Engine:         database.Engine,
			Port:           database.Port,
			Storage:        database.StorageGB,
			InternalDomain: database.InternalDomain,
			CreatedAt:      database.CreatedAt,
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
