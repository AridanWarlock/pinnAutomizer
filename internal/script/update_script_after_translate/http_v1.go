package update_script_after_translate

import (
	"fmt"
	"net/http"
	"pinnAutomizer/pkg/json"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type Request struct {
	ID      uuid.UUID `json:"id"`
	Scripts []string  `json:"scripts"`
	Count   int       `json:"count"`
}

func HttpV1Handler(log zerolog.Logger) http.HandlerFunc {
	log = log.With().Str("component", "http_V1: script.UpdateScriptAfterTranslate").Logger()

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

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "decode script id err", http.StatusBadRequest)
		return
	}

	input := Input{
		ID:   id,
		Text: strings.Join(req.Scripts, "\n"),
	}

	err = usecase.UpdateScriptAfterTranslate(r.Context(), input)
	if err != nil {
		http.Error(w, fmt.Sprintf("update script err: %s", err.Error()), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
