package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/timmyjinks/tysoncloud/cloudflare"
	"github.com/timmyjinks/tysoncloud/config"
	"github.com/timmyjinks/tysoncloud/store"
)

type Application struct {
	Config       config.Config
	Supabase     *store.SupabaseStore
	Cloudflare   *cloudflare.CloudflareService
	TaskRegistry *TaskRegistry
}

func Mount(app *Application) http.Handler {
	r := mux.NewRouter()
	app.registerRoutes(r)

	return r
}

func (app *Application) Start(mux http.Handler) error {
	server := &http.Server{
		Addr:        app.Config.Server.Addr,
		Handler:     mux,
		ReadTimeout: 90 * time.Second,
		IdleTimeout: 90 * time.Second,
	}

	fmt.Printf("Server running on http://localhost:%s\n", app.Config.Server.Addr)
	fmt.Printf("Server running on http://0.0.0.0:%s\n", app.Config.Server.Addr)
	return server.ListenAndServe()
}
