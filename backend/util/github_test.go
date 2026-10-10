package util

import "testing"

func TestGithubCloneURL(t *testing.T) {
	tests := []struct {
		name     string
		fullName string
		want     string
		wantErr  bool
	}{
		{name: "valid", fullName: "timmyjinks/tysoncloud", want: "https://github.com/timmyjinks/tysoncloud.git"},
		{name: "dots and dashes", fullName: "my-org/my.repo_name", want: "https://github.com/my-org/my.repo_name.git"},
		{name: "empty", fullName: "", wantErr: true},
		{name: "no owner", fullName: "tysoncloud", wantErr: true},
		{name: "path traversal", fullName: "owner/../evil", wantErr: true},
		{name: "extra segment", fullName: "owner/repo/extra", wantErr: true},
		{name: "full url", fullName: "https://evil.example/owner/repo", wantErr: true},
		{name: "host injection", fullName: "owner@evil.example/repo", wantErr: true},
		{name: "dot repo", fullName: "owner/..", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GithubCloneURL(tt.fullName)
			if (err != nil) != tt.wantErr {
				t.Fatalf("GithubCloneURL(%q) err = %v, wantErr %v", tt.fullName, err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("GithubCloneURL(%q) = %q, want %q", tt.fullName, got, tt.want)
			}
		})
	}
}
