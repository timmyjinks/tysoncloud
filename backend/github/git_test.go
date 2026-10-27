package github

import (
	"context"
	"encoding/base64"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestGitAuthEnv(t *testing.T) {
	const token = "ghs_secret123"
	env := gitAuthEnv(token)

	got := map[string]string{}
	for _, kv := range env {
		if strings.Contains(kv, token) {
			t.Fatalf("raw token leaked into env entry %q", kv)
		}
		if k, v, ok := strings.Cut(kv, "="); ok && strings.HasPrefix(k, "GIT_CONFIG_") {
			got[k] = v
		}
	}
	if got["GIT_CONFIG_COUNT"] != "1" || got["GIT_CONFIG_KEY_0"] != "http.https://github.com/.extraheader" {
		t.Fatalf("git config env = %v", got)
	}
	want := "AUTHORIZATION: basic " + base64.StdEncoding.EncodeToString([]byte("x-access-token:"+token))
	if got["GIT_CONFIG_VALUE_0"] != want {
		t.Fatalf("extraheader = %q, want %q", got["GIT_CONFIG_VALUE_0"], want)
	}

	for _, kv := range gitAuthEnv("") {
		if strings.HasPrefix(kv, "GIT_CONFIG_KEY_0=") {
			t.Fatalf("no token should add no git config, got %q", kv)
		}
	}
}

// The clone must not persist the token anywhere in the build context, and
// stripGitDir must leave only the working tree.
func TestCloneKeepsTokenOutOfBuildContext(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	const token = "ghs_secret123"

	src := t.TempDir()
	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"-c", "user.email=t@t", "-c", "user.name=t", "commit", "-q", "--allow-empty", "-m", "init"},
	} {
		if out, err := exec.Command("git", append([]string{"-C", src}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if err := os.WriteFile(filepath.Join(src, "index.js"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", src, "add", "-A").CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	if out, err := exec.Command("git", "-C", src, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "-q", "-m", "file").CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}

	cloneDir, err := cloneRepo(context.Background(), "file://"+src, token, "main")
	if err != nil {
		t.Fatalf("cloneRepo: %v", err)
	}
	defer os.RemoveAll(cloneDir)

	cfg, err := os.ReadFile(filepath.Join(cloneDir, ".git", "config"))
	if err != nil {
		t.Fatalf("read .git/config: %v", err)
	}
	if strings.Contains(string(cfg), token) {
		t.Fatalf(".git/config contains the token:\n%s", cfg)
	}

	if err := stripGitDir(cloneDir); err != nil {
		t.Fatalf("stripGitDir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cloneDir, ".git")); !os.IsNotExist(err) {
		t.Fatalf(".git still present after stripGitDir (err=%v)", err)
	}
	if _, err := os.Stat(filepath.Join(cloneDir, "index.js")); err != nil {
		t.Fatalf("working tree file missing: %v", err)
	}
}
