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
	KubeConfig string `env:"KUBECONFIG"`
}

type Server struct {
	Addr                string `env:"ADDR"`
	AllowedOrigins      string `env:"ALLOWED_ORIGINS"`
	ClerkApiKey         string `env:"CLERK_API_KEY"`
	ClusterIp           string `env:"CLUSTER_IP"`
	GithubWebhookSecret string `env:"GITHUB_WEBHOOK_SECRET"`
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
			Addr:                getString("ADDR", ":8080"),
			AllowedOrigins:      getString("ALLOWED_ORIGINS", "http://localhost:3000"),
			ClerkApiKey:         getStringOrDie("CLERK_API_KEY"),
			ClusterIp:           getString("CLUSTER_IP", "192.168.0.18"),
			GithubWebhookSecret: getString("GITHUB_WEBHOOK_SECRET", ""),
		},
		Supabase: Supabase{
			ProjectURL: getStringOrDie("SUPABASE_URL"),
			APIKey:     getStringOrDie("SUPABASE_API_KEY"),
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
