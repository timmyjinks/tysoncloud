package config

import "testing"

func setRequired(t *testing.T) {
	t.Setenv("CLERK_API_KEY", "sk_test")
	t.Setenv("SUPABASE_URL", "https://supabase.example")
	t.Setenv("SUPABASE_API_KEY", "anon")
	t.Setenv("GITHUB_WEBHOOK_SECRET", "whsec")
}

func TestLoadDefaults(t *testing.T) {
	setRequired(t)
	for _, k := range []string{"ADDR", "ALLOWED_ORIGINS", "CLUSTER_IP", "REGISTRY_URL", "KUBECONFIG", "BUILDKIT_HOST", "BUILDKIT_TLS_CA_CERT", "BUILDKIT_TLS_CERT", "BUILDKIT_TLS_KEY", "REGISTRY_DOCKER_CONFIG"} {
		t.Setenv(k, "")
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Server.Addr != ":8080" {
		t.Errorf("Addr = %q, want :8080", cfg.Server.Addr)
	}
	if cfg.Server.AllowedOrigins != "" {
		t.Errorf("AllowedOrigins = %q, want empty (server falls back to production origins)", cfg.Server.AllowedOrigins)
	}
	if cfg.Server.ClerkApiKey != "sk_test" || cfg.Supabase.ProjectURL != "https://supabase.example" || cfg.Supabase.APIKey != "anon" {
		t.Errorf("required values not loaded: %+v", cfg)
	}
	if cfg.Registry.URL == "" || cfg.KubeConfig == "" || cfg.Server.ClusterIp == "" {
		t.Errorf("defaults missing: %+v", cfg)
	}
	if cfg.Buildkit.TLSEnabled() || cfg.Registry.DockerConfigDir != "" {
		t.Errorf("buildkit TLS / registry auth should be off by default: %+v", cfg)
	}
}

func TestLoadBuildkitTLS(t *testing.T) {
	setRequired(t)
	t.Setenv("BUILDKIT_HOST", "tcp://buildkitd.tc-build.svc.cluster.local:1234")
	t.Setenv("BUILDKIT_TLS_CA_CERT", "/certs/ca.crt")
	t.Setenv("BUILDKIT_TLS_CERT", "/certs/tls.crt")
	t.Setenv("BUILDKIT_TLS_KEY", "/certs/tls.key")
	t.Setenv("REGISTRY_DOCKER_CONFIG", "/docker")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.Buildkit.TLSEnabled() || cfg.Buildkit.Host != "tcp://buildkitd.tc-build.svc.cluster.local:1234" || cfg.Registry.DockerConfigDir != "/docker" {
		t.Errorf("buildkit/registry config not loaded: %+v", cfg)
	}
}

func TestLoadRejectsPartialBuildkitTLS(t *testing.T) {
	setRequired(t)
	t.Setenv("BUILDKIT_TLS_CA_CERT", "/certs/ca.crt")
	t.Setenv("BUILDKIT_TLS_CERT", "")
	t.Setenv("BUILDKIT_TLS_KEY", "/certs/tls.key")
	if _, err := Load(); err == nil {
		t.Fatal("Load accepted a partial BuildKit TLS config")
	}
}

func TestLoadOverrides(t *testing.T) {
	setRequired(t)
	t.Setenv("ADDR", ":9999")
	t.Setenv("REGISTRY_URL", "registry.local:5000")
	t.Setenv("GITHUB_WEBHOOK_SECRET", "whsec")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Server.Addr != ":9999" || cfg.Registry.URL != "registry.local:5000" || cfg.Github.WebhookSecret != "whsec" {
		t.Errorf("overrides not applied: %+v", cfg)
	}
}

func TestLoadPanicsWithoutRequired(t *testing.T) {
	for _, missing := range []string{"CLERK_API_KEY", "SUPABASE_URL", "SUPABASE_API_KEY", "GITHUB_WEBHOOK_SECRET"} {
		t.Run(missing, func(t *testing.T) {
			setRequired(t)
			t.Setenv(missing, "")
			defer func() {
				if recover() == nil {
					t.Fatalf("Load did not panic without %s", missing)
				}
			}()
			Load()
		})
	}
}

func TestGetInt(t *testing.T) {
	t.Setenv("TC_TEST_INT", "42")
	if got := getInt("TC_TEST_INT", 1); got != 42 {
		t.Errorf("getInt = %d, want 42", got)
	}
	t.Setenv("TC_TEST_INT", "nope")
	if got := getInt("TC_TEST_INT", 1); got != 1 {
		t.Errorf("getInt invalid = %d, want fallback 1", got)
	}
	t.Setenv("TC_TEST_INT", "")
	if got := getInt("TC_TEST_INT", 7); got != 7 {
		t.Errorf("getInt empty = %d, want fallback 7", got)
	}
}
