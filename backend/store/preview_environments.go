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
			views, err := s.GetPreviewEnvironmentViewsBySource(id)
			if err != nil {
				continue
			}
			for _, v := range views {
				membership = append(membership, v.Service)
				if !seen[v.Environment.Id] {
					seen[v.Environment.Id] = true
					envs = append(envs, v.Environment)
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
	views, err := s.GetPreviewEnvironmentViewsByRepoPR(repoId, prNumber)
	if err != nil {
		return 0
	}
	for _, v := range views {
		if v.Environment.CommentID != 0 {
			return v.Environment.CommentID
		}
	}
	return 0
}

func (s *SupabaseStore) SetPreviewCommentIDByRepoPR(repoId int64, prNumber int, commentID int64) {
	views, err := s.GetPreviewEnvironmentViewsByRepoPR(repoId, prNumber)
	if err != nil {
		return
	}
	seen := map[string]bool{}
	for _, v := range views {
		if v.Environment.Id == "" || seen[v.Environment.Id] || v.Environment.CommentID == commentID {
			continue
		}
		seen[v.Environment.Id] = true
		_, _, _ = s.cli.From("preview_environments").
			Update(map[string]interface{}{"comment_id": commentID}, "", "").
			Eq("id", v.Environment.Id).
			Execute()
	}
}
