package store

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/supabase-community/postgrest-go"
)

type PreviewEnvironmentService struct {
	Id                   string    `json:"id,omitempty"`
	PreviewEnvironmentId string    `json:"preview_environment_id"`
	SourceType           string    `json:"type"`
	SourceServiceId      string    `json:"source_service_id"`
	CreatedAt            time.Time `json:"created_at,omitempty"`
}

type PreviewEnvironmentSource struct {
	Type         string
	Id           string
	ProjectId    string
	Name         string
	ResourceName string
	Port         int32
	RepoId       int64
	RepoName     string
	Branch       string
	Engine       string
	Status       string
}

type PreviewEnvironmentView struct {
	Environment PreviewEnvironment
	Service     PreviewEnvironmentService
}

func PreviewCopyName(base string, prNumber int) string {
	return fmt.Sprintf("%s-pr-%d", base, prNumber)
}

func (s *SupabaseStore) GetPreviewEnvironmentServices(envID string) ([]PreviewEnvironmentService, error) {
	res, _, err := s.cli.From("preview_environment_services").
		Select("*", "exact", false).
		Eq("preview_environment_id", envID).
		Order("created_at", &postgrest.OrderOpts{Ascending: false}).
		Execute()
	if err != nil {
		return nil, err
	}
	out := []PreviewEnvironmentService{}
	if err := json.Unmarshal(res, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *SupabaseStore) GetPreviewEnvironmentService(envID, sourceType, sourceServiceID string) (PreviewEnvironmentService, error) {
	res, _, err := s.cli.From("preview_environment_services").
		Select("*", "exact", false).
		Eq("preview_environment_id", envID).
		Eq("type", sourceType).
		Eq("source_service_id", sourceServiceID).
		Single().
		Execute()
	if err != nil {
		return PreviewEnvironmentService{}, err
	}
	var out PreviewEnvironmentService
	if err := json.Unmarshal(res, &out); err != nil {
		return PreviewEnvironmentService{}, err
	}
	return out, nil
}

func (s *SupabaseStore) EnsurePreviewEnvironmentService(envID, sourceType, sourceServiceID string) (PreviewEnvironmentService, error) {
	if existing, err := s.GetPreviewEnvironmentService(envID, sourceType, sourceServiceID); err == nil && existing.Id != "" {
		return existing, nil
	}
	res, _, err := s.cli.From("preview_environment_services").Insert(map[string]interface{}{
		"preview_environment_id": envID,
		"type":                   sourceType,
		"source_service_id":      sourceServiceID,
		"created_at":             time.Now().UTC().Format(time.RFC3339),
	}, false, "", "", "").Single().Execute()
	if err != nil {
		if existing, aerr := s.GetPreviewEnvironmentService(envID, sourceType, sourceServiceID); aerr == nil && existing.Id != "" {
			return existing, nil
		}
		return PreviewEnvironmentService{}, err
	}
	var out PreviewEnvironmentService
	if err := json.Unmarshal(res, &out); err != nil {
		return PreviewEnvironmentService{}, err
	}
	return out, nil
}

func (s *SupabaseStore) GetPreviewEnvironmentSource(sourceType, sourceServiceID string) (PreviewEnvironmentSource, error) {
	switch sourceType {
	case "github_service":
		g, err := s.GetGithubServiceById(sourceServiceID)
		if err != nil {
			return PreviewEnvironmentSource{}, err
		}
		return PreviewEnvironmentSource{
			Type:         "github_service",
			Id:           g.Id,
			ProjectId:    g.ProjectId,
			Name:         g.Name,
			ResourceName: g.ResourceName,
			Port:         g.Port,
			RepoId:       g.RepoId,
			RepoName:     g.RepoName,
			Branch:       g.Branch,
			Status:       g.Status,
		}, nil
	case "service":
		v, err := s.GetServiceById(sourceServiceID)
		if err != nil {
			return PreviewEnvironmentSource{}, err
		}
		return PreviewEnvironmentSource{
			Type:         "service",
			Id:           v.Id,
			ProjectId:    v.ProjectId,
			Name:         v.Name,
			ResourceName: v.ResourceName,
			Port:         v.Port,
			Status:       v.Status,
		}, nil
	case "database":
		d, err := s.GetDatabaseById(sourceServiceID)
		if err != nil {
			return PreviewEnvironmentSource{}, err
		}
		return PreviewEnvironmentSource{
			Type:         "database",
			Id:           d.Id,
			ProjectId:    d.ProjectId,
			Name:         d.Name,
			ResourceName: d.ResourceName,
			Port:         d.Port,
			Engine:       d.Engine,
		}, nil
	default:
		return PreviewEnvironmentSource{}, errUnknownPreviewSourceType(sourceType)
	}
}

func (s *SupabaseStore) GetPreviewEnvironmentViewsBySource(sourceServiceID string) ([]PreviewEnvironmentView, error) {
	res, _, err := s.cli.From("preview_environment_services").
		Select("*", "exact", false).
		Eq("source_service_id", sourceServiceID).
		Order("created_at", &postgrest.OrderOpts{Ascending: false}).
		Execute()
	if err != nil {
		return nil, err
	}
	var children []PreviewEnvironmentService
	if err := json.Unmarshal(res, &children); err != nil {
		return nil, err
	}
	out := make([]PreviewEnvironmentView, 0, len(children))
	for _, c := range children {
		env, err := s.GetPreviewEnvironmentByID(c.PreviewEnvironmentId)
		if err != nil {
			continue
		}
		out = append(out, PreviewEnvironmentView{Environment: env, Service: c})
	}
	return out, nil
}

func (s *SupabaseStore) GetPreviewEnvironmentViewsByRepoPR(repoId int64, prNumber int) ([]PreviewEnvironmentView, error) {
	services, err := s.GetGithubServicesByRepoId(repoId)
	if err != nil {
		return nil, err
	}
	out := []PreviewEnvironmentView{}
	for _, svc := range services {
		views, err := s.GetPreviewEnvironmentViewsBySource(svc.Id)
		if err != nil {
			continue
		}
		for _, v := range views {
			if v.Environment.Pr != prNumber {
				continue
			}
			out = append(out, v)
		}
	}
	return out, nil
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

func (s *SupabaseStore) DeletePreviewEnvironmentServicesBySource(sourceServiceID string) error {
	_, _, err := s.cli.From("preview_environment_services").Delete("", "").Eq("source_service_id", sourceServiceID).Execute()
	return err
}
