package store

import (
	"encoding/json"
	"time"

	"github.com/supabase-community/postgrest-go"
)

type ServicesTable struct {
	ID            string    `json:"id,omitempty"`
	ProjectId     string    `json:"project_id,omitempty"`
	Name          string    `json:"name,omitempty"`
	ResourceName  string    `json:"resource_name,omitempty"`
	Status        string    `json:"status,omitempty"`
	URL           string    `json:"url,omitempty"`
	PublicDomain  string    `json:"public_domain,omitempty"`
	PrivateDomain string    `json:"private_domain,omitempty"`
	Port          int32     `json:"port,omitempty"`
	Image         string    `json:"image,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

func (s *SupabaseStore) GetServicesByID(project_id string) ([]ServicesTable, error) {
	res, _, err := s.cli.From("services").Select("*", "exact", false).Eq("project_id", project_id).Order("created_at", &postgrest.OrderOpts{Ascending: false}).Execute()
	if err != nil {
		return nil, err
	}

	var table []ServicesTable = []ServicesTable{}
	if err := json.Unmarshal(res, &table); err != nil {
		return nil, err
	}

	return table, nil
}

func (s *SupabaseStore) GetServices() ([]ServicesTable, error) {
	res, _, err := s.cli.From("services").Select("*", "exact", false).Order("created_at", &postgrest.OrderOpts{Ascending: false}).Execute()
	if err != nil {
		return nil, err
	}

	var table []ServicesTable = []ServicesTable{}
	if err := json.Unmarshal(res, &table); err != nil {
		return nil, err
	}

	return table, nil
}

func (s *SupabaseStore) CreateService(projectId, name, image, status string) error {
	_, _, err := s.cli.From("services").Insert(struct {
		ID        string `json:"id,omitempty"`
		ProjectId string `json:"project_id,omitempty"`
		Name      string `json:"name,omitempty"`
		Status    string `json:"status,omitempty"`
		Image     string `json:"image,omitempty"`
	}{
		ProjectId: projectId,
		Name:      name,
		Status:    status,
		Image:     image,
	}, false, "", "", "").Execute()
	if err != nil {
		return err
	}

	return nil
}
