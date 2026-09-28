package app

import (
	"context"
	"fmt"
	"net"

	grpcv1 "github.com/MorozkoArt/CodeCollab/services/auth/internal/api/grpc/v1"
	"github.com/MorozkoArt/CodeCollab/services/auth/pkg/authv1"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type GRPCApp struct {
	server *grpc.Server
	port   int
}

func NewGRPC(port int, handler *grpcv1.Server) *GRPCApp {
	srv := grpc.NewServer()
	authv1.RegisterAuthServiceServer(srv, handler)
	reflection.Register(srv)

	return &GRPCApp{server: srv, port: port}
}

func (a *GRPCApp) Run(ctx context.Context) error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", a.port))
	if err != nil {
		return fmt.Errorf("grpc listen: %w", err)
	}

	log.Info().Ctx(ctx).Int("port", a.port).Msg("gRPC server started")
	return a.server.Serve(lis)
}

func (a *GRPCApp) Stop(ctx context.Context) {
	a.server.GracefulStop()
	log.Info().Ctx(ctx).Msg("gRPC server stopped")
}
