package store

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/supabase-community/postgrest-go"
)

// PreviewEnvironmentsTable tracks one PR preview per parent github_service.
//
// All services for the same repo+PR share ONE namespace
// (util.PreviewNamespaceForRepo); each row keeps its own resource_name and
// hostname inside it. No env values are stored here — env is inherited
// per-service on every build from that service's K8s secret in the parent
// `proj-<id>` namespace, plus PREVIEW_URL.
type PreviewEnvironmentsTable struct {
	Id                 string `json:"id,omitempty"`
	GithubConnectionId string `json:"github_connection_id"`
	GithubServiceId    string `json:"github_service_id"`
	RepoId             int64  `json:"repo_id"`
	RepoName           string `json:"repo_name,omitempty"`
	PrNumber           int    `json:"pr_number"`
	HeadSHA            string `json:"head_sha,omitempty"`
	HeadRef            string `json:"head_ref,omitempty"`
	Namespace          string `json:"namespace"`
	ResourceName       string `json:"resource_name"`
	Hostname           string `json:"hostname,omitempty"`
	Image              string `json:"image,omitempty"`
	Status             string `json:"status"`
	// CommentID is the single live PR comment for the whole PR (shared by all
	// services in a monorepo fan-out). Stored redundantly on every row for the
	// PR so any row can recover it; SetPreviewCommentID keeps them in sync.
	CommentID int64     `json:"comment_id,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

func (s *SupabaseStore) GetPreviewEnvironmentsByRepoPR(repoId int64, prNumber int) ([]PreviewEnvironmentsTable, error) {
	res, _, err := s.cli.From("preview_environments").
		Select("*", "exact", false).
		Eq("repo_id", strconv.FormatInt(repoId, 10)).
		Eq("pr_number", strconv.Itoa(prNumber)).
		Execute()
	if err != nil {
		return nil, err
	}
	var out []PreviewEnvironmentsTable = []PreviewEnvironmentsTable{}
	if err := json.Unmarshal(res, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *SupabaseStore) GetPreviewEnvironmentsByService(serviceId string) ([]PreviewEnvironmentsTable, error) {
	res, _, err := s.cli.From("preview_environments").
		Select("*", "exact", false).
		Eq("github_service_id", serviceId).
		Order("created_at", &postgrest.OrderOpts{Ascending: false}).
		Execute()
	if err != nil {
		return nil, err
	}
	var out []PreviewEnvironmentsTable = []PreviewEnvironmentsTable{}
	if err := json.Unmarshal(res, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *SupabaseStore) GetPreviewEnvironment(serviceId string, prNumber int) (PreviewEnvironmentsTable, error) {
	res, _, err := s.cli.From("preview_environments").
		Select("*", "exact", false).
		Eq("github_service_id", serviceId).
		Eq("pr_number", strconv.Itoa(prNumber)).
		Single().
		Execute()
	if err != nil {
		return PreviewEnvironmentsTable{}, err
	}
	var out PreviewEnvironmentsTable
	if err := json.Unmarshal(res, &out); err != nil {
		return PreviewEnvironmentsTable{}, err
	}
	return out, nil
}

// UpsertPreviewEnvironment inserts or updates the tracking row for
// (github_service_id, pr_number). Uses direct PostgREST (no RPC) so no DB
// function is required.
func (s *SupabaseStore) UpsertPreviewEnvironment(row PreviewEnvironmentsTable) (PreviewEnvironmentsTable, error) {
	if existing, err := s.GetPreviewEnvironment(row.GithubServiceId, row.PrNumber); err == nil && existing.Id != "" {
		update := map[string]interface{}{
			"head_sha":      row.HeadSHA,
			"head_ref":      row.HeadRef,
			"namespace":     row.Namespace,
			"resource_name": row.ResourceName,
			"hostname":      row.Hostname,
			"status":        row.Status,
		}
		if row.Image != "" {
			update["image"] = row.Image
		}
		if row.RepoName != "" {
			update["repo_name"] = row.RepoName
		}
		if row.CommentID != 0 {
			update["comment_id"] = row.CommentID
		}
		res, _, err := s.cli.From("preview_environments").
			Update(update, "", "").
			Eq("id", existing.Id).
			Single().
			Execute()
		if err != nil {
			return PreviewEnvironmentsTable{}, err
		}
		var out PreviewEnvironmentsTable
		if err := json.Unmarshal(res, &out); err != nil {
			return PreviewEnvironmentsTable{}, err
		}
		return out, nil
	}
	res, _, err := s.cli.From("preview_environments").Insert(map[string]interface{}{
		"github_connection_id": row.GithubConnectionId,
		"github_service_id":    row.GithubServiceId,
		"repo_id":              row.RepoId,
		"repo_name":            row.RepoName,
		"pr_number":            row.PrNumber,
		"head_sha":             row.HeadSHA,
		"head_ref":             row.HeadRef,
		"namespace":            row.Namespace,
		"resource_name":        row.ResourceName,
		"hostname":             row.Hostname,
		"image":                row.Image,
		"status":               row.Status,
		"comment_id":           row.CommentID,
	}, false, "", "", "").Single().Execute()
	if err != nil {
		return PreviewEnvironmentsTable{}, err
	}
	var out PreviewEnvironmentsTable
	if err := json.Unmarshal(res, &out); err != nil {
		return PreviewEnvironmentsTable{}, err
	}
	return out, nil
}

func (s *SupabaseStore) UpdatePreviewEnvironmentStatus(id, status string) error {
	_, _, err := s.cli.From("preview_environments").
		Update(map[string]interface{}{"status": status}, "", "").
		Eq("id", id).
		Execute()
	return err
}

func (s *SupabaseStore) UpdatePreviewEnvironmentImage(id, image, status string) error {
	update := map[string]interface{}{"image": image}
	if status != "" {
		update["status"] = status
	}
	_, _, err := s.cli.From("preview_environments").
		Update(update, "", "").
		Eq("id", id).
		Execute()
	return err
}

func (s *SupabaseStore) DeletePreviewEnvironment(id string) error {
	_, _, err := s.cli.From("preview_environments").Delete("", "").Eq("id", id).Execute()
	return err
}

// GetPreviewCommentID returns the shared live-comment id for a PR (stored
// redundantly on every row; first non-zero wins, 0 = none yet).
func (s *SupabaseStore) GetPreviewCommentID(repoId int64, prNumber int) int64 {
	rows, err := s.GetPreviewEnvironmentsByRepoPR(repoId, prNumber)
	if err != nil {
		return 0
	}
	for _, r := range rows {
		if r.CommentID != 0 {
			return r.CommentID
		}
	}
	return 0
}

// SetPreviewCommentID syncs the shared live-comment id onto every row for a PR.
func (s *SupabaseStore) SetPreviewCommentID(repoId int64, prNumber int, commentID int64) {
	rows, err := s.GetPreviewEnvironmentsByRepoPR(repoId, prNumber)
	if err != nil {
		return
	}
	for _, r := range rows {
		if r.Id == "" || r.CommentID == commentID {
			continue
		}
		_, _, _ = s.cli.From("preview_environments").
			Update(map[string]interface{}{"comment_id": commentID}, "", "").
			Eq("id", r.Id).
			Execute()
	}
}

func (s *SupabaseStore) DeletePreviewEnvironmentsByService(serviceId string) error {
	_, _, err := s.cli.From("preview_environments").Delete("", "").Eq("github_service_id", serviceId).Execute()
	return err
}
