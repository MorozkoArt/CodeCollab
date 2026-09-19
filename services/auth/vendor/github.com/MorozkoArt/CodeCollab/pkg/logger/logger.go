package logger

import (
	"context"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	envDevelopment = "development"
	envProduction  = "production"
)

func Init(ctx context.Context, env string) {
	zerolog.TimeFieldFormat = time.RFC3339

	switch env {
	case envDevelopment:
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}).With().Caller().Logger()
		zerolog.SetGlobalLevel(zerolog.DebugLevel)

	default:
		log.Logger = log.With().Caller().Logger()
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	log.Debug().Ctx(ctx).Str("env", env).Msg("Logger initialized")
}
