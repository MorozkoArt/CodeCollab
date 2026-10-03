package grpcv1

import (
	"context"

	"github.com/MorozkoArt/CodeCollab/pkg/grpcerr"
	pkgjwt "github.com/MorozkoArt/CodeCollab/pkg/jwt"
	"github.com/MorozkoArt/CodeCollab/services/auth/internal/repo/user"
	"github.com/MorozkoArt/CodeCollab/services/auth/internal/services"
	"github.com/MorozkoArt/CodeCollab/services/auth/pkg/authv1"
	"google.golang.org/grpc/codes"
)

type Server struct {
	authv1.UnimplementedAuthServiceServer
	authSvc services.AuthService
}

func NewServer(authSvc services.AuthService) *Server {
	return &Server{authSvc: authSvc}
}

func (s *Server) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.RegisterResponse, error) {
	if err := s.authSvc.Register(ctx, services.RegisterInput{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	}); err != nil {
		return nil, grpcerr.Error(ctx, err, "register",
			grpcerr.Map(services.ErrUserExists, codes.AlreadyExists, "user already exists"),
			grpcerr.Map(services.ErrTooManyRequests, codes.ResourceExhausted, "try again later"),
		)
	}
	return &authv1.RegisterResponse{}, nil
}

func (s *Server) ConfirmRegistration(ctx context.Context, req *authv1.ConfirmRegistrationRequest) (*authv1.ConfirmRegistrationResponse, error) {
	if err := s.authSvc.ConfirmRegistration(ctx, req.Email, req.Code); err != nil {
		return nil, grpcerr.Error(ctx, err, "confirm_registration",
			grpcerr.Map(services.ErrInvalidCode, codes.InvalidArgument, "invalid or expired code"),
		)
	}
	return &authv1.ConfirmRegistrationResponse{}, nil
}

func (s *Server) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	if err := s.authSvc.Login(ctx, req.Email, req.Password); err != nil {
		return nil, grpcerr.Error(ctx, err, "login",
			grpcerr.Map(services.ErrInvalidPassword, codes.Unauthenticated, "invalid credentials"),
			grpcerr.Map(services.ErrEmailNotVerified, codes.FailedPrecondition, "email not verified"),
			grpcerr.Map(services.ErrTooManyRequests, codes.ResourceExhausted, "try again later"),
		)
	}
	return &authv1.LoginResponse{}, nil
}

func (s *Server) VerifyLogin(ctx context.Context, req *authv1.VerifyLoginRequest) (*authv1.VerifyLoginResponse, error) {
	token, err := s.authSvc.VerifyLogin(ctx, req.Email, req.Code)
	if err != nil {
		return nil, grpcerr.Error(ctx, err, "verify_login",
			grpcerr.Map(services.ErrInvalidCode, codes.Unauthenticated, "invalid or expired code"),
		)
	}
	return &authv1.VerifyLoginResponse{Token: token}, nil
}

func (s *Server) Validate(ctx context.Context, req *authv1.ValidateRequest) (*authv1.ValidateResponse, error) {
	userID, err := s.authSvc.Validate(ctx, req.Token)
	if err != nil {
		return nil, grpcerr.Error(ctx, err, "validate",
			grpcerr.Map(pkgjwt.ErrInvalidToken, codes.Unauthenticated, "invalid token"),
		)
	}
	return &authv1.ValidateResponse{UserId: userID}, nil
}

func (s *Server) GetUser(ctx context.Context, req *authv1.GetUserRequest) (*authv1.GetUserResponse, error) {
	u, err := s.authSvc.GetUser(ctx, req.UserId)
	if err != nil {
		return nil, grpcerr.Error(ctx, err, "get_user",
			grpcerr.Map(user.ErrUserNotFound, codes.NotFound, "user not found"),
		)
	}
	return &authv1.GetUserResponse{
		Id:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		CreatedAt: u.CreatedAt.String(),
	}, nil
}
