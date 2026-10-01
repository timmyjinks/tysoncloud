package store

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/supabase-community/postgrest-go"
)

type PreviewEnvironment struct {
	Id        string    `json:"id,omitempty"`
	ProjectId string    `json:"project_id"`
	Name      string    `json:"name"`
	CommentID int64     `json:"comment_id,omitempty"`
	Pr        int       `json:"pr,omitempty"`
	Namespace string    `json:"namespace"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

type PreviewEnvironmentService struct {
	Id                   int64     `json:"id,omitempty"`
	PreviewEnvironmentId string    `json:"preview_environment_id"`
	SourceType           string    `json:"type"`
	SourceServiceId      string    `json:"source_service_id"`
	CreatedAt            time.Time `json:"created_at,omitempty"`
}

type PreviewSource struct {
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

type PreviewServiceView struct {
	Env     PreviewEnvironment
	Service PreviewEnvironmentService
}

func nullableInt(v int) interface{} {
	if v == 0 {
		return nil
	}
	return v
}

func nullableInt64(v int64) interface{} {
	if v == 0 {
		return nil
	}
	return v
}

func (s *SupabaseStore) GetPreviewEnvironmentsByProject(projectId string) ([]PreviewEnvironment, error) {
	res, _, err := s.cli.From("preview_environments").
		Select("*", "exact", false).
		Eq("project_id", projectId).
		Order("created_at", &postgrest.OrderOpts{Ascending: false}).
		Execute()
	if err != nil {
		return nil, err
	}
	out := []PreviewEnvironment{}
	if err := json.Unmarshal(res, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *SupabaseStore) GetPreviewEnvironmentByID(id string) (PreviewEnvironment, error) {
	res, _, err := s.cli.From("preview_environments").
		Select("*", "exact", false).
		Eq("id", id).
		Single().
		Execute()
	if err != nil {
		return PreviewEnvironment{}, err
	}
	var out PreviewEnvironment
	if err := json.Unmarshal(res, &out); err != nil {
		return PreviewEnvironment{}, err
	}
	return out, nil
}

func (s *SupabaseStore) GetPreviewEnvironmentByNamespace(namespace string) (PreviewEnvironment, error) {
	res, _, err := s.cli.From("preview_environments").
		Select("*", "exact", false).
		Eq("namespace", namespace).
		Single().
		Execute()
	if err != nil {
		return PreviewEnvironment{}, err
	}
	var out PreviewEnvironment
	if err := json.Unmarshal(res, &out); err != nil {
		return PreviewEnvironment{}, err
	}
	return out, nil
}

func (s *SupabaseStore) GetPreviewEnvironmentByProjectPR(projectId string, prNumber int) (PreviewEnvironment, error) {
	res, _, err := s.cli.From("preview_environments").
		Select("*", "exact", false).
		Eq("project_id", projectId).
		Eq("pr", strconv.Itoa(prNumber)).
		Single().
		Execute()
	if err != nil {
		return PreviewEnvironment{}, err
	}
	var out PreviewEnvironment
	if err := json.Unmarshal(res, &out); err != nil {
		return PreviewEnvironment{}, err
	}
	return out, nil
}

func (s *SupabaseStore) EnsurePreviewEnvironment(projectId, name string, prNumber int, namespace string, commentID int64) (PreviewEnvironment, error) {
	res, _, err := s.cli.From("preview_environments").
		Select("*", "exact", false).
		Eq("project_id", projectId).
		Eq("name", name).
		Single().
		Execute()
	if err == nil {
		var existing PreviewEnvironment
		if uerr := json.Unmarshal(res, &existing); uerr == nil && existing.Id != "" {
			return existing, nil
		}
	}
	ins, _, err := s.cli.From("preview_environments").Insert(map[string]interface{}{
		"project_id": projectId,
		"name":       name,
		"comment_id": nullableInt64(commentID),
		"pr":         nullableInt(prNumber),
		"namespace":  namespace,
	}, false, "", "", "").Single().Execute()
	if err != nil {
		if adopted, aerr := s.GetPreviewEnvironmentByNamespace(namespace); aerr == nil && adopted.Id != "" {
			return adopted, nil
		}
		return PreviewEnvironment{}, err
	}
	var out PreviewEnvironment
	if err := json.Unmarshal(ins, &out); err != nil {
		return PreviewEnvironment{}, err
	}
	return out, nil
}

func (s *SupabaseStore) DeletePreviewEnvironment(id string) error {
	_, _, err := s.cli.From("preview_environments").Delete("", "").Eq("id", id).Execute()
	return err
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
	if existing, err := s.GetPreviewEnvironmentService(envID, sourceType, sourceServiceID); err == nil && existing.Id != 0 {
		return existing, nil
	}
	res, _, err := s.cli.From("preview_environment_services").Insert(map[string]interface{}{
		"preview_environment_id": envID,
		"type":                   sourceType,
		"source_service_id":      sourceServiceID,
		"created_at":             time.Now().UTC().Format(time.RFC3339),
	}, false, "", "", "").Single().Execute()
	if err != nil {
		if existing, aerr := s.GetPreviewEnvironmentService(envID, sourceType, sourceServiceID); aerr == nil && existing.Id != 0 {
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

func (s *SupabaseStore) GetPreviewSource(sourceType, sourceServiceID string) (PreviewSource, error) {
	switch sourceType {
	case "github_service":
		g, err := s.GetGithubServiceById(sourceServiceID)
		if err != nil {
			return PreviewSource{}, err
		}
		return PreviewSource{
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
			return PreviewSource{}, err
		}
		return PreviewSource{
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
			return PreviewSource{}, err
		}
		return PreviewSource{
			Type:         "database",
			Id:           d.Id,
			ProjectId:    d.ProjectId,
			Name:         d.Name,
			ResourceName: d.ResourceName,
			Port:         d.Port,
			Engine:       d.Engine,
		}, nil
	default:
		return PreviewSource{}, errUnknownPreviewSourceType(sourceType)
	}
}

func (s *SupabaseStore) GetPreviewServiceViewsBySource(sourceServiceID string) ([]PreviewServiceView, error) {
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
	out := make([]PreviewServiceView, 0, len(children))
	for _, c := range children {
		env, err := s.GetPreviewEnvironmentByID(c.PreviewEnvironmentId)
		if err != nil {
			continue
		}
		out = append(out, PreviewServiceView{Env: env, Service: c})
	}
	return out, nil
}

func (s *SupabaseStore) GetPreviewServiceViewsByRepoPR(repoId int64, prNumber int) ([]PreviewServiceView, error) {
	services, err := s.GetGithubServicesByRepoId(repoId)
	if err != nil {
		return nil, err
	}
	out := []PreviewServiceView{}
	for _, svc := range services {
		views, err := s.GetPreviewServiceViewsBySource(svc.Id)
		if err != nil {
			continue
		}
		for _, v := range views {
			if v.Env.Pr != prNumber {
				continue
			}
			out = append(out, v)
		}
	}
	return out, nil
}

func (s *SupabaseStore) GetPreviewEnvironmentsForProject(projectId string) ([]PreviewEnvironment, []PreviewEnvironmentService, error) {
	owned, err := s.GetPreviewEnvironmentsByProject(projectId)
	if err != nil {
		return nil, nil, err
	}
	seen := map[string]bool{}
	for _, e := range owned {
		seen[e.Id] = true
	}
	envs := append([]PreviewEnvironment{}, owned...)
	var membership []PreviewEnvironmentService
	collect := func(ids []string) {
		for _, id := range ids {
			views, err := s.GetPreviewServiceViewsBySource(id)
			if err != nil {
				continue
			}
			for _, v := range views {
				membership = append(membership, v.Service)
				if !seen[v.Env.Id] {
					seen[v.Env.Id] = true
					envs = append(envs, v.Env)
				}
			}
		}
	}
	githubSources, err := s.GetGithubServicesByProjectId(projectId)
	if err != nil {
		return envs, membership, nil
	}
	githubIDs := make([]string, 0, len(githubSources))
	for _, svc := range githubSources {
		githubIDs = append(githubIDs, svc.Id)
	}
	collect(githubIDs)
	plainSources, err := s.GetServicesByProjectId(projectId)
	if err == nil {
		plainIDs := make([]string, 0, len(plainSources))
		for _, svc := range plainSources {
			plainIDs = append(plainIDs, svc.Id)
		}
		collect(plainIDs)
	}
	dbSources, err := s.GetDatabasesByProjectId(projectId)
	if err == nil {
		dbIDs := make([]string, 0, len(dbSources))
		for _, db := range dbSources {
			dbIDs = append(dbIDs, db.Id)
		}
		collect(dbIDs)
	}
	return envs, membership, nil
}

func (s *SupabaseStore) GetPreviewCommentID(projectId string, prNumber int) int64 {
	env, err := s.GetPreviewEnvironmentByProjectPR(projectId, prNumber)
	if err != nil {
		return 0
	}
	return env.CommentID
}

func (s *SupabaseStore) SetPreviewCommentID(projectId string, prNumber int, commentID int64) {
	env, err := s.GetPreviewEnvironmentByProjectPR(projectId, prNumber)
	if err != nil || env.Id == "" {
		return
	}
	_, _, _ = s.cli.From("preview_environments").
		Update(map[string]interface{}{"comment_id": commentID}, "", "").
		Eq("id", env.Id).
		Execute()
}

func (s *SupabaseStore) GetPreviewCommentIDByRepoPR(repoId int64, prNumber int) int64 {
	views, err := s.GetPreviewServiceViewsByRepoPR(repoId, prNumber)
	if err != nil {
		return 0
	}
	for _, v := range views {
		if v.Env.CommentID != 0 {
			return v.Env.CommentID
		}
	}
	return 0
}

func (s *SupabaseStore) SetPreviewCommentIDByRepoPR(repoId int64, prNumber int, commentID int64) {
	views, err := s.GetPreviewServiceViewsByRepoPR(repoId, prNumber)
	if err != nil {
		return
	}
	seen := map[string]bool{}
	for _, v := range views {
		if v.Env.Id == "" || seen[v.Env.Id] || v.Env.CommentID == commentID {
			continue
		}
		seen[v.Env.Id] = true
		_, _, _ = s.cli.From("preview_environments").
			Update(map[string]interface{}{"comment_id": commentID}, "", "").
			Eq("id", v.Env.Id).
			Execute()
	}
}

func (s *SupabaseStore) DeletePreviewEnvironmentServicesBySource(sourceServiceID string) error {
	_, _, err := s.cli.From("preview_environment_services").Delete("", "").Eq("source_service_id", sourceServiceID).Execute()
	return err
}
