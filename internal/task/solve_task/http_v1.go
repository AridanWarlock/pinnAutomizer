package solve_task

import (
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"net/http"
	"pinnAutomizer/internal/middleware/auth"
	"pinnAutomizer/pkg/json"
)

type Request struct {
	TaskID    uuid.UUID      `json:"task_id"`
	Constants map[string]any `json:"constants"`
}

func HttpV1Handler(log zerolog.Logger) http.HandlerFunc {
	log = log.With().Str("component", "http_v1: task.SolveTask").Logger()

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

	userID := r.Context().Value(auth.UserIDKey).(uuid.UUID)

	in := Input{
		TaskID:    req.TaskID,
		Constants: req.Constants,
		UserID:    userID,
	}

	err := usecase.SolveTask(r.Context(), in)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
}
