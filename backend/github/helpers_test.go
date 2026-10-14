package github

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/timmyjinks/tysoncloud/config"
)

func TestRegistryTag(t *testing.T) {
	tests := []struct {
		registry, name, tag, want string
	}{
		{"10.0.0.1:5000", "web", "abc", "10.0.0.1:5000/web:abc"},
		{" 10.0.0.1:5000/ ", " web ", "abc", "10.0.0.1:5000/web:abc"},
		{"10.0.0.1:5000", "web", "", "10.0.0.1:5000/web:latest"},
		{"", "web", "abc", "local/web:abc"},
	}
	for _, tt := range tests {
		if got := RegistryTag(tt.registry, tt.name, tt.tag); got != tt.want {
			t.Errorf("RegistryTag(%q, %q, %q) = %q, want %q", tt.registry, tt.name, tt.tag, got, tt.want)
		}
	}
}

func TestIsInfraBuildError(t *testing.T) {
	tests := []struct {
		err  error
		want bool
	}{
		{nil, false},
		{errors.New("buildctl: failed to dial"), true},
		{errors.New("Cannot connect to the Docker daemon at unix:///var/run/docker.sock"), true},
		{errors.New("dial tcp 10.0.0.1:1234: connection refused"), true},
		{errors.New("lookup buildkitd: no such host"), true},
		{errors.New("npm ERR! missing script: build"), false},
	}
	for _, tt := range tests {
		if got := IsInfraBuildError(tt.err); got != tt.want {
			t.Errorf("IsInfraBuildError(%v) = %v, want %v", tt.err, got, tt.want)
		}
	}
}

func TestGetInstallationTokenShortCircuits(t *testing.T) {
	ctx := context.Background()

	if _, err := NewService(config.Github{InstallationToken: "tok"}, config.Registry{}).GetInstallationToken(ctx, ""); err == nil {
		t.Errorf("empty installation id accepted")
	}

	tok, err := NewService(config.Github{InstallationToken: "tok"}, config.Registry{}).GetInstallationToken(ctx, "1")
	if err != nil || tok != "tok" {
		t.Errorf("static token = %q, %v; want tok", tok, err)
	}

	if _, err := NewService(config.Github{}, config.Registry{}).GetInstallationToken(ctx, "1"); err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Errorf("unconfigured app err = %v, want not configured", err)
	}
}

func TestGenerateAppJWT(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	pemKey := string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}))
	// Env files store the key on one line with literal \n.
	escaped := strings.ReplaceAll(pemKey, "\n", `\n`)

	s := NewService(config.Github{AppID: "12345", AppPrivateKey: escaped}, config.Registry{})
	signed, err := s.generateAppJWT()
	if err != nil {
		t.Fatalf("generateAppJWT: %v", err)
	}

	parsed, err := jwt.Parse(signed, func(*jwt.Token) (any, error) { return &key.PublicKey, nil }, jwt.WithValidMethods([]string{"RS256"}))
	if err != nil || !parsed.Valid {
		t.Fatalf("jwt invalid: %v", err)
	}
	if iss, _ := parsed.Claims.GetIssuer(); iss != "12345" {
		t.Fatalf("iss = %q, want 12345", iss)
	}

	bad := NewService(config.Github{AppID: "1", AppPrivateKey: "not a key"}, config.Registry{})
	if _, err := bad.generateAppJWT(); err == nil {
		t.Fatalf("invalid key accepted")
	}
}

func TestResolveBuildContext(t *testing.T) {
	clone := t.TempDir()
	if err := os.MkdirAll(filepath.Join(clone, "apps", "web"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(clone, "README.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		rootDir string
		want    string
		wantErr bool
	}{
		{"", clone, false},
		{".", clone, false},
		{"apps/web", filepath.Join(clone, "apps", "web"), false},
		{"missing", "", true},
		{"README.md", "", true},
		{"../escape", "", true},
	}
	for _, tt := range tests {
		got, err := resolveBuildContext(clone, tt.rootDir)
		if (err != nil) != tt.wantErr || got != tt.want {
			t.Errorf("resolveBuildContext(%q) = (%q, %v), want (%q, err=%v)", tt.rootDir, got, err, tt.want, tt.wantErr)
		}
	}
}

func TestBuildLogStore(t *testing.T) {
	s := NewBuildLogStore()
	s.Append("svc", "first")
	s.Append("svc", "") // ignored

	id, ch, snap := s.Subscribe("svc")
	if !reflect.DeepEqual(snap, []string{"first"}) {
		t.Fatalf("snapshot = %v, want [first]", snap)
	}

	s.Append("svc", "second")
	if got := <-ch; got != "second" {
		t.Fatalf("live line = %q, want second", got)
	}

	s.Append("other", "elsewhere")
	select {
	case l := <-ch:
		t.Fatalf("received line for another service: %q", l)
	default:
	}

	s.Unsubscribe("svc", id)
	if _, ok := <-ch; ok {
		t.Fatalf("channel not closed after unsubscribe")
	}
	s.Unsubscribe("svc", id)     // double unsubscribe is a no-op
	s.Unsubscribe("missing", 99) // unknown service is a no-op
}

func TestBuildLogStoreCapsHistory(t *testing.T) {
	s := NewBuildLogStore()
	for i := range maxBuildLogLines + 5 {
		s.Append("svc", fmt.Sprintf("line %d", i))
	}
	_, _, snap := s.Subscribe("svc")
	if len(snap) != maxBuildLogLines {
		t.Fatalf("history len = %d, want %d", len(snap), maxBuildLogLines)
	}
	if snap[0] != "line 5" {
		t.Fatalf("oldest line = %q, want line 5", snap[0])
	}
}

func TestBuildLogStoreSlowSubscriberDoesNotBlock(t *testing.T) {
	s := NewBuildLogStore()
	s.Subscribe("svc") // never drained
	for i := range 200 {
		s.Append("svc", fmt.Sprintf("line %d", i))
	}
}

func TestServiceBuildLogsLazyInit(t *testing.T) {
	var s Service // zero value: logs is nil
	s.UnsubscribeBuildLogs("svc", 0)
	s.AppendBuildLog("svc", "hello")
	_, _, snap := s.SubscribeBuildLogs("svc")
	if !reflect.DeepEqual(snap, []string{"hello"}) {
		t.Fatalf("snapshot = %v", snap)
	}
}

func TestLogWriter(t *testing.T) {
	var got []string
	w := &logWriter{fn: func(l string) { got = append(got, l) }, buf: &bytes.Buffer{}, mu: &sync.Mutex{}}

	w.Write([]byte("one\ntwo\n\nthree\n"))
	w.Write([]byte("partial"))

	want := []string{"one", "two", "three", "partial"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("lines = %q, want %q", got, want)
	}
	if w.buf.String() != "one\ntwo\n\nthree\npartial" {
		t.Fatalf("buffer = %q", w.buf.String())
	}
}
