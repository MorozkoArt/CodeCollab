package grpcerr

import (
	"context"
	"errors"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mapping struct {
	err  error
	code codes.Code
	msg  string
}

// Error маппит доменную ошибку в gRPC статус
// Если ошибка не найдена в маппинге - логирует и возвращает Internal
func Error(ctx context.Context, err error, op string, mappings ...mapping) error {
	for _, m := range mappings {
		if errors.Is(err, m.err) {
			return status.Error(m.code, m.msg)
		}
	}
	log.Error().Err(err).Ctx(ctx).Msgf("grpc %s: internal error", op)
	return status.Error(codes.Internal, op+" failed")
}

// Map создаёт маппинг доменной ошибки в gRPC код
func Map(err error, code codes.Code, msg string) mapping {
	return mapping{err: err, code: code, msg: msg}
}
