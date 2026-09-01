package v1

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/MorozkoArt/CodeCollab/internal/domain"
	"github.com/MorozkoArt/CodeCollab/internal/repository"
	"github.com/MorozkoArt/CodeCollab/internal/services"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog/log"
)

type Response struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

type authHandler struct {
	authService *services.AuthService
	validate    *validator.Validate
}

func newAuthHandler(svc *services.AuthService) *authHandler {
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
// @Param        request body      domain.RegisterRequest true "Register request"
// @Success      201     {object}  Response{data=string}
// @Failure      400     {object}  Response
// @Failure      409     {object}  Response
// @Failure      500     {object}  Response
// @Router       /auth/register [post]
func (h *authHandler) register(w http.ResponseWriter, r *http.Request) {
	var req domain.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, r, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		sendError(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.authService.Register(r.Context(), &req); err != nil {
		if errors.Is(err, repository.ErrUserExists) {
			sendError(w, r, "user with this email already exists", http.StatusConflict)
			return
		}
		log.Error().Err(err).Ctx(r.Context()).Msg("Failed to register user")
		sendError(w, r, "internal server error", http.StatusInternalServerError)
		return
	}

	sendSuccess(w, "user registered successfully", http.StatusCreated)
}

// login godoc
// @Summary      Login
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body      domain.LoginRequest true "Login request"
// @Success      200     {object}  Response{data=domain.LoginResponse}
// @Failure      400     {object}  Response
// @Failure      401     {object}  Response
// @Router       /auth/login [post]
func (h *authHandler) login(w http.ResponseWriter, r *http.Request) {
	var req domain.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, r, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.validate.Struct(req); err != nil {
		sendError(w, r, err.Error(), http.StatusBadRequest)
		return
	}

	user, token, err := h.authService.Login(r.Context(), &req)
	if err != nil {
		sendError(w, r, "invalid email or password", http.StatusUnauthorized)
		return
	}

	sendSuccess(w, domain.LoginResponse{Token: token, User: user}, http.StatusOK)
}

func sendError(w http.ResponseWriter, r *http.Request, message string, statusCode int) {
	log.Warn().
		Ctx(r.Context()).
		Str("path", r.URL.Path).
		Int("status", statusCode).
		Str("error", message).
		Msg("Request error")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(Response{Success: false, Error: message}); err != nil {
		log.Error().Err(err).Msg("Failed to encode JSON response")
	}
}

func sendSuccess(w http.ResponseWriter, data any, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(Response{Success: true, Data: data}); err != nil {
		log.Error().Err(err).Msg("Failed to encode JSON response")
	}
}
