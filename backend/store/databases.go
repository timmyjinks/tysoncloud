package store

import (
	"encoding/json"
	"time"

	"github.com/supabase-community/postgrest-go"
)

type DatabasesTable struct {
	ID            string    `json:"id,omitempty"`
	ProjectId     string    `json:"project_id,omitempty"`
	Name          string    `json:"name,omitempty"`
	Engine        string    `json:"engine,omitempty"`
	ResourceName  string    `json:"resource_name,omitempty"`
	PrivateDomain string    `json:"private_domain,omitempty"`
	Port          int32     `json:"port,omitempty"`
	StorageGB     int32     `json:"storage,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

func (s *SupabaseStore) GetDatabasesByProjectID(project_id string) ([]DatabasesTable, error) {
	res, _, err := s.cli.From("databases").Select("*", "exact", false).Eq("project_id", project_id).Order("created_at", &postgrest.OrderOpts{Ascending: false}).Execute()
	if err != nil {
		return nil, err
	}

	var table []DatabasesTable = []DatabasesTable{}
	if err := json.Unmarshal(res, &table); err != nil {
		return nil, err
	}

	return table, nil
}

func (s *SupabaseStore) GetDatabases() ([]DatabasesTable, error) {
	res, _, err := s.cli.From("databases").Select("*", "exact", false).Order("created_at", &postgrest.OrderOpts{Ascending: false}).Execute()
	if err != nil {
		return nil, err
	}

	var table []DatabasesTable = []DatabasesTable{}
	if err := json.Unmarshal(res, &table); err != nil {
		return nil, err
	}

	return table, nil
}
