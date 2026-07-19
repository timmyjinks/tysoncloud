package store

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/supabase-community/postgrest-go"
)

type DatabasesTable struct {
	Id            string    `json:"id,omitempty"`
	ProjectId     string    `json:"project_id,omitempty"`
	Name          string    `json:"name,omitempty"`
	Engine        string    `json:"engine,omitempty"`
	ResourceName  string    `json:"resource_name,omitempty"`
	PrivateDomain string    `json:"private_domain,omitempty"`
	Port          int32     `json:"port,omitempty"`
	StorageGB     int32     `json:"storage,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

func (s *SupabaseStore) GetDatabase(projectId, userId string) (DatabasesTable, error) {
	res, _, err := s.cli.From("databases").Select("*", "exact", false).Eq("project_id", projectId).Eq("user_id", userId).Order("created_at", &postgrest.OrderOpts{Ascending: false}).Single().Execute()
	if err != nil {
		return DatabasesTable{}, err
	}

	var table []DatabasesTable = []DatabasesTable{}
	if err := json.Unmarshal(res, &table); err != nil {
		return DatabasesTable{}, err
	}

	return table[0], nil
}

func (s *SupabaseStore) GetDatabases(projectId, userId string) ([]DatabasesTable, error) {
	res, _, err := s.cli.From("databases").Select("*", "exact", false).Eq("project_id", projectId).Eq("user_id", userId).Order("created_at", &postgrest.OrderOpts{Ascending: false}).Execute()
	if err != nil {
		return nil, err
	}

	var table []DatabasesTable = []DatabasesTable{}
	if err := json.Unmarshal(res, &table); err != nil {
		return nil, err
	}

	return table, nil
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
		return DatabasesTable{}, nil
	}

	var pgErr PostgrestError
	if err := json.Unmarshal([]byte(result), &pgErr); err == nil && pgErr.Message != "" {
		return DatabasesTable{}, fmt.Errorf("create_service failed: %s", pgErr.Message)
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
		return DatabasesTable{}, nil
	}

	var pgErr PostgrestError
	if err := json.Unmarshal([]byte(result), &pgErr); err == nil && pgErr.Message != "" {
		return DatabasesTable{}, fmt.Errorf("update_database failed: %s", pgErr.Message)
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
		return fmt.Errorf("delete_service failed: %s", pgErr.Message)
	}

	return nil
}
