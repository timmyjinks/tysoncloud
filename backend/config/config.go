package config

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Server     Server
	Supabase   Supabase
	Github     Github
	KubeConfig string `env:"KUBECONFIG"`
}

type Server struct {
	Addr           string `env:"ADDR"`
	AllowedOrigins string `env:"ALLOWED_ORIGINS"`
	ClerkApiKey    string `env:"CLERK_API_KEY"`
	ClusterIp      string `env:"CLUSTER_IP"`
}

type Github struct {
	WebhookSecret     string `env:"GITHUB_WEBHOOK_SECRET"`
	AppSlug           string `env:"GITHUB_APP_SLUG"`
	AppID             string `env:"GITHUB_APP_ID"`
	AppPrivateKey     string `env:"GITHUB_APP_PRIVATE_KEY"`
	InstallationToken string `env:"GITHUB_INSTALLATION_TOKEN"`
}

type Supabase struct {
	ProjectURL string `env:"SUPABASE_URL"`
	APIKey     string `env:"SUPABASE_API_KEY"`
}

func Load() (Config, error) {
	err := godotenv.Load()
	if err != nil {
		log.Println(err)
	}

	return Config{
		Server: Server{
			Addr:           getString("ADDR", ":8080"),
			AllowedOrigins: getString("ALLOWED_ORIGINS", "http://localhost:3000"),
			ClerkApiKey:    getStringOrDie("CLERK_API_KEY"),
			ClusterIp:      getString("CLUSTER_IP", "192.168.0.18"),
		},
		Supabase: Supabase{
			ProjectURL: getStringOrDie("SUPABASE_URL"),
			APIKey:     getStringOrDie("SUPABASE_API_KEY"),
		},
		Github: Github{
			WebhookSecret:     getString("GITHUB_WEBHOOK_SECRET", ""),
			AppSlug:           getString("GITHUB_APP_SLUG", ""),
			AppID:             getString("GITHUB_APP_ID", ""),
			AppPrivateKey:     getString("GITHUB_APP_PRIVATE_KEY", ""),
			InstallationToken: getString("GITHUB_INSTALLATION_TOKEN", ""),
		},
		KubeConfig: getString("KUBECONFIG", "~/.kube/config"),
	}, nil
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
