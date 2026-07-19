package store

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/supabase-community/postgrest-go"
)

type ServicesTable struct {
	Id            string    `json:"id"`
	ProjectId     string    `json:"project_id"`
	Name          string    `json:"name"`
	ResourceName  string    `json:"resource_name"`
	Status        string    `json:"status"`
	URL           string    `json:"url"`
	PublicDomain  string    `json:"public_domain"`
	PrivateDomain string    `json:"private_domain"`
	Port          int32     `json:"port"`
	Image         string    `json:"image"`
	CreatedAt     time.Time `json:"created_at"`
}

type PostgrestError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details"`
	Hint    string `json:"hint"`
}

func (s *SupabaseStore) GetService(id, userId string) (ServicesTable, error) {
	res, _, err := s.cli.From("services").
		Select("*, projects!inner(user_id)", "exact", false).
		Eq("id", id).
		Eq("projects.user_id", userId).
		Order("created_at", &postgrest.OrderOpts{Ascending: false}).
		Single().
		Execute()
	if err != nil {
		return ServicesTable{}, err
	}

	var table ServicesTable
	if err := json.Unmarshal(res, &table); err != nil {
		return ServicesTable{}, err
	}

	return table, nil
}

func (s *SupabaseStore) GetServices(projectId, userId string) ([]ServicesTable, error) {
	res, _, err := s.cli.From("services").
		Select("*, projects!inner(user_id)", "exact", false).
		Eq("project_id", projectId).
		Eq("projects.user_id", userId).
		Order("created_at", &postgrest.OrderOpts{Ascending: false}).
		Execute()
	if err != nil {
		return nil, err
	}

	var table []ServicesTable = []ServicesTable{}
	if err := json.Unmarshal(res, &table); err != nil {
		return nil, err
	}

	return table, nil
}

func (s *SupabaseStore) CreateService(userId, projectId, name, image string, port int32) (ServicesTable, error) {
	result := s.cli.Rpc("create_service", "", map[string]interface{}{
		"p_project_id": projectId,
		"p_user_id":    userId,
		"p_name":       name,
		"p_image":      image,
		"p_port":       port,
	})

	var res ServicesTable
	if err := json.Unmarshal([]byte(result), &res); err != nil {
		return ServicesTable{}, nil
	}

	var pgErr PostgrestError
	if err := json.Unmarshal([]byte(result), &pgErr); err == nil && pgErr.Message != "" {
		return ServicesTable{}, fmt.Errorf("create_service failed: %s", pgErr.Message)
	}

	return res, nil
}

func (s *SupabaseStore) UpdateService(id, userId, name, image string, port int32) (ServicesTable, error) {
	result := s.cli.Rpc("update_service", "", map[string]interface{}{
		"p_id":      id,
		"p_user_id": userId,
		"p_name":    name,
		"p_image":   image,
		"p_port":    port,
	})

	var res ServicesTable
	if err := json.Unmarshal([]byte(result), &res); err != nil {
		return ServicesTable{}, nil
	}

	var pgErr PostgrestError
	if err := json.Unmarshal([]byte(result), &pgErr); err == nil && pgErr.Message != "" {
		return ServicesTable{}, fmt.Errorf("create_service failed: %s", pgErr.Message)
	}

	return res, nil
}

func (s *SupabaseStore) DeleteService(id, userId string) error {
	result := s.cli.Rpc("delete_service", "", map[string]interface{}{
		"p_id":      id,
		"p_user_id": userId,
	})

	var pgErr PostgrestError
	if err := json.Unmarshal([]byte(result), &pgErr); err == nil && pgErr.Message != "" {
		return fmt.Errorf("create_service failed: %s", pgErr.Message)
	}

	return nil
}
