package store

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/supabase-community/postgrest-go"
)

type GithubServicesTable struct {
	Id                 string    `json:"id"`
	ProjectId          string    `json:"project_id"`
	GithubConnectionId string    `json:"github_connection_id"`
	RepoId             int64     `json:"repo_id"`
	RepoName           string    `json:"repo_name"`
	Name               string    `json:"name"`
	ResourceName       string    `json:"resource_name"`
	Status             string    `json:"status"`
	PublicDomain       string    `json:"public_domain"`
	PrivateDomain      string    `json:"private_domain"`
	Port               int32     `json:"port"`
	RootDir            string    `json:"root_dir"`
	CreatedAt          time.Time `json:"created_at"`
}

func (s *SupabaseStore) GetGithubService(id, userId string) (GithubServicesTable, error) {
	res, _, err := s.cli.From("github_services").
		Select("*, projects!inner(user_id)", "exact", false).
		Eq("id", id).
		Eq("projects.user_id", userId).
		Order("created_at", &postgrest.OrderOpts{Ascending: false}).
		Single().
		Execute()
	if err != nil {
		return GithubServicesTable{}, err
	}

	var table GithubServicesTable
	if err := json.Unmarshal(res, &table); err != nil {
		return GithubServicesTable{}, err
	}

	return table, nil
}

func (s *SupabaseStore) GetGithubServices(projectId, userId string) ([]GithubServicesTable, error) {
	res, _, err := s.cli.From("github_services").
		Select("*, projects!inner(user_id)", "exact", false).
		Eq("project_id", projectId).
		Eq("projects.user_id", userId).
		Order("created_at", &postgrest.OrderOpts{Ascending: false}).
		Execute()
	if err != nil {
		return nil, err
	}

	var table []GithubServicesTable = []GithubServicesTable{}
	if err := json.Unmarshal(res, &table); err != nil {
		return nil, err
	}

	return table, nil
}

func (s *SupabaseStore) CreateGithubService(userId, projectId, name, githubConnectionId, repo string, repoId int64, rootDir string, domain *string, port int32) (GithubServicesTable, error) {
	result := s.cli.Rpc("create_github_service", "", map[string]interface{}{
		"p_project_id":           projectId,
		"p_github_connection_id": githubConnectionId,
		"p_repo_id":              repoId,
		"p_user_id":              userId,
		"p_repo_name":            repo,
		"p_name":                 name,
		"p_root_dir":             rootDir,
		"p_domain":               domain,
		"p_port":                 port,
	})

	var pgErr PostgrestError
	if err := json.Unmarshal([]byte(result), &pgErr); err == nil && pgErr.Message != "" {
		return GithubServicesTable{}, rpcError("create_github_service", pgErr)
	}

	var res GithubServicesTable
	if err := json.Unmarshal([]byte(result), &res); err != nil {
		return GithubServicesTable{}, err
	}

	return res, nil
}

func (s *SupabaseStore) UpdateGithubService(id, userId, name string, domain *string, port int32) (GithubServicesTable, error) {
	result := s.cli.Rpc("update_github_service", "", map[string]interface{}{
		"p_id":            id,
		"p_user_id":       userId,
		"p_name":          name,
		"p_port":          port,
		"p_public_domain": domain,
	})

	var pgErr PostgrestError
	if err := json.Unmarshal([]byte(result), &pgErr); err == nil && pgErr.Message != "" {
		return GithubServicesTable{}, rpcError("update_github_service", pgErr)
	}

	var res GithubServicesTable
	if err := json.Unmarshal([]byte(result), &res); err != nil {
		return GithubServicesTable{}, err
	}

	return res, nil
}

func (s *SupabaseStore) GetGithubServicesByRepoId(repoId int64) ([]GithubServicesTable, error) {
	res, _, err := s.cli.From("github_services").
		Select("*", "exact", false).
		Eq("repo_id", strconv.FormatInt(repoId, 10)).
		Execute()
	if err != nil {
		return nil, err
	}

	var table []GithubServicesTable = []GithubServicesTable{}
	if err := json.Unmarshal(res, &table); err != nil {
		return nil, err
	}

	return table, nil
}

func (s *SupabaseStore) UpdateGithubServiceStatus(id, userId, status string) (GithubServicesTable, error) {
	result := s.cli.Rpc("update_github_service_status", "", map[string]interface{}{
		"p_id":      id,
		"p_user_id": userId,
		"p_status":  status,
	})

	var pgErr PostgrestError
	if err := json.Unmarshal([]byte(result), &pgErr); err == nil && pgErr.Message != "" {
		// Graceful fallback: if the RPC is missing (schema cache), try the by-id variant
		// so builds don't 500 on missing function `update_github_service_status`.
		if isMissingFunctionError(pgErr) {
			return s.UpdateGithubServiceStatusById(id, status)
		}
		return GithubServicesTable{}, rpcError("update_github_service_status", pgErr)
	}

	var res GithubServicesTable
	if err := json.Unmarshal([]byte(result), &res); err != nil {
		return GithubServicesTable{}, err
	}

	return res, nil
}

func isMissingFunctionError(pgErr PostgrestError) bool {
	msg := pgErr.Message
	return contains(msg, "Could not find the function") || contains(msg, "schema cache")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (func() bool {
		for i := 0; i <= len(s)-len(substr); i++ {
			if s[i:i+len(substr)] == substr {
				return true
			}
		}
		return false
	})()
}

func (s *SupabaseStore) UpdateGithubServiceStatusById(id, status string) (GithubServicesTable, error) {
	result := s.cli.Rpc("update_github_service_status_by_id", "", map[string]interface{}{
		"p_id":     id,
		"p_status": status,
	})

	var pgErr PostgrestError
	if err := json.Unmarshal([]byte(result), &pgErr); err == nil && pgErr.Message != "" {
		if isMissingFunctionError(pgErr) {
			// Fallback to direct table update when RPC is missing (schema cache).
			return s.updateGithubServiceStatusDirect(id, status)
		}
		return GithubServicesTable{}, rpcError("update_github_service_status_by_id", pgErr)
	}

	var res GithubServicesTable
	if err := json.Unmarshal([]byte(result), &res); err != nil {
		return GithubServicesTable{}, err
	}

	return res, nil
}

func (s *SupabaseStore) updateGithubServiceStatusDirect(id, status string) (GithubServicesTable, error) {
	res, _, err := s.cli.From("github_services").
		Update(map[string]interface{}{"status": status}, "", "").
		Eq("id", id).
		Execute()
	if err != nil {
		return GithubServicesTable{}, err
	}
	// Supabase may return array with one element
	var arr []GithubServicesTable
	if err := json.Unmarshal(res, &arr); err == nil && len(arr) == 1 {
		return arr[0], nil
	}
	var single GithubServicesTable
	if err := json.Unmarshal(res, &single); err != nil {
		return GithubServicesTable{}, err
	}
	return single, nil
}

func (s *SupabaseStore) DeleteGithubService(id, userId string) error {
	result := s.cli.Rpc("delete_github_service", "", map[string]interface{}{
		"p_id":      id,
		"p_user_id": userId,
	})

	var pgErr PostgrestError
	if err := json.Unmarshal([]byte(result), &pgErr); err == nil && pgErr.Message != "" {
		return rpcError("delete_github_service", pgErr)
	}

	return nil
}

func (s *SupabaseStore) GetGithubServiceById(id string) (GithubServicesTable, error) {
	res, _, err := s.cli.From("github_services").
		Select("*", "exact", false).
		Eq("id", id).
		Single().
		Execute()
	if err != nil {
		return GithubServicesTable{}, err
	}

	var table GithubServicesTable
	if err := json.Unmarshal(res, &table); err != nil {
		return GithubServicesTable{}, err
	}

	return table, nil
}
