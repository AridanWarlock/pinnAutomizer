package login

import (
	"net/http"
	"pinnAutomizer/pkg/json"
	"pinnAutomizer/pkg/render"

	"github.com/rs/zerolog"
)

type Request struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type Response struct {
	AccessToken string `json:"accessToken"`
}

func HttpV1Handler(log zerolog.Logger) http.HandlerFunc {
	log = log.With().Str("component", "http_v1: auth.Login").Logger()

	return func(w http.ResponseWriter, r *http.Request) {
		httpV1(w, r, log)
	}
}

func httpV1(w http.ResponseWriter, r *http.Request, log zerolog.Logger) {
	log = log.With().Ctx(r.Context()).Logger()

	var req Request

	if !json.MustParse(w, r, &req, log) {
		return
	}

	in := Input{
		Login:       req.Login,
		Password:    req.Password,
		Fingerprint: []byte(r.Header.Get("X-Fingerprint")),
	}

	out, err := usecase.Login(r.Context(), in)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refreshToken",
		Path:     "/api/v1/auth/refresh",
		Value:    out.RefreshTokenString,
		Expires:  out.RefreshTokenExpiresAt,
		SameSite: http.SameSiteStrictMode,
		HttpOnly: true,
	})

	response := Response{
		AccessToken: out.AccessTokenString,
	}
	render.JSON(w, response, http.StatusOK)
}
