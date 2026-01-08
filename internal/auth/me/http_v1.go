package me

import (
	"net/http"
	"pinnAutomizer/internal/middleware/auth"
	"pinnAutomizer/pkg/render"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type Response struct {
	ID    uuid.UUID `json:"id"`
	Login string    `json:"login"`
}

func HttpV1Handler(log zerolog.Logger) http.HandlerFunc {
	log = log.With().Str("component", "http_V1: auth.Me").Logger()

	return func(w http.ResponseWriter, r *http.Request) {
		httpV1(w, r, log)
	}
}

func httpV1(w http.ResponseWriter, r *http.Request, log zerolog.Logger) {
	log = log.With().Ctx(r.Context()).Logger()

	userID := r.Context().Value(auth.UserIDKey).(uuid.UUID)

	out, err := usecase.Me(r.Context(), Input{ID: userID})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	render.JSON(w, Response{
		ID:    out.ID,
		Login: out.Login,
	}, http.StatusOK)
}
