package main

import (
	"log"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/timmyjinks/tysoncloud/cloudflare"
	"github.com/timmyjinks/tysoncloud/config"
	"github.com/timmyjinks/tysoncloud/db"
	"github.com/timmyjinks/tysoncloud/deploy"
	"github.com/timmyjinks/tysoncloud/kubernetes"
	"github.com/timmyjinks/tysoncloud/server"
	"github.com/timmyjinks/tysoncloud/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	cfg.UpdateFromArgs()

	cloudflareService := cloudflare.NewCloudflareService(cfg.Cloudflare.ApiToken, cfg.Cloudflare.TunnelID, cfg.Cloudflare.ZoneID, cfg.Cloudflare.BaseDomain)
	supabaseCli, err := db.NewSupabaseStorage(cfg.Supabase.ProjectURL, cfg.Supabase.APIKey)
	if err != nil {
		panic(err)
	}
	supabaseService := store.NewSupabaseStore(supabaseCli)

	kubernetesService, err := kubernetes.NewKubernetesService(cfg.KubeConfig)
	if err != nil {
		panic(err)
	}

	deployService := deploy.NewDeployService(kubernetesService)
	if err != nil {
		panic(err)
	}

	clerk.SetKey(cfg.Server.ClerkApiKey)

	taskRegistry := server.NewTaskRegistry()

	app := &server.Application{
		Config:       cfg,
		Supabase:     supabaseService,
		Cloudflare:   cloudflareService,
		Deploy:       deployService,
		TaskRegistry: taskRegistry,
	}

	s := server.Mount(app)
	err = app.Start(s)
	log.Fatal(err)
}
