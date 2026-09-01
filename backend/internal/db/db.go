package db

import (
	"context"

	"github.com/MorozkoArt/CodeCollab/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

func NewPostgresDB(ctx context.Context, cfg *config.DBConfig) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, cfg.DSN())
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}

	log.Info().Msg("Connection to PostgreSQL established")
	return pool, nil
}
