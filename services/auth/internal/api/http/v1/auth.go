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

const maxBodyBytes = 1 << 20

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
// @Summary      Register a new user (step 1: sends confirmation code to email)
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body      RegisterRequest true "Register request"
// @Success      202     {object}  response.Response
// @Failure      400     {object}  response.Response
// @Failure      409     {object}  response.Response
// @Failure      429     {object}  response.Response
// @Failure      500     {object}  response.Response
// @Router       /auth/register [post]
func (h *authHandler) register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if !h.decode(w, r, &req) {
		return
	}

	err := h.authService.Register(r.Context(), services.RegisterInput{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		h.fail(w, r, "register", err)
		return
	}

	response.SendSuccess(w, r, nil, http.StatusAccepted)
}

// confirmRegistration godoc
// @Summary      Confirm registration (step 2: verify code from email)
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body      CodeRequest true "Email and code"
// @Success      201     {object}  response.Response
// @Failure      400     {object}  response.Response
// @Failure      500     {object}  response.Response
// @Router       /auth/register/confirm [post]
func (h *authHandler) confirmRegistration(w http.ResponseWriter, r *http.Request) {
	var req CodeRequest
	if !h.decode(w, r, &req) {
		return
	}

	if err := h.authService.ConfirmRegistration(r.Context(), req.Email, req.Code); err != nil {
		h.fail(w, r, "confirm_registration", err)
		return
	}

	response.SendSuccess(w, r, nil, http.StatusCreated)
}

// login godoc
// @Summary      Login (step 1: check password, send code to email)
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body      LoginRequest true "Login request"
// @Success      202     {object}  response.Response
// @Failure      400     {object}  response.Response
// @Failure      401     {object}  response.Response
// @Failure      403     {object}  response.Response
// @Failure      429     {object}  response.Response
// @Failure      500     {object}  response.Response
// @Router       /auth/login [post]
func (h *authHandler) login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if !h.decode(w, r, &req) {
		return
	}

	if err := h.authService.Login(r.Context(), req.Email, req.Password); err != nil {
		h.fail(w, r, "login", err)
		return
	}

	response.SendSuccess(w, r, nil, http.StatusAccepted)
}

// verifyLogin godoc
// @Summary      Login (step 2: verify code from email, get JWT)
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body      CodeRequest true "Email and code"
// @Success      200     {object}  response.Response{data=LoginResponse}
// @Failure      400     {object}  response.Response
// @Failure      500     {object}  response.Response
// @Router       /auth/login/verify [post]
func (h *authHandler) verifyLogin(w http.ResponseWriter, r *http.Request) {
	var req CodeRequest
	if !h.decode(w, r, &req) {
		return
	}

	token, err := h.authService.VerifyLogin(r.Context(), req.Email, req.Code)
	if err != nil {
		h.fail(w, r, "verify_login", err)
		return
	}

	response.SendSuccess(w, r, LoginResponse{Token: token}, http.StatusOK)
}

func (h *authHandler) decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		response.SendError(w, r, "invalid request body", http.StatusBadRequest)
		return false
	}

	if err := h.validate.Struct(dst); err != nil {
		response.SendError(w, r, err.Error(), http.StatusBadRequest)
		return false
	}

	return true
}

func (*authHandler) fail(w http.ResponseWriter, r *http.Request, op string, err error) {
	switch {
	case errors.Is(err, services.ErrUserExists):
		response.SendError(w, r, "user with this email already exists", http.StatusConflict)
	case errors.Is(err, services.ErrInvalidPassword):
		response.SendError(w, r, "invalid email or password", http.StatusUnauthorized)
	case errors.Is(err, services.ErrEmailNotVerified):
		response.SendError(w, r, "email is not verified", http.StatusForbidden)
	case errors.Is(err, services.ErrInvalidCode):
		response.SendError(w, r, "invalid or expired code", http.StatusBadRequest)
	case errors.Is(err, services.ErrTooManyRequests):
		response.SendError(w, r, "code was sent recently, try again later", http.StatusTooManyRequests)
	default:
		log.Error().Err(err).Ctx(r.Context()).Msgf("%s: internal error", op)
		response.SendError(w, r, "internal server error", http.StatusInternalServerError)
	}
}
