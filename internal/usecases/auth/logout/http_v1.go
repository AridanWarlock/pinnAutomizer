package logout

import (
	"net/http"
	"pinnAutomizer/internal/middleware/auth"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

func HttpV1Handler(log zerolog.Logger) http.HandlerFunc {
	log = log.With().Str("component", "http_v1: auth.Logout").Logger()

	return func(w http.ResponseWriter, r *http.Request) {
		httpV1(w, r, log)
	}
}

func httpV1(w http.ResponseWriter, r *http.Request, log zerolog.Logger) {
	log = log.With().Ctx(r.Context()).Logger()

	userID := r.Context().Value(auth.UserClaimsKey).(uuid.UUID)

	err := usecase.Logout(r.Context(), Input{ID: userID})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refreshToken",
		Path:     "/api/v1/auth/refresh",
		MaxAge:   -1,
		HttpOnly: true,
	})

	w.WriteHeader(http.StatusOK)
}
