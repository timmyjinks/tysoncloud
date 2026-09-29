package server

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/gorilla/mux"
	"github.com/timmyjinks/tysoncloud/deploy"
	"github.com/timmyjinks/tysoncloud/store"
)

func (app *Application) GetDatabase(w http.ResponseWriter, r *http.Request) {
	databaseId := mux.Vars(r)["database_id"]
	if databaseId == "" {
		writeError(w, http.StatusBadRequest, "A database ID is required.", nil)
		return
	}

	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, msgUnauthorized, nil)
		return
	}

	database, err := app.Supabase.GetDatabase(databaseId, claims.Subject)
	if err != nil {
		writeError(w, http.StatusNotFound, "We couldn't find that database.", err)
		return
	}

	env, err := app.Deploy.GetServiceEnv(r.Context(), deploy.Service{
		Namespace: "proj-" + database.ProjectId,
		Name:      database.ResourceName + "-app",
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Couldn't load the database's connection details.", err)
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
		Env:            env,
		CreatedAt:      database.CreatedAt,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, msgServerError, err)
		return
	}
}

func (app *Application) GetDatabases(w http.ResponseWriter, r *http.Request) {
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

	databases, err := app.Supabase.GetDatabases(projectId, claims.Subject)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Couldn't load the project's databases.", err)
		return
	}
	if copies, cerr := app.Supabase.GetPreviewCopyIDs(projectId); cerr == nil {
		databases = filterPreviewCopies(databases, func(d store.DatabasesTable) string { return d.Id }, copies)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(ToDatabasesResponse(databases)); err != nil {
		writeError(w, http.StatusInternalServerError, msgServerError, err)
		return
	}
}

func (app *Application) CreateDatabase(w http.ResponseWriter, r *http.Request) {
	projectId := mux.Vars(r)["project_id"]

	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, msgUnauthorized, nil)
		return
	}

	var database DatabaseCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&database); err != nil {
		writeError(w, http.StatusBadRequest, "That database request wasn't valid.", err)
		return
	}

	if database.Name == "" {
		writeError(w, http.StatusBadRequest, "Database name is required.", nil)
		return
	}

	port, err := getPort(database.Engine)
	if err != nil {
		writeError(w, http.StatusBadRequest, "That database engine isn't supported.", err)
		return
	}

	userId := claims.Subject

	res, err := app.Supabase.CreateDatabase(userId, projectId, database.Name, database.Engine, port, database.StorageGB)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Couldn't create the database.", err)
		return
	}

	if err := app.Deploy.CreateDatabase(r.Context(), deploy.Database{
		Namespace: "proj-" + projectId,
		Name:      res.ResourceName,
		Engine:    database.Engine,
		StorageGB: database.StorageGB,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "Your database was created, but we couldn't finish setting it up. A refresh will show its current status.", err)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (app *Application) UpdateDatabase(w http.ResponseWriter, r *http.Request) {
	projectId := mux.Vars(r)["project_id"]
	if projectId == "" {
		writeError(w, http.StatusBadRequest, "A project ID is required.", nil)
		return
	}

	databaseId := mux.Vars(r)["database_id"]
	if databaseId == "" {
		writeError(w, http.StatusBadRequest, "A database ID is required.", nil)
		return
	}

	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, msgUnauthorized, nil)
		return
	}

	var database DatabaseUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&database); err != nil {
		writeError(w, http.StatusBadRequest, "That database request wasn't valid.", err)
		return
	}

	if database.Name == nil || *database.Name == "" {
		writeError(w, http.StatusBadRequest, "Database name is required.", nil)
		return
	}
	if database.Engine == nil || *database.Engine == "" {
		writeError(w, http.StatusBadRequest, "A database engine is required.", nil)
		return
	}
	if database.StorageGB == nil {
		writeError(w, http.StatusBadRequest, "A storage amount is required.", nil)
		return
	}

	userId := claims.Subject

	res, err := app.Supabase.UpdateDatabase(databaseId, userId, *database.Name, *database.StorageGB)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Couldn't save the database.", err)
		return
	}

	if err := app.Deploy.CreateDatabase(r.Context(), deploy.Database{
		Namespace: "proj-" + projectId,
		Name:      res.ResourceName,
		Engine:    res.Engine,
		StorageGB: *database.StorageGB,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "We saved your changes, but couldn't apply them to your database. A refresh will show its current status.", err)
		return
	}
}

func (app *Application) DeleteDatabase(w http.ResponseWriter, r *http.Request) {
	projectId := mux.Vars(r)["project_id"]
	if projectId == "" {
		writeError(w, http.StatusBadRequest, "A project ID is required.", nil)
		return
	}

	databaseId := mux.Vars(r)["database_id"]
	if databaseId == "" {
		writeError(w, http.StatusBadRequest, "A database ID is required.", nil)
		return
	}

	claims, ok := clerk.SessionClaimsFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, msgUnauthorized, nil)
		return
	}

	if err := app.Supabase.DeleteDatabase(databaseId, claims.Subject); err != nil {
		writeError(w, http.StatusInternalServerError, "Couldn't delete the database.", err)
		return
	}

	if err := app.Deploy.DeleteDatabase(r.Context(), deploy.Database{
		Namespace: "proj-" + projectId,
		Name:      "db-" + databaseId,
		Engine:    "postgres",
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "Your database was deleted, but we couldn't fully clean it up.", err)
		return
	}

	w.WriteHeader(204)
}

func (app *Application) DeleteDatabases(w http.ResponseWriter, r *http.Request) {
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

	var req BulkDeleteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "That deletion request wasn't valid.", err)
		return
	}

	if len(req.Ids) == 0 {
		writeError(w, http.StatusBadRequest, "At least one database ID is required.", nil)
		return
	}

	deleted := []string{}
	failed := []FailedDelete{}
	for _, databaseId := range req.Ids {
		if err := app.Supabase.DeleteDatabase(databaseId, claims.Subject); err != nil {
			failed = append(failed, FailedDelete{Id: databaseId, Error: "Couldn't delete the database."})
			continue
		}

		if err := app.Deploy.DeleteDatabase(r.Context(), deploy.Database{
			Namespace: "proj-" + projectId,
			Name:      "db-" + databaseId,
			Engine:    "postgres",
		}); err != nil {
			slog.Error("failed to clean up database infrastructure", "database_id", databaseId, "err", err)
		}

		deleted = append(deleted, databaseId)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(BulkDeleteResponse{Deleted: deleted, Failed: failed}); err != nil {
		writeError(w, http.StatusInternalServerError, msgServerError, err)
		return
	}
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
