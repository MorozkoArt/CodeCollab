package v1_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/MorozkoArt/CodeCollab/pkg/jwt"
	"github.com/MorozkoArt/CodeCollab/pkg/password"
	"github.com/MorozkoArt/CodeCollab/pkg/response"
	v1 "github.com/MorozkoArt/CodeCollab/services/auth/internal/api/http/v1"
	"github.com/MorozkoArt/CodeCollab/services/auth/internal/domain"
	"github.com/MorozkoArt/CodeCollab/services/auth/internal/otp"
	codeRepo "github.com/MorozkoArt/CodeCollab/services/auth/internal/repo/code"
	userRepo "github.com/MorozkoArt/CodeCollab/services/auth/internal/repo/user"
	"github.com/MorozkoArt/CodeCollab/services/auth/internal/services"
	"github.com/MorozkoArt/CodeCollab/services/auth/pkg/enum"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	testJWTSecret = "test_secret"
	testJWTExpiry = time.Hour

	routeRegister        = "/api/v1/auth/register"
	routeRegisterConfirm = "/api/v1/auth/register/confirm"
	routeLogin           = "/api/v1/auth/login"
	routeLoginVerify     = "/api/v1/auth/login/verify"

	testEmail    = "test@example.com"
	testUsername = "testuser"
	testPassword = "password123"
	testUserID   = int64(1)
	testCode     = "123456"
)

var testOTPSecret = []byte("test_otp_secret_test_otp_secret_")

var testPasswordHash = sync.OnceValue(func() string {
	h, err := password.Hash(testPassword)
	if err != nil {
		panic(err)
	}
	return h
})

type env struct {
	users  *userRepo.MockUserRepository
	codes  *codeRepo.MockCodeRepository
	mailer *services.MockMailer
}

func newEnv() *env {
	return &env{
		users:  new(userRepo.MockUserRepository),
		codes:  new(codeRepo.MockCodeRepository),
		mailer: new(services.MockMailer),
	}
}

func (e *env) router() http.Handler {
	jwtSvc := jwt.NewService(testJWTSecret, testJWTExpiry)
	r := chi.NewRouter()
	v1.Register(r, services.NewAuthService(e.users, e.codes, jwtSvc, e.mailer, testOTPSecret), jwtSvc)
	return r
}

func (e *env) expectIssueCode(purpose string, upsertErr error) {
	e.codes.On("Upsert", mock.Anything, testUserID, purpose,
		mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(upsertErr)
	if upsertErr == nil {
		e.mailer.On("SendCode", mock.Anything, testEmail, purpose, mock.Anything, mock.Anything).Return(nil)
	}
}

func (e *env) expectAttempt(purpose, code string) {
	e.codes.On("Attempt", mock.Anything, testUserID, purpose, mock.Anything, mock.Anything).
		Return(otp.Hash(testOTPSecret, testUserID, purpose, code), nil)
}

func unverifiedUser() *domain.User {
	return &domain.User{ID: testUserID, Email: testEmail, Username: testUsername, Password: testPasswordHash()}
}

func verifiedUser() *domain.User {
	u := unverifiedUser()
	at := time.Now().Add(-time.Hour)
	u.EmailVerifiedAt = &at
	return u
}

func postRequest(t *testing.T, h http.Handler, path string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, path, bytes.NewBuffer(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func postJSON(t *testing.T, h http.Handler, path string, payload any) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(payload)
	require.NoError(t, err)
	return postRequest(t, h, path, body)
}

func decodeResponse(t *testing.T, w *httptest.ResponseRecorder) response.Response {
	t.Helper()
	var resp response.Response
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	return resp
}

func TestRegister(t *testing.T) {
	validReq := v1.RegisterRequest{Email: "new@example.com", Password: testPassword, Username: "newuser"}

	t.Run("success: 202", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", mock.Anything, "new@example.com").Return(nil, userRepo.ErrUserNotFound)
		e.users.On("Create", mock.Anything, mock.Anything).Run(func(args mock.Arguments) {
			args.Get(1).(*domain.User).ID = testUserID
		}).Return(nil)
		e.codes.On("Upsert", mock.Anything, testUserID, enum.PurposeRegister,
			mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		e.mailer.On("SendCode", mock.Anything, "new@example.com", enum.PurposeRegister, mock.Anything, mock.Anything).Return(nil)

		w := postJSON(t, e.router(), routeRegister, validReq)
		assert.Equal(t, http.StatusAccepted, w.Code)
		assert.True(t, decodeResponse(t, w).Success)
		e.users.AssertExpectations(t)
		e.codes.AssertExpectations(t)
		e.mailer.AssertExpectations(t)
	})

	t.Run("user exists: 409", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", mock.Anything, "new@example.com").Return(verifiedUser(), nil)

		w := postJSON(t, e.router(), routeRegister, validReq)
		assert.Equal(t, http.StatusConflict, w.Code)
		e.users.AssertExpectations(t)
	})

	t.Run("cooldown: 429", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", mock.Anything, "new@example.com").Return(unverifiedUser(), nil)
		e.codes.On("Upsert", mock.Anything, testUserID, enum.PurposeRegister,
			mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(codeRepo.ErrCooldown)

		w := postJSON(t, e.router(), routeRegister, validReq)
		assert.Equal(t, http.StatusTooManyRequests, w.Code)
		e.mailer.AssertNumberOfCalls(t, "SendCode", 0)
	})

	t.Run("invalid body: 400", func(t *testing.T) {
		w := postRequest(t, newEnv().router(), routeRegister, []byte("bad json"))
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("validation errors: 400", func(t *testing.T) {
		cases := map[string]v1.RegisterRequest{
			"short password": {Email: "a@example.com", Password: "short", Username: "newuser"},
			"bad email":      {Email: "not-an-email", Password: testPassword, Username: "newuser"},
			"short username": {Email: "a@example.com", Password: testPassword, Username: "ab"},
		}
		for name, req := range cases {
			t.Run(name, func(t *testing.T) {
				w := postJSON(t, newEnv().router(), routeRegister, req)
				assert.Equal(t, http.StatusBadRequest, w.Code)
			})
		}
	})
}

func TestConfirmRegistration(t *testing.T) {
	t.Run("success: 201", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", mock.Anything, testEmail).Return(unverifiedUser(), nil)
		e.expectAttempt(enum.PurposeRegister, testCode)
		e.codes.On("Delete", mock.Anything, testUserID, enum.PurposeRegister).Return(nil)
		e.users.On("MarkEmailVerified", mock.Anything, testUserID, mock.Anything).Return(nil)

		w := postJSON(t, e.router(), routeRegisterConfirm, v1.CodeRequest{Email: testEmail, Code: testCode})
		assert.Equal(t, http.StatusCreated, w.Code)
		e.users.AssertExpectations(t)
		e.codes.AssertExpectations(t)
	})

	t.Run("wrong code: 400", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", mock.Anything, testEmail).Return(unverifiedUser(), nil)
		e.expectAttempt(enum.PurposeRegister, "000000")

		w := postJSON(t, e.router(), routeRegisterConfirm, v1.CodeRequest{Email: testEmail, Code: testCode})
		assert.Equal(t, http.StatusBadRequest, w.Code)
		e.users.AssertNumberOfCalls(t, "MarkEmailVerified", 0)
	})

	t.Run("malformed code: 400", func(t *testing.T) {
		for _, code := range []string{"", "12345", "1234567", "12345a"} {
			w := postJSON(t, newEnv().router(), routeRegisterConfirm, v1.CodeRequest{Email: testEmail, Code: code})
			assert.Equal(t, http.StatusBadRequest, w.Code, "code %q", code)
		}
	})
}

func TestLogin(t *testing.T) {
	t.Run("success: 202, no token yet", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", mock.Anything, testEmail).Return(verifiedUser(), nil)
		e.expectIssueCode(enum.PurposeLogin, nil)

		w := postJSON(t, e.router(), routeLogin, v1.LoginRequest{Email: testEmail, Password: testPassword})
		assert.Equal(t, http.StatusAccepted, w.Code)

		resp := decodeResponse(t, w)
		assert.True(t, resp.Success)
		assert.Nil(t, resp.Data)
		e.mailer.AssertExpectations(t)
	})

	t.Run("wrong password: 401", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", mock.Anything, testEmail).Return(verifiedUser(), nil)

		w := postJSON(t, e.router(), routeLogin, v1.LoginRequest{Email: testEmail, Password: "wrong_password"})
		assert.Equal(t, http.StatusUnauthorized, w.Code)
		e.mailer.AssertNumberOfCalls(t, "SendCode", 0)
	})

	t.Run("unknown user: same 401 and message as wrong password", func(t *testing.T) {
		wrong := newEnv()
		wrong.users.On("GetByEmail", mock.Anything, testEmail).Return(verifiedUser(), nil)
		wWrong := postJSON(t, wrong.router(), routeLogin, v1.LoginRequest{Email: testEmail, Password: "wrong_password"})

		unknown := newEnv()
		unknown.users.On("GetByEmail", mock.Anything, testEmail).Return(nil, userRepo.ErrUserNotFound)
		wUnknown := postJSON(t, unknown.router(), routeLogin, v1.LoginRequest{Email: testEmail, Password: "wrong_password"})

		assert.Equal(t, http.StatusUnauthorized, wUnknown.Code)
		assert.Equal(t, decodeResponse(t, wWrong).Error, decodeResponse(t, wUnknown).Error)
	})

	t.Run("email not verified: 403", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", mock.Anything, testEmail).Return(unverifiedUser(), nil)

		w := postJSON(t, e.router(), routeLogin, v1.LoginRequest{Email: testEmail, Password: testPassword})
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("cooldown: 429", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", mock.Anything, testEmail).Return(verifiedUser(), nil)
		e.expectIssueCode(enum.PurposeLogin, codeRepo.ErrCooldown)

		w := postJSON(t, e.router(), routeLogin, v1.LoginRequest{Email: testEmail, Password: testPassword})
		assert.Equal(t, http.StatusTooManyRequests, w.Code)
	})

	t.Run("invalid body: 400", func(t *testing.T) {
		w := postRequest(t, newEnv().router(), routeLogin, []byte("bad json"))
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestVerifyLogin(t *testing.T) {
	t.Run("success: 200 with token", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", mock.Anything, testEmail).Return(verifiedUser(), nil)
		e.expectAttempt(enum.PurposeLogin, testCode)
		e.codes.On("Delete", mock.Anything, testUserID, enum.PurposeLogin).Return(nil)

		w := postJSON(t, e.router(), routeLoginVerify, v1.CodeRequest{Email: testEmail, Code: testCode})
		assert.Equal(t, http.StatusOK, w.Code)

		resp := decodeResponse(t, w)
		assert.True(t, resp.Success)
		data, ok := resp.Data.(map[string]any)
		require.True(t, ok)
		assert.NotEmpty(t, data["token"])
		e.codes.AssertExpectations(t)
	})

	t.Run("wrong code: 400, no token", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", mock.Anything, testEmail).Return(verifiedUser(), nil)
		e.expectAttempt(enum.PurposeLogin, "000000")

		w := postJSON(t, e.router(), routeLoginVerify, v1.CodeRequest{Email: testEmail, Code: testCode})
		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Nil(t, decodeResponse(t, w).Data)
	})

	t.Run("attempts exhausted: 400", func(t *testing.T) {
		e := newEnv()
		e.users.On("GetByEmail", mock.Anything, testEmail).Return(verifiedUser(), nil)
		e.codes.On("Attempt", mock.Anything, testUserID, enum.PurposeLogin, mock.Anything, mock.Anything).
			Return(nil, codeRepo.ErrCodeNotFound)

		w := postJSON(t, e.router(), routeLoginVerify, v1.CodeRequest{Email: testEmail, Code: testCode})
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
