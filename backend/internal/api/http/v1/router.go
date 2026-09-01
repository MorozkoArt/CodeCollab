package v1

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/MorozkoArt/CodeCollab/internal/api/http/middleware"
	"github.com/MorozkoArt/CodeCollab/internal/services"
	"github.com/MorozkoArt/CodeCollab/pkg/jwt"
	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger"
)

func Register(r chi.Router, authSvc *services.AuthService, jwtSvc *jwt.Service) {
	auth := newAuthHandler(authSvc)

	// Swagger UI
	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	r.Route("/api/v1", func(r chi.Router) {
		// Публичные роуты
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", auth.register)
			r.Post("/login", auth.login)
		})

		// Защищённые роуты
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(jwtSvc))

			// TODO: защищённые эндпоинты (rooms, sessions, etc.)
		})

		// Health check
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"status":    "ok",
				"services":  "codecollab",
				"timestamp": time.Now().Format(time.RFC3339),
			})
		})
	})
}
