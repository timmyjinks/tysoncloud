package github

import (
	"reflect"
	"testing"
)

func TestRailpackEnvArgs(t *testing.T) {
	if got := railpackEnvArgs(nil); got != nil {
		t.Fatalf("nil env = %v, want nil", got)
	}
	if got := railpackEnvArgs(map[string][]byte{}); got != nil {
		t.Fatalf("empty env = %v, want nil", got)
	}

	got := railpackEnvArgs(map[string][]byte{
		"API_URL":  []byte("https://x"),
		"bad-key!": []byte("skip"),
		"":         []byte("skip"),
		"PORT":     []byte("8080"),
		"_PRIVATE": []byte("ok"),
		"9INVALID": []byte("skip"),
	})
	want := []string{"--env", "API_URL=https://x", "--env", "PORT=8080", "--env", "_PRIVATE=ok"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}

	if keys := railpackEnvKeys(map[string][]byte{"B": []byte("1"), "A": []byte("2")}); !reflect.DeepEqual(keys, []string{"A", "B"}) {
		t.Fatalf("keys = %v, want [A B]", keys)
	}
}

func TestBuildctlSecretArgs(t *testing.T) {
	if got := buildctlSecretArgs(nil); got != nil {
		t.Fatalf("nil env = %v, want nil", got)
	}

	got := buildctlSecretArgs(map[string][]byte{
		"CLERK_KEY": []byte("sk_x"),
		"bad-key!":  []byte("skip"),
		"API_URL":   []byte("https://x"),
	})
	want := []string{"--secret", "id=API_URL,env=API_URL", "--secret", "id=CLERK_KEY,env=CLERK_KEY"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestSecretsHash(t *testing.T) {
	if got := secretsHash(nil); got != "" {
		t.Fatalf("nil env hash = %q, want empty", got)
	}
	a := secretsHash(map[string][]byte{"A": []byte("1"), "B": []byte("2")})
	b := secretsHash(map[string][]byte{"B": []byte("2"), "A": []byte("1")})
	if a == "" || a != b {
		t.Fatalf("hash not deterministic: %q vs %q", a, b)
	}
	if c := secretsHash(map[string][]byte{"A": []byte("1"), "B": []byte("CHANGED")}); c == a {
		t.Fatalf("hash insensitive to value change: %q", c)
	}
	if len(a) != 64 {
		t.Fatalf("hash len = %d, want 64 hex chars", len(a))
	}
}

func TestExportEnv(t *testing.T) {
	base := []string{"PATH=/bin", "API_URL=old", "OTHER=1"}
	got := exportEnv(base, map[string][]byte{"API_URL": []byte("new"), "bad-key!": []byte("skip")})
	want := []string{"PATH=/bin", "OTHER=1", "API_URL=new"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if got := exportEnv(base, nil); !reflect.DeepEqual(got, base) {
		t.Fatalf("nil env changed base: %v", got)
	}
}
