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

	_ "github.com/MorozkoArt/CodeCollab/docs"
	v1 "github.com/MorozkoArt/CodeCollab/internal/api/http/v1"
	"github.com/MorozkoArt/CodeCollab/internal/app"
	"github.com/MorozkoArt/CodeCollab/internal/config"
	"github.com/MorozkoArt/CodeCollab/internal/db"
	"github.com/MorozkoArt/CodeCollab/internal/repository"
	"github.com/MorozkoArt/CodeCollab/internal/services"
	jwtpkg "github.com/MorozkoArt/CodeCollab/pkg/jwt"
	"github.com/MorozkoArt/CodeCollab/pkg/logger"
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

	logger.Init(cfg.AppConfig.AppEnv)

	log.Info().Str("env", cfg.AppConfig.AppEnv).Msg("Starting CodeCollab")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := db.NewPostgresDB(ctx, cfg.DBConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer pool.Close()

	userRepo := repository.NewUserRepository(pool)
	jwtSvc := jwtpkg.NewService(cfg.JWTSecret(), cfg.TokenExpiry())
	authSvc := services.NewAuthService(userRepo, jwtSvc)

	httpApp := app.New(cfg.ServerConfig, func(r chi.Router) {
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
