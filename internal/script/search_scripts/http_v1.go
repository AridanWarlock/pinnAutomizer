package search_scripts

import (
	"fmt"
	"net/http"
	"pinnAutomizer/internal/domain"
	"pinnAutomizer/internal/domain/pagination"
	"pinnAutomizer/internal/middleware/auth"
	"pinnAutomizer/pkg/render"
	"strconv"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type Response struct {
	Scripts []script `json:"data"`
	Count   int      `json:"count"`
}

type script struct {
	ID         uuid.UUID `json:"id"`
	Filename   string    `json:"filename"`
	UploadTime string    `json:"uploadTime"`
	Text       string    `json:"text"`
}

func toApiModels(scripts []*domain.Script) []script {
	scriptResponses := make([]script, len(scripts))

	for i, s := range scripts {
		scriptResponses[i] = script{
			ID:         s.ID,
			Filename:   s.Filename,
			UploadTime: s.UploadTime.Format("2006-01-02 15:04:05"),
			Text:       s.Text,
		}
	}

	return scriptResponses
}

func HttpV1Handler(log zerolog.Logger) http.HandlerFunc {
	log = log.With().Str("component", "http_V1: script.SearchScripts").Logger()

	return func(w http.ResponseWriter, r *http.Request) {
		httpV1(w, r, log)
	}
}

func httpV1(w http.ResponseWriter, r *http.Request, log zerolog.Logger) {
	log = log.With().Ctx(r.Context()).Logger()

	q := r.URL.Query()
	f := &domain.ScriptFilter{}

	filename := q.Get("filename")
	if filename != "" {
		f.Filename = &filename
	}

	page, err := strconv.Atoi(q.Get("page"))
	if err != nil {
		http.Error(w, fmt.Sprintf("parse page error: %s", err.Error()), http.StatusBadRequest)
		return
	}
	if page <= 0 {
		http.Error(w, "parse page error: negative page", http.StatusBadRequest)
		return
	}
	page--

	limit, err := strconv.Atoi(q.Get("pageSize"))
	if err != nil {
		http.Error(w, fmt.Sprintf("parse perPage error: %s", err.Error()), http.StatusBadRequest)
		return
	}
	if limit <= 0 {
		http.Error(w, "parse perPage error: zero or negative perPage", http.StatusBadRequest)
		return
	}

	offset := limit * page
	userID := r.Context().Value(auth.UserIDKey).(uuid.UUID)

	input := Input{
		userID: userID,
		f:      f,
		p: pagination.NewOptions(
			pagination.WithLimit(limit),
			pagination.WithOffset(offset),
			pagination.WithSortFields(pagination.OrderBy("upload_time", true)),
		),
	}
	output, err := usecase.SearchScripts(r.Context(), input)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	render.JSON(w, Response{
		Scripts: toApiModels(output.Scripts),
		Count:   output.Count,
	}, http.StatusOK)
}
