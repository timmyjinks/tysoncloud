package store

import (
	"encoding/json"
	"time"

	"github.com/supabase-community/postgrest-go"
)

type DatabasesTable struct {
	Id             string    `json:"id"`
	ProjectId      string    `json:"project_id"`
	Name           string    `json:"name"`
	Engine         string    `json:"engine"`
	ResourceName   string    `json:"resource_name"`
	InternalDomain string    `json:"internal_domain"`
	Port           int32     `json:"port"`
	StorageGB      int32     `json:"storage"`
	Production     *bool     `json:"production,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

func (s *SupabaseStore) GetDatabase(id, userId string) (DatabasesTable, error) {
	res, _, err := s.cli.From("databases").
		Select("*, projects!inner(user_id)", "exact", false).
		Eq("id", id).
		Eq("projects.user_id", userId).
		Order("created_at", &postgrest.OrderOpts{Ascending: false}).
		Single().
		Execute()
	if err != nil {
		return DatabasesTable{}, err
	}

	var table DatabasesTable
	if err := json.Unmarshal(res, &table); err != nil {
		return DatabasesTable{}, err
	}

	return table, nil
}

func (s *SupabaseStore) GetDatabaseById(id string) (DatabasesTable, error) {
	res, _, err := s.cli.From("databases").
		Select("*", "exact", false).
		Eq("id", id).
		Single().
		Execute()
	if err != nil {
		return DatabasesTable{}, err
	}

	var table DatabasesTable
	if err := json.Unmarshal(res, &table); err != nil {
		return DatabasesTable{}, err
	}

	return table, nil
}

func (s *SupabaseStore) GetDatabases(projectId, userId string) ([]DatabasesTable, error) {
	res, _, err := s.cli.From("databases").
		Select("*, projects!inner(user_id)", "exact", false).
		Eq("project_id", projectId).
		Eq("projects.user_id", userId).
		Order("created_at", &postgrest.OrderOpts{Ascending: false}).
		Execute()

	if err != nil {
		return nil, err
	}

	var table []DatabasesTable = []DatabasesTable{}
	if err := json.Unmarshal(res, &table); err != nil {
		return nil, err
	}

	return table, nil
}

func (s *SupabaseStore) GetDatabasesByProjectId(projectId string) ([]DatabasesTable, error) {
	res, _, err := s.cli.From("databases").
		Select("*", "exact", false).
		Eq("project_id", projectId).
		Order("created_at", &postgrest.OrderOpts{Ascending: false}).
		Execute()

	if err != nil {
		return nil, err
	}

	var table []DatabasesTable = []DatabasesTable{}
	if err := json.Unmarshal(res, &table); err != nil {
		return nil, err
	}

	return table, nil
}

func (s *SupabaseStore) GetProductionDatabasesByProjectId(projectId string) ([]DatabasesTable, error) {
	all, err := s.GetDatabasesByProjectId(projectId)
	if err != nil {
		return nil, err
	}
	out := make([]DatabasesTable, 0, len(all))
	for _, db := range all {
		if IsProductionRow(db.Production) {
			out = append(out, db)
		}
	}
	return out, nil
}

func (s *SupabaseStore) CreateDatabase(userId, projectId, name, engine string, port, storageGB int32) (DatabasesTable, error) {
	result := s.cli.Rpc("create_database", "", map[string]interface{}{
		"p_project_id": projectId,
		"p_user_id":    userId,
		"p_name":       name,
		"p_engine":     engine,
		"p_port":       port,
		"p_storage_gb": storageGB,
	})

	var res DatabasesTable
	if err := json.Unmarshal([]byte(result), &res); err != nil {
		return DatabasesTable{}, err
	}

	var pgErr PostgrestError
	if err := json.Unmarshal([]byte(result), &pgErr); err == nil && pgErr.Message != "" {
		return DatabasesTable{}, rpcError("create_database", pgErr)
	}

	return res, nil
}

func (s *SupabaseStore) UpdateDatabase(id, userId, name string, storageGB int32) (DatabasesTable, error) {
	result := s.cli.Rpc("update_database", "", map[string]interface{}{
		"p_id":         id,
		"p_user_id":    userId,
		"p_name":       name,
		"p_storage_gb": storageGB,
	})

	var res DatabasesTable
	if err := json.Unmarshal([]byte(result), &res); err != nil {
		return DatabasesTable{}, err
	}

	var pgErr PostgrestError
	if err := json.Unmarshal([]byte(result), &pgErr); err == nil && pgErr.Message != "" {
		return DatabasesTable{}, rpcError("update_database", pgErr)
	}

	return res, nil
}

func (s *SupabaseStore) DeleteDatabase(id, userId string) error {
	result := s.cli.Rpc("delete_database", "", map[string]interface{}{
		"p_id":      id,
		"p_user_id": userId,
	})

	var pgErr PostgrestError
	if err := json.Unmarshal([]byte(result), &pgErr); err == nil && pgErr.Message != "" {
		return rpcError("delete_database", pgErr)
	}

	return nil
}
