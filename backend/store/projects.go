package store

import (
	"encoding/json"
	"time"

	"github.com/supabase-community/postgrest-go"
)

type ProjectsTable struct {
	ID        string    `json:"id,omitempty"`
	UserId    string    `json:"user_id"`
	Namespace string    `json:"namespace,omitempty"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *SupabaseStore) GetProjects() ([]ProjectsTable, error) {
	res, _, err := s.cli.From("projects").Select("*", "exact", false).Order("created_at", &postgrest.OrderOpts{Ascending: false}).Execute()
	if err != nil {
		return nil, err
	}

	var table []ProjectsTable = []ProjectsTable{}
	if err := json.Unmarshal(res, &table); err != nil {
		return nil, err
	}

	return table, nil
}

func (s *SupabaseStore) CreateProject(id, userId, name string) error {
	_, _, err := s.cli.From("projects").Insert(struct {
		ID     string `json:"id,omitempty"`
		UserId string `json:"user_id"`
		Name   string `json:"name"`
	}{
		ID:     id,
		UserId: userId,
		Name:   name,
	}, false, "", "", "").Execute()
	if err != nil {
		return err
	}

	return nil
}
