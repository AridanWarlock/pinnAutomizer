package refresh

import (
	"net/http"
	"pinnAutomizer/pkg/render"

	"github.com/rs/zerolog"
)

type Response struct {
	AccessToken string `json:"accessToken"`
}

func HttpV1Handler(log zerolog.Logger) http.HandlerFunc {
	log = log.With().Str("component", "http_v1: auth.Refresh").Logger()

	return func(w http.ResponseWriter, r *http.Request) {
		httpV1(w, r, log)
	}
}

func httpV1(w http.ResponseWriter, r *http.Request, log zerolog.Logger) {
	log = log.With().Ctx(r.Context()).Logger()

	refreshToken, err := r.Cookie("refreshToken")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	in := Input{RefreshTokenString: refreshToken.Value}
	output, err := usecase.Refresh(r.Context(), in)

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	render.JSON(w,
		Response{AccessToken: string(output.AccessToken)},
		http.StatusOK,
	)
}
