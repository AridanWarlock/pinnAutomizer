package register

import (
	"net/http"
	"pinnAutomizer/pkg/json"

	"github.com/rs/zerolog"
)

type Request struct {
	Login             string `json:"login"`
	Password          string `json:"password"`
	PasswordConfirmed string `json:"passwordConfirmed"`
}

func HttpV1Handler(log zerolog.Logger) http.HandlerFunc {
	log = log.With().Str("component", "http_V1: auth.Register").Logger()

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
		Login:             req.Login,
		Password:          req.Password,
		PasswordConfirmed: req.PasswordConfirmed,
	}

	err := usecase.Register(r.Context(), in)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusCreated)
}
