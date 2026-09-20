// @title           CodeCollab API
// @version         1.0
// @description     Collaborative code editor platform API
// @host            auth.codecollab.local
// @schemes         https
// @BasePath        /api/v1
// @securityDefinitions.apikey BearerAuth
// @in              header
// @name            Authorization
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	pkgdb "github.com/MorozkoArt/CodeCollab/pkg/db"
	jwtpkg "github.com/MorozkoArt/CodeCollab/pkg/jwt"
	"github.com/MorozkoArt/CodeCollab/pkg/logger"
	"github.com/MorozkoArt/CodeCollab/services/auth/config"
	_ "github.com/MorozkoArt/CodeCollab/services/auth/docs"
	v1 "github.com/MorozkoArt/CodeCollab/services/auth/internal/api/http/v1"
	"github.com/MorozkoArt/CodeCollab/services/auth/internal/app"
	"github.com/MorozkoArt/CodeCollab/services/auth/internal/repo"
	"github.com/MorozkoArt/CodeCollab/services/auth/internal/services"
	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

const (
	envFileFlag     = "WITH_ENV_FILE"
	envFileFlagOff  = "0"
	shutdownTimeout = 10 * time.Second
)

func init() {
	if os.Getenv(envFileFlag) != envFileFlagOff {
		if err := godotenv.Load(); err != nil {
			log.Info().Msg("No .env file found, using system env")
		}
	}
}

func main() {
	if err := run(); err != nil {
		log.Fatal().Err(err).Msg("Application terminated with error")
	}
}

func run() error {
	cfg := config.NewConfig()

	logger.Init(cfg.App.AppEnv)

	log.Info().Str("env", cfg.App.AppEnv).Msg("Starting CodeCollab")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dbClient, err := pkgdb.NewDBClient(ctx, cfg.DB)
	if err != nil {
		return fmt.Errorf("init db client: %w", err)
	}
	defer dbClient.Close()
	userRepo := repo.NewUserRepository(dbClient.SQL(), dbClient.Builder())

	jwtSvc := jwtpkg.NewService(cfg.JWTSecret(), cfg.TokenExpiry())
	authSvc := services.NewAuthService(userRepo, jwtSvc)

	httpApp := app.New(cfg.Server, func(r chi.Router) {
		v1.Register(r, authSvc, jwtSvc)
	})

	go func() {
		if err := httpApp.Run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error().Err(err).Msg("HTTP server error")
			stop()
		}
	}()

	<-ctx.Done()

	log.Info().Msg("Shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := httpApp.Stop(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown error: %w", err)
	}

	log.Info().Msg("Server stopped")
	return nil
}
