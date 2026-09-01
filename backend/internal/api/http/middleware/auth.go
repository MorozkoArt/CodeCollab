package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/MorozkoArt/CodeCollab/pkg/jwt"
	"github.com/rs/zerolog/log"
)

type contextKey string

const ClaimsKey contextKey = "claims"

func Auth(jwtService *jwt.Service) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))

			if token == "" {
				log.Warn().
					Ctx(r.Context()).
					Str("path", r.URL.Path).
					Str("remote_addr", r.RemoteAddr).
					Msg("Unauthorized: missing token")

				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			claims, err := jwtService.ValidateToken(token)
			if err != nil {
				log.Warn().
					Err(err).
					Ctx(r.Context()).
					Str("path", r.URL.Path).
					Msg("Unauthorized: invalid token")

				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), ClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
