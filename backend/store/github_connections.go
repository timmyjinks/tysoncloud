package store

import (
	"encoding/json"
	"time"
)

type GithubConnectionsTable struct {
	Id             string    `json:"id"`
	UserId         string    `json:"user_id"`
	InstallationId string    `json:"installation_id"`
	CreatedAt      time.Time `json:"created_at"`
}

func (s *SupabaseStore) GetGithubConnection(userId string) (GithubConnectionsTable, error) {
	res, _, err := s.cli.From("github_connections").
		Select("*", "exact", false).
		Eq("user_id", userId).
		Single().
		Execute()
	if err != nil {
		return GithubConnectionsTable{}, err
	}

	var table GithubConnectionsTable
	if err := json.Unmarshal(res, &table); err != nil {
		return GithubConnectionsTable{}, err
	}

	return table, nil
}

func (s *SupabaseStore) GetGithubConnectionByInstallationId(installationId string) (GithubConnectionsTable, error) {
	res, _, err := s.cli.From("github_connections").
		Select("*", "exact", false).
		Eq("installation_id", installationId).
		Single().
		Execute()
	if err != nil {
		return GithubConnectionsTable{}, err
	}

	var table GithubConnectionsTable
	if err := json.Unmarshal(res, &table); err != nil {
		return GithubConnectionsTable{}, err
	}

	return table, nil
}

func (s *SupabaseStore) CreateGithubConnection(userId, installationId string) error {
	_, _, err := s.cli.From("github_connections").Insert(GithubConnectionsTable{
		UserId:         userId,
		InstallationId: installationId,
	}, false, "", "", "").Execute()
	if err != nil {
		return err
	}
	return nil
}

func (s *SupabaseStore) DeleteGithubConnection(id, userId string) error {
	_, _, err := s.cli.From("github_connections").Delete("", "").
		Eq("id", id).
		Eq("user_id", userId).
		Execute()
	if err != nil {
		return err
	}
	return nil
}

func (s *SupabaseStore) DeleteGithubConnectionByInstallationId(installationId string) error {
	_, _, err := s.cli.From("github_connections").Delete("", "").
		Eq("installation_id", installationId).
		Execute()
	if err != nil {
		return err
	}
	return nil
}
