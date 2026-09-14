package response

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"
)

type Response struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

func SendError(w http.ResponseWriter, r *http.Request, message string, statusCode int) {
	log.Warn().
		Ctx(r.Context()).
		Str("path", r.URL.Path).
		Int("status", statusCode).
		Str("error", message).
		Msg("Request error")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(Response{
		Success: false,
		Error:   message,
	}); err != nil {
		log.Error().Err(err).Msg("Failed to encode error response")
	}
}

func SendSuccess(w http.ResponseWriter, data any, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(Response{
		Success: true,
		Data:    data,
	}); err != nil {
		log.Error().Err(err).Msg("Failed to encode success response")
	}
}
