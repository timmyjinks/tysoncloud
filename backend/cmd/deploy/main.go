package main

import (
	"log"

	"github.com/timmyjinks/tysoncloud/cloudflare"
	"github.com/timmyjinks/tysoncloud/config"
	"github.com/timmyjinks/tysoncloud/db"
	"github.com/timmyjinks/tysoncloud/server"
	"github.com/timmyjinks/tysoncloud/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	cfg.UpdateFromArgs()

	cloudflareService := cloudflare.NewCloudflareService(cfg.Cloudflare.ApiToken, cfg.Cloudflare.TunnelID, cfg.Cloudflare.ZoneID)
	supabaseCli, err := db.NewSupabaseStorage(cfg.Supabase.ProjectURL, cfg.Supabase.APIKey)
	if err != nil {
		panic(err)
	}
	supabaseService := store.NewSupabaseStore(supabaseCli)

	app := &server.Application{
		Config:     cfg,
		Supabase:   supabaseService,
		Cloudflare: cloudflareService,
	}

	s := server.Mount(app)
	err = app.Start(s)
	log.Fatal(err)
}
