// @title           CodeCollab Auth API
// @version         1.0
// @description     Auth service API
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
	pkgjwt "github.com/MorozkoArt/CodeCollab/pkg/jwt"
	"github.com/MorozkoArt/CodeCollab/pkg/logger"
	_ "github.com/MorozkoArt/CodeCollab/services/auth/docs"
	grpcv1 "github.com/MorozkoArt/CodeCollab/services/auth/internal/api/grpc/v1"
	v1 "github.com/MorozkoArt/CodeCollab/services/auth/internal/api/http/v1"
	"github.com/MorozkoArt/CodeCollab/services/auth/internal/app"
	"github.com/MorozkoArt/CodeCollab/services/auth/internal/config"
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

func main() {
	if err := run(); err != nil {
		log.Fatal().Err(err).Msg("Application terminated with error")
	}
}

func run() error {
	if os.Getenv(envFileFlag) != envFileFlagOff {
		if err := godotenv.Load(); err != nil {
			log.Info().Msg("No .env file found, using system env")
		}
	}

	cfg := config.NewConfig()
	logger.Init(cfg.App.Env())

	log.Info().Str("env", cfg.App.Env()).Msg("Starting CodeCollab Auth")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dbClient, err := pkgdb.NewDBClient(ctx, cfg.DB)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer dbClient.Close()

	userRepo := repo.NewUserRepository(dbClient.SQL(), dbClient.Builder())
	jwtSvc := pkgjwt.NewService(cfg.Auth.JWTSecret(), cfg.Auth.TokenExpiry())
	authSvc := services.NewAuthService(userRepo, jwtSvc)

	httpApp := app.New(cfg.Server, func(r chi.Router) {
		v1.Register(r, authSvc, jwtSvc)
	})

	grpcApp := app.NewGRPC(cfg.Server.GRPCPort(), grpcv1.NewServer(authSvc))

	errCh := make(chan error, 2)

	go func() {
		if err := httpApp.Run(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("http: %w", err)
		}
	}()

	go func() {
		if err := grpcApp.Run(); err != nil {
			errCh <- fmt.Errorf("grpc: %w", err)
		}
	}()

	select {
	case <-ctx.Done():
		log.Info().Msg("Shutting down gracefully...")
	case err := <-errCh:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := httpApp.Stop(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("HTTP server shutdown error")
	}

	grpcApp.Stop()

	log.Info().Msg("Server stopped")
	return nil
}
