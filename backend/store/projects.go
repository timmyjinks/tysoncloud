package store

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/supabase-community/postgrest-go"
)

type ProjectsTable struct {
	Id        string    `json:"id"`
	UserId    string    `json:"user_id"`
	Namespace string    `json:"namespace"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *SupabaseStore) GetProject(id string, userId string) (ProjectsTable, error) {
	res, _, err := s.cli.From("projects").Select("*", "exact", false).Order("created_at", &postgrest.OrderOpts{Ascending: false}).Eq("id", id).Eq("user_id", userId).Single().Execute()
	if err != nil {
		return ProjectsTable{}, err
	}

	var table ProjectsTable
	if err := json.Unmarshal(res, &table); err != nil {
		return ProjectsTable{}, err
	}

	return table, nil
}

func (s *SupabaseStore) GetProjects(userId string) ([]ProjectsTable, error) {
	res, _, err := s.cli.From("projects").Select("*", "exact", false).Order("created_at", &postgrest.OrderOpts{Ascending: false}).Eq("user_id", userId).Execute()
	if err != nil {
		return nil, err
	}

	var table []ProjectsTable = []ProjectsTable{}
	if err := json.Unmarshal(res, &table); err != nil {
		return nil, err
	}

	return table, nil
}

func (s *SupabaseStore) CreateProject(userId, name string) (ProjectsTable, error) {
	bytes, _, err := s.cli.From("projects").Insert(struct {
		UserId string `json:"user_id"`
		Name   string `json:"name"`
	}{
		UserId: userId,
		Name:   name,
	}, false, "", "", "").Execute()
	if err != nil {
		return ProjectsTable{}, err
	}

	var res []ProjectsTable = []ProjectsTable{}
	if err := json.Unmarshal(bytes, &res); err != nil {
		return ProjectsTable{}, err
	}

	if len(res) == 0 {
		return ProjectsTable{}, errors.New("error creating project")
	}

	return res[0], nil
}

func (s *SupabaseStore) UpdateProject(id, userId, name string) error {
	_, _, err := s.cli.From("projects").Update(struct {
		Id     string `json:"id"`
		UserId string `json:"user_id"`
		Name   string `json:"name"`
	}{
		Id:     id,
		UserId: userId,
		Name:   name,
	}, "", "").Eq("user_id", userId).Eq("id", id).Execute()
	if err != nil {
		return err
	}

	return nil
}

func (s *SupabaseStore) DeleteProject(userId, id string) error {
	_, _, err := s.cli.From("projects").Delete("", "").Eq("id", id).Eq("user_id", userId).Execute()
	if err != nil {
		return err
	}
	return nil
}
