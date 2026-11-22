package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Server     Server
	Supabase   Supabase
	Github     Github
	Registry   Registry
	Buildkit   Buildkit
	KubeConfig string `env:"KUBECONFIG"`
}

type Server struct {
	Addr           string `env:"ADDR"`
	AllowedOrigins string `env:"ALLOWED_ORIGINS"`
	ClerkApiKey    string `env:"CLERK_API_KEY"`
	ClusterIp      string `env:"CLUSTER_IP"`
	MaxProjects    int    `env:"MAX_PROJECTS"`
}

type Github struct {
	WebhookSecret       string `env:"GITHUB_WEBHOOK_SECRET"`
	AppSlug             string `env:"GITHUB_APP_SLUG"`
	AppID               string `env:"GITHUB_APP_ID"`
	AppPrivateKey       string `env:"GITHUB_APP_PRIVATE_KEY"`
	InstallationToken   string `env:"GITHUB_INSTALLATION_TOKEN"`
	ClientID            string `env:"GITHUB_APP_CLIENT_ID"`
	ClientSecret        string `env:"GITHUB_APP_CLIENT_SECRET"`
	MaxConcurrentBuilds int    `env:"MAX_CONCURRENT_BUILDS"`
}

type Registry struct {
	URL             string `env:"REGISTRY_URL"`
	DockerConfigDir string `env:"REGISTRY_DOCKER_CONFIG"`
}

type Buildkit struct {
	Host       string `env:"BUILDKIT_HOST"`
	TLSCACert  string `env:"BUILDKIT_TLS_CA_CERT"`
	TLSCert    string `env:"BUILDKIT_TLS_CERT"`
	TLSKey     string `env:"BUILDKIT_TLS_KEY"`
	ServerName string `env:"BUILDKIT_TLS_SERVER_NAME"`
}

type Supabase struct {
	ProjectURL string `env:"SUPABASE_URL"`
	APIKey     string `env:"SUPABASE_API_KEY"`
}

func Load() (Config, error) {
	if err := godotenv.Load(); err != nil {
		slog.Error(err.Error())
	}

	cfg := Config{
		Server: Server{
			Addr:           getString("ADDR", ":8080"),
			AllowedOrigins: getString("ALLOWED_ORIGINS", ""),
			ClerkApiKey:    getStringOrDie("CLERK_API_KEY"),
			ClusterIp:      getString("CLUSTER_IP", "192.168.0.18"),
			MaxProjects:    getInt("MAX_PROJECTS", 10),
		},
		Supabase: Supabase{
			ProjectURL: getStringOrDie("SUPABASE_URL"),
			APIKey:     getStringOrDie("SUPABASE_API_KEY"),
		},
		Github: Github{
			WebhookSecret:       getStringOrDie("GITHUB_WEBHOOK_SECRET"),
			AppSlug:             getString("GITHUB_APP_SLUG", ""),
			AppID:               getString("GITHUB_APP_ID", ""),
			AppPrivateKey:       getString("GITHUB_APP_PRIVATE_KEY", ""),
			InstallationToken:   getString("GITHUB_INSTALLATION_TOKEN", ""),
			ClientID:            getString("GITHUB_APP_CLIENT_ID", ""),
			ClientSecret:        getString("GITHUB_APP_CLIENT_SECRET", ""),
			MaxConcurrentBuilds: getInt("MAX_CONCURRENT_BUILDS", 2),
		},
		Registry: Registry{
			URL:             getString("REGISTRY_URL", "10.43.41.193:5000"),
			DockerConfigDir: getString("REGISTRY_DOCKER_CONFIG", ""),
		},
		Buildkit: Buildkit{
			Host:       getString("BUILDKIT_HOST", ""),
			TLSCACert:  getString("BUILDKIT_TLS_CA_CERT", ""),
			TLSCert:    getString("BUILDKIT_TLS_CERT", ""),
			TLSKey:     getString("BUILDKIT_TLS_KEY", ""),
			ServerName: getString("BUILDKIT_TLS_SERVER_NAME", ""),
		},
		KubeConfig: getString("KUBECONFIG", "~/.kube/config"),
	}
	if err := cfg.Buildkit.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (b Buildkit) validate() error {
	set := 0
	for _, v := range []string{b.TLSCACert, b.TLSCert, b.TLSKey} {
		if v != "" {
			set++
		}
	}
	if set != 0 && set != 3 {
		return fmt.Errorf("BUILDKIT_TLS_CA_CERT, BUILDKIT_TLS_CERT and BUILDKIT_TLS_KEY must be set together")
	}
	return nil
}

func (b Buildkit) TLSEnabled() bool {
	return b.TLSCACert != "" && b.TLSCert != "" && b.TLSKey != ""
}

func getString(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}

func getStringOrDie(key string) string {
	val := os.Getenv(key)
	if val == "" {
		panic(fmt.Errorf("env is required for key: %s", key))
	}
	return val
}

func getInt(key string, fallback int) int {
	val := os.Getenv(key)
	i, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return i
}
