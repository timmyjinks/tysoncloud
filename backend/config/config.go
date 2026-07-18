package config

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/caarlos0/env/v6"
	"github.com/joho/godotenv"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Server     Server   `toml:"server"`
	Supabase   Supabase `toml:"supabase"`
	Cloudflare CloudflareConfig
}

type CloudflareConfig struct {
	AccountID  string `env:"ACCOUNT_ID"`
	ZoneID     string `env:"ZONE_ID"`
	TunnelID   string `env:"TUNNEL_ID"`
	ApiToken   string `env:"CLOUDFLARE_API_TOKEN"`
	BaseDomain string `env:"BASE_DOMAIN"`
}

type Server struct {
	Addr    string `toml:"addr" env:"ADDR"`
	DistDir string `toml:"dist_dir" env:"DIST_DIR"`
	Origin  string `toml:"host" env:"ORIGIN"`
}

type Supabase struct {
	ProjectURL string `toml:"project_url"`
	APIKey     string `toml:"api_key" env:"SUPABASE_API_KEY"`
	ServiceKey string `env:"SUPABASE_SERVICE_KEY"`
}

func Load() (Config, error) {
	var config Config
	err := godotenv.Load()
	if err != nil {
		log.Println(err)
	}

	if _, err := toml.DecodeFile("./configs/config.toml", &config); err != nil {
		fmt.Println("Failed to read config: `", err, "` using defaults")
		config = Config{}
	}

	if err := env.Parse(&config); err != nil {
		return Config{}, err
	}

	config.Server.InitDefaults()
	config.Supabase.InitDefaults()

	return config, nil
}

func (s *Server) InitDefaults() {
	addrEnv := os.Getenv("LISTEN_ADDR")
	if addrEnv != "" {
		s.Addr = addrEnv
	}
	if s.Addr == "" {
		s.Addr = ":3000"
	}
	distDirEnv := os.Getenv("DIST_DIR")
	if distDirEnv != "" {
		s.DistDir = distDirEnv
	}
	if s.DistDir == "" {
		s.DistDir = "./web/src"
	}
	origin := os.Getenv("HOST")
	if origin != "" {
		s.Origin = origin
	}
	if s.Origin == "" {
		s.Origin = "localhost:3000"
	}
}

func (s *Supabase) InitDefaults() {
	projectURLEnv := os.Getenv("SUPABASE_URL")
	if projectURLEnv != "" {
		s.ProjectURL = projectURLEnv
	}
	if s.ProjectURL == "" {
		s.ProjectURL = "http://127.0.0.1:54321"
	}
	apiKeyEnv := os.Getenv("SUPABASE_API_KEY")
	if apiKeyEnv != "" {
		s.APIKey = apiKeyEnv
	}
	if s.APIKey == "" {
		s.APIKey = ""
	}
	serviceKeyEnv := os.Getenv("SUPABASE_SERVICE_KEY")
	if serviceKeyEnv != "" {
		s.ServiceKey = serviceKeyEnv
	}
	if s.ServiceKey == "" {
		s.ServiceKey = ""
	}
}

func (s *Server) InitFlags() {
	flag.StringVar(&s.Addr, "addr", s.Addr, "The address for the server to listen on")
	flag.StringVar(&s.DistDir, "dist-dir", s.DistDir, "The location of the built frontend")
}

func (s *Supabase) InitFlags() {
	flag.StringVar(&s.ProjectURL, "supabase-project-url", s.ProjectURL, "The Supabase project URL that the internal client will use")
	flag.StringVar(&s.APIKey, "supabase-api-key", s.APIKey, "The API key for the internal Supabase client to use")
}

func (c *Config) UpdateFromArgs() {
	c.Server.InitFlags()
	c.Supabase.InitFlags()
	flag.Parse()
}
