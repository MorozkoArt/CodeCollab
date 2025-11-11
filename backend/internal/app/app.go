package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/MorozkoArt/CodeCollab/internal/config"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/log"

	httpmiddleware "github.com/MorozkoArt/CodeCollab/internal/api/http/middleware"
)

type App struct {
	server *http.Server
}

func New(cfg *config.ServerConfig, handler func(chi.Router)) *App {
	r := chi.NewRouter()

	r.Use(chimiddleware.Recoverer)
	r.Use(httpmiddleware.Logger)

	handler(r)

	return &App{
		server: &http.Server{
			Addr:         fmt.Sprintf("%s:%d", cfg.Host(), cfg.Port()),
			Handler:      r,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
	}
}

func (a *App) Run() error {
	log.Info().Str("addr", a.server.Addr).Msg("Starting HTTP server")
	return a.server.ListenAndServe()
}

func (a *App) Stop(ctx context.Context) error {
	log.Info().Msg("Shutting down HTTP server")
	return a.server.Shutdown(ctx)
}
