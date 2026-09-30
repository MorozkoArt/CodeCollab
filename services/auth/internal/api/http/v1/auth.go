package v1

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/MorozkoArt/CodeCollab/pkg/response"
	"github.com/MorozkoArt/CodeCollab/services/auth/internal/services"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog/log"
)

type authHandler struct {
	authService services.AuthService
	validate    *validator.Validate
}

func newAuthHandler(svc services.AuthService) *authHandler {
	return &authHandler{
		authService: svc,
		validate:    validator.New(),
	}
}

// register godoc
// @Summary      Register a new user
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body      RegisterRequest true "Register request"
// @Success      201     {object}  response.Response
// @Failure      400     {object}  response.Response
// @Failure      409     {object}  response.Response
// @Failure      500     {object}  response.Response
// @Router       /auth/register [post]
func (h *authHandler) register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.SendError(w, r, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		response.SendError(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	err := h.authService.Register(r.Context(), services.RegisterInput{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		if errors.Is(err, services.ErrUserExists) {
			response.SendError(w, r, "user with this email already exists", http.StatusConflict)
			return
		}
		log.Error().Err(err).Ctx(r.Context()).Msg("register: internal error")
		response.SendError(w, r, "internal server error", http.StatusInternalServerError)
		return
	}

	response.SendSuccess(w, r, nil, http.StatusCreated)
}

// login godoc
// @Summary      Login
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body      LoginRequest true "Login request"
// @Success      200     {object}  response.Response{data=LoginResponse}
// @Failure      400     {object}  response.Response
// @Failure      401     {object}  response.Response
// @Router       /auth/login [post]
func (h *authHandler) login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.SendError(w, r, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		response.SendError(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	token, err := h.authService.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, services.ErrInvalidPassword) {
			response.SendError(w, r, "invalid email or password", http.StatusUnauthorized)
			return
		}
		log.Error().Err(err).Ctx(r.Context()).Msg("login: internal error")
		response.SendError(w, r, "internal server error", http.StatusInternalServerError)
		return
	}

	response.SendSuccess(w, r, LoginResponse{Token: token}, http.StatusOK)
}
