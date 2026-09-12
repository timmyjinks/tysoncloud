package main

import (
	"log"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/timmyjinks/tysoncloud/config"
	"github.com/timmyjinks/tysoncloud/db"
	"github.com/timmyjinks/tysoncloud/deploy"
	"github.com/timmyjinks/tysoncloud/github"
	"github.com/timmyjinks/tysoncloud/kubernetes"
	"github.com/timmyjinks/tysoncloud/server"
	"github.com/timmyjinks/tysoncloud/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	supabaseCli, err := db.NewSupabaseStorage(cfg.Supabase.ProjectURL, cfg.Supabase.APIKey)
	if err != nil {
		panic(err)
	}
	supabaseService := store.NewSupabaseStore(supabaseCli)

	kubernetesService, err := kubernetes.NewKubernetesService(cfg.KubeConfig, cfg.Server.ClusterIp)
	if err != nil {
		panic(err)
	}

	deployService := deploy.NewDeployService(kubernetesService)

	githubService := github.NewService(cfg.Github, cfg.Registry)

	clerk.SetKey(cfg.Server.ClerkApiKey)

	taskRegistry := server.NewTaskRegistry()

	app := &server.Application{
		Config:       cfg,
		Supabase:     supabaseService,
		Deploy:       deployService,
		Github:       githubService,
		TaskRegistry: taskRegistry,
	}

	s := server.Mount(app)
	err = app.Start(s)
	log.Fatal(err)
}
