package store

import (
	"encoding/json"
	"fmt"
)

func PreviewCopyName(base string, prNumber int) string {
	return fmt.Sprintf("%s-pr-%d", base, prNumber)
}

func (s *SupabaseStore) GetProjectOwnerId(projectId string) (string, error) {
	res, _, err := s.cli.From("projects").
		Select("user_id", "exact", false).
		Eq("id", projectId).
		Single().
		Execute()
	if err != nil {
		return "", err
	}
	var out struct {
		UserId string `json:"user_id"`
	}
	if err := json.Unmarshal(res, &out); err != nil {
		return "", err
	}
	if out.UserId == "" {
		return "", fmt.Errorf("project %s has no owner", projectId)
	}
	return out.UserId, nil
}

func (s *SupabaseStore) GetPreviewCopyIDs(projectId string) (map[string]bool, error) {
	_, membership, err := s.GetPreviewEnvironmentsForProject(projectId)
	if err != nil {
		return nil, err
	}
	ids := make(map[string]bool, len(membership))
	for _, m := range membership {
		ids[m.SourceServiceId] = true
	}
	return ids, nil
}

func (s *SupabaseStore) IsPreviewCopy(projectId, sourceID string) bool {
	ids, err := s.GetPreviewCopyIDs(projectId)
	if err != nil {
		return false
	}
	return ids[sourceID]
}

func (s *SupabaseStore) EnsureGithubServiceCopy(ownerId string, src GithubServicesTable, prNumber int) (GithubServicesTable, error) {
	name := PreviewCopyName(src.Name, prNumber)
	if all, err := s.GetGithubServicesByProjectId(src.ProjectId); err == nil {
		for _, g := range all {
			if g.Name == name {
				return g, nil
			}
		}
	}
	created, err := s.CreateGithubService(ownerId, src.ProjectId, name, src.GithubConnectionId, src.RepoName, src.RepoId, src.RootDir, src.Branch, nil, src.Port)
	if err != nil {
		if all, aerr := s.GetGithubServicesByProjectId(src.ProjectId); aerr == nil {
			for _, g := range all {
				if g.Name == name {
					return g, nil
				}
			}
		}
		return GithubServicesTable{}, err
	}
	return created, nil
}

func (s *SupabaseStore) EnsureServiceCopy(ownerId string, src ServicesTable, prNumber int) (ServicesTable, error) {
	name := PreviewCopyName(src.Name, prNumber)
	if all, err := s.GetServicesByProjectId(src.ProjectId); err == nil {
		for _, v := range all {
			if v.Name == name {
				// Keep the copy tracking prod's image (port stays
				// preview-owned via the preview edit endpoint).
				if v.Image != src.Image {
					if updated, uerr := s.UpdateServiceImageById(v.Id, src.Image); uerr == nil {
						return updated, nil
					}
				}
				return v, nil
			}
		}
	}
	created, err := s.CreateService(ownerId, src.ProjectId, name, src.Image, nil, src.Port)
	if err != nil {
		if all, aerr := s.GetServicesByProjectId(src.ProjectId); aerr == nil {
			for _, v := range all {
				if v.Name == name {
					return v, nil
				}
			}
		}
		return ServicesTable{}, err
	}
	return created, nil
}

func (s *SupabaseStore) EnsureDatabaseCopy(ownerId string, src DatabasesTable, prNumber int) (DatabasesTable, error) {
	name := PreviewCopyName(src.Name, prNumber)
	if all, err := s.GetDatabasesByProjectId(src.ProjectId); err == nil {
		for _, d := range all {
			if d.Name == name {
				return d, nil
			}
		}
	}
	created, err := s.CreateDatabase(ownerId, src.ProjectId, name, src.Engine, src.Port, src.StorageGB)
	if err != nil {
		if all, aerr := s.GetDatabasesByProjectId(src.ProjectId); aerr == nil {
			for _, d := range all {
				if d.Name == name {
					return d, nil
				}
			}
		}
		return DatabasesTable{}, err
	}
	return created, nil
}

func (s *SupabaseStore) UpdateServiceStatusById(id, status string) (ServicesTable, error) {
	res, _, err := s.cli.From("services").
		Update(map[string]interface{}{"status": status}, "", "").
		Eq("id", id).
		Execute()
	if err != nil {
		return ServicesTable{}, err
	}
	var arr []ServicesTable
	if err := json.Unmarshal(res, &arr); err == nil && len(arr) == 1 {
		return arr[0], nil
	}
	var single ServicesTable
	if err := json.Unmarshal(res, &single); err != nil {
		return ServicesTable{}, err
	}
	return single, nil
}

func (s *SupabaseStore) UpdateServiceImageById(id, image string) (ServicesTable, error) {
	res, _, err := s.cli.From("services").
		Update(map[string]interface{}{"image": image}, "", "").
		Eq("id", id).
		Execute()
	if err != nil {
		return ServicesTable{}, err
	}
	var arr []ServicesTable
	if err := json.Unmarshal(res, &arr); err == nil && len(arr) == 1 {
		return arr[0], nil
	}
	var single ServicesTable
	if err := json.Unmarshal(res, &single); err != nil {
		return ServicesTable{}, err
	}
	return single, nil
}

func (s *SupabaseStore) UpdateServicePortById(id string, port int32) (ServicesTable, error) {
	res, _, err := s.cli.From("services").
		Update(map[string]interface{}{"port": port}, "", "").
		Eq("id", id).
		Execute()
	if err != nil {
		return ServicesTable{}, err
	}
	var arr []ServicesTable
	if err := json.Unmarshal(res, &arr); err == nil && len(arr) == 1 {
		return arr[0], nil
	}
	var single ServicesTable
	if err := json.Unmarshal(res, &single); err != nil {
		return ServicesTable{}, err
	}
	return single, nil
}

func (s *SupabaseStore) UpdateGithubServicePortById(id string, port int32) (GithubServicesTable, error) {
	res, _, err := s.cli.From("github_services").
		Update(map[string]interface{}{"port": port}, "", "").
		Eq("id", id).
		Execute()
	if err != nil {
		return GithubServicesTable{}, err
	}
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

func (s *SupabaseStore) DeletePreviewCopy(ownerId, sourceType, id string) error {
	switch sourceType {
	case "github_service":
		return s.DeleteGithubService(id, ownerId)
	case "service":
		return s.DeleteService(id, ownerId)
	case "database":
		return s.DeleteDatabase(id, ownerId)
	default:
		return errUnknownPreviewSourceType(sourceType)
	}
}
