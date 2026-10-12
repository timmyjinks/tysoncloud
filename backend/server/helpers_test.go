package server

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/timmyjinks/tysoncloud/deploy"
	"github.com/timmyjinks/tysoncloud/store"
)

func TestValidateToml(t *testing.T) {
	cfg := Config{
		Services: []Services{
			{Name: "ok", Image: "nginx", Port: 80},
			{Image: "nginx", Port: 80},
			{Name: "noimage", Port: 80},
			{Name: "noport", Image: "nginx"},
		},
		Databases: []Databases{
			{Name: "db", Engine: "postgres"},
			{Engine: "postgres"},
			{Name: "noengine"},
			{Name: "neg", Engine: "postgres", StorageGB: -1},
		},
	}
	got := ValidateToml(cfg)
	want := []Issue{
		{Message: "Service #2 is missing a name."},
		{Message: `Service "noimage" is missing a Docker image.`},
		{Message: `Service "noport" has an invalid port (must be 1 or greater).`},
		{Message: "Database #2 is missing a name."},
		{Message: `Database "noengine" is missing an engine.`},
		{Message: `Database "neg" has an invalid storage_gb (must be 0 or greater).`},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("issues =\n%v\nwant\n%v", got, want)
	}

	if issues := ValidateToml(Config{Services: []Services{{Name: "a", Image: "b", Port: 1}}}); len(issues) != 0 {
		t.Fatalf("valid config produced issues: %v", issues)
	}
}

func TestTomlParseIssue(t *testing.T) {
	var cfg Config
	_, err := toml.Decode("[[services]]\nname = \"web\"\nport = \n", &cfg)
	if err == nil {
		t.Fatalf("expected toml error")
	}
	issue := tomlParseIssue(err)
	if issue.Line != 3 {
		t.Fatalf("line = %d, want 3 (msg %q)", issue.Line, issue.Message)
	}
	if strings.HasPrefix(issue.Message, "toml: ") || strings.Contains(issue.Message, "\n") {
		t.Fatalf("message not cleaned: %q", issue.Message)
	}

	plain := tomlParseIssue(errors.New("toml: something\nmore detail"))
	if plain.Line != 0 || plain.Message != "something" {
		t.Fatalf("plain issue = %+v", plain)
	}
}

func TestToProjectData(t *testing.T) {
	services, databases := ToProjectData(
		[]store.ServicesTable{{ProjectId: "p1", ResourceName: "svc-abc", PublicDomain: "web.example.com", Image: "nginx", Port: 80}},
		[]store.DatabasesTable{{ProjectId: "p1", ResourceName: "db-abc", Engine: "postgres", StorageGB: 10}},
	)
	wantSvc := deploy.Service{Namespace: "proj-p1", Name: "svc-abc", Hostname: "web.example.com", Image: "nginx", Port: 80, Env: map[string][]byte{}}
	if len(services) != 1 || !reflect.DeepEqual(services[0], wantSvc) {
		t.Fatalf("services = %+v", services)
	}
	wantDB := deploy.Database{Namespace: "proj-p1", Name: "db-abc", Engine: "postgres", StorageGB: 10}
	if len(databases) != 1 || databases[0] != wantDB {
		t.Fatalf("databases = %+v", databases)
	}

	s, d := ToProjectData(nil, nil)
	if s == nil || d == nil {
		t.Fatalf("empty input should give empty non-nil slices (JSON [] not null)")
	}
}

func TestResponseMappersReturnEmptySlices(t *testing.T) {
	if ToProjectsResponse(nil) == nil || ToServicesResponse(nil) == nil || ToDatabasesResponse(nil) == nil || ToGithubServicesResponse(nil) == nil {
		t.Fatalf("mappers must return [] for nil input so JSON encodes [] not null")
	}
}

func TestToServicesResponse(t *testing.T) {
	now := time.Now()
	got := ToServicesResponse([]store.ServicesTable{{Id: "s1", ProjectId: "p1", Name: "web", Image: "nginx", Port: 80, Status: "running", PublicDomain: "pub", PrivateDomain: "priv", CreatedAt: now}})
	want := ServiceResponse{Id: "s1", ProjectId: "p1", Name: "web", Image: "nginx", Port: 80, Status: "running", PublicDomain: "pub", InternalDomain: "priv", CreatedAt: now}
	if !reflect.DeepEqual(got, []ServiceResponse{want}) {
		t.Fatalf("got %+v", got)
	}
}

func TestToDatabasesResponse(t *testing.T) {
	got := ToDatabasesResponse([]store.DatabasesTable{{Id: "d1", ProjectId: "p1", Name: "db", Engine: "postgres", Port: 5432, StorageGB: 5, InternalDomain: "db.internal"}})
	if len(got) != 1 || got[0].Storage != 5 || got[0].InternalDomain != "db.internal" || got[0].Port != 5432 {
		t.Fatalf("got %+v", got)
	}
}

func TestToGithubServicesResponseDefaultsBranch(t *testing.T) {
	got := ToGithubServicesResponse([]store.GithubServicesTable{
		{Id: "a", Branch: ""},
		{Id: "b", Branch: "  "},
		{Id: "c", Branch: "develop", RepoName: "me/repo", PrivateDomain: "priv"},
	})
	if got[0].Branch != "main" || got[1].Branch != "main" || got[2].Branch != "develop" {
		t.Fatalf("branches = %q %q %q", got[0].Branch, got[1].Branch, got[2].Branch)
	}
	if got[2].Repo != "me/repo" || got[2].InternalDomain != "priv" {
		t.Fatalf("mapping = %+v", got[2])
	}
}

func TestGetPort(t *testing.T) {
	if p, err := getPort("postgres"); err != nil || p != 5432 {
		t.Fatalf("postgres = %d, %v", p, err)
	}
	if _, err := getPort("mysql"); !errors.Is(err, invalidPort) {
		t.Fatalf("mysql err = %v, want invalidPort", err)
	}
}

func TestToPreviewEnvironmentServiceResponse(t *testing.T) {
	src := store.PreviewEnvironmentSource{Name: "web", RepoName: "me/repo", Branch: "feat", Port: 3000, Engine: ""}
	env := store.PreviewEnvironment{Id: "e1", Name: "pr-7", Pr: 7}
	child := store.PreviewEnvironmentService{Id: "c1", SourceType: "github_service", SourceServiceId: "src1"}

	got := toPreviewEnvironmentServiceResponse(src, env, child, "web-pr-7.example.com", nil, "running")
	if got.URL != "https://web-pr-7.example.com" || got.PreviewURL != got.URL {
		t.Errorf("url = %q / %q", got.URL, got.PreviewURL)
	}
	if got.PrURL != "https://github.com/me/repo/pull/7" {
		t.Errorf("pr url = %q", got.PrURL)
	}
	if got.Env == nil {
		t.Errorf("nil env should become empty map")
	}
	if got.Id != "c1" || got.PreviewEnvId != "e1" || got.SourceServiceId != "src1" || got.GithubServiceId != "src1" || got.Status != "running" || got.Port != 3000 {
		t.Errorf("mapping = %+v", got)
	}

	bare := toPreviewEnvironmentServiceResponse(store.PreviewEnvironmentSource{}, env, child, "", map[string]string{"A": "1"}, "")
	if bare.URL != "" || bare.PrURL != "" || bare.Env["A"] != "1" {
		t.Errorf("bare = %+v", bare)
	}
}

func TestPreviewStatusEmoji(t *testing.T) {
	tests := map[string]string{
		"running":   "✅",
		" Running ": "✅",
		"failed":    "❌",
		"building":  "🏗️",
		"deploying": "🏗️",
		"pending":   "🏗️",
		"torn_down": "🧹",
		"unknown":   "🏗️",
	}
	for in, want := range tests {
		if got := previewStatusEmoji(in); got != want {
			t.Errorf("previewStatusEmoji(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTruncateError(t *testing.T) {
	if got := truncateError("  short  ", 10); got != "short" {
		t.Errorf("got %q", got)
	}
	if got := truncateError("abcdefghij", 4); got != "abcd…" {
		t.Errorf("got %q", got)
	}
	if got := truncateError("abcd", 4); got != "abcd" {
		t.Errorf("exact length truncated: %q", got)
	}
}

func TestPreviewMarker(t *testing.T) {
	m := previewMarker(42, 7)
	if m != "<!-- tysoncloud-preview-pr:42:7 -->" {
		t.Fatalf("marker = %q", m)
	}
	if previewMarker(42, 8) == m || previewMarker(43, 7) == m {
		t.Fatalf("markers collide across repo/pr")
	}
}

func TestRenderPreviewMainComment(t *testing.T) {
	marker := previewMarker(1, 7)

	empty := renderPreviewMainComment(marker, "me/repo", 7, "abc", nil, nil)
	if !strings.HasPrefix(empty, marker+"\n") || !strings.Contains(empty, "No preview services tracked yet.") || !strings.HasSuffix(empty, previewRetryBox) {
		t.Fatalf("empty comment = %q", empty)
	}

	lines := []previewCommentLine{
		{ServiceID: "ok", Name: "web", Hostname: "web-pr-7.example.com", Status: "running"},
		{ServiceID: "bad", Name: "api", Hostname: "api-pr-7.example.com", Status: "running"},
	}
	errs := map[string]string{"bad": "line one\nline two"}
	body := renderPreviewMainComment(marker, "me/repo", 7, "ABCDEF0123456789", lines, errs)

	for _, want := range []string{
		"## TYSONCLOUD preview — PR #7",
		"- ✅ `web` (`abcdef012345`): https://web-pr-7.example.com",
		"- ❌ `api` failed (`abcdef012345`).",
		"  ```\n  line one\n  line two\n  ```",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("comment missing %q:\n%s", want, body)
		}
	}
	if !strings.HasSuffix(body, previewRetryBox) {
		t.Errorf("comment missing retry box")
	}
}

func TestRenderPreviewNotices(t *testing.T) {
	marker := previewMarker(1, 7)
	unlinked := renderPreviewUnlinkedComment(marker, "me/repo", 7)
	if !strings.HasPrefix(unlinked, marker) || !strings.Contains(unlinked, "not associated") || !strings.HasSuffix(unlinked, previewRetryBox) {
		t.Errorf("unlinked = %q", unlinked)
	}
	torn := renderPreviewTornDownComment(marker, 7)
	if !strings.HasPrefix(torn, marker) || !strings.Contains(torn, "torn down") || strings.Contains(torn, previewRetryBox) {
		t.Errorf("torn down = %q", torn)
	}
}

func TestPreviewErrorRegistry(t *testing.T) {
	const repo, pr = int64(990001), 3
	views := []store.PreviewEnvironmentView{
		{Service: store.PreviewEnvironmentService{SourceServiceId: "a"}},
		{Service: store.PreviewEnvironmentService{SourceServiceId: "b"}},
	}
	t.Cleanup(func() {
		clearPreviewError(repo, pr, "a")
		clearPreviewError(repo, pr, "b")
	})

	recordPreviewError(repo, pr, "a", "  build failed  ")
	recordPreviewError(repo, pr, "b", "   ") // blank errors are ignored
	recordPreviewError(repo, pr+1, "b", "other pr")

	got := snapshotPreviewErrors(repo, pr, views)
	if !reflect.DeepEqual(got, map[string]string{"a": "build failed"}) {
		t.Fatalf("snapshot = %v", got)
	}

	recordPreviewError(repo, pr, "a", strings.Repeat("x", 3000))
	if got := snapshotPreviewErrors(repo, pr, views)["a"]; len([]rune(got)) != 2001 {
		t.Fatalf("stored error len = %d runes, want 2000 + ellipsis", len([]rune(got)))
	}

	clearPreviewError(repo, pr, "a")
	if got := snapshotPreviewErrors(repo, pr, views); len(got) != 0 {
		t.Fatalf("snapshot after clear = %v", got)
	}
	clearPreviewError(repo, pr+1, "b")
}

func TestPreviewCommentLockIsPerPR(t *testing.T) {
	a := previewCommentLock(990002, 1)
	if previewCommentLock(990002, 1) != a {
		t.Fatalf("same repo/pr returned different locks")
	}
	if previewCommentLock(990002, 2) == a {
		t.Fatalf("different PRs share a lock")
	}
}
