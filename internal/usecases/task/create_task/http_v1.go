package create_task

import (
	"net/http"
	"pinnAutomizer/internal/middleware/auth"
	"pinnAutomizer/pkg/json"
	"pinnAutomizer/pkg/render"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type Request struct {
	Name         string         `json:"name"`
	Description  string         `json:"description"`
	Constants    map[string]any `json:"constants"`
	EquationType string         `json:"equation_type"`
}

type Response struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`

	Status    string         `json:"status"`
	Constants map[string]any `json:"constants"`

	EquationType string `json:"equation_type"`

	CreatedAt time.Time `json:"created_at"`
}

func HttpV1Handler(log zerolog.Logger) http.HandlerFunc {
	log = log.With().Str("component", "http_v1: task.CreateTask").Logger()

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
		Name: req.Name,

		Description: req.Description,
		Constants:   req.Constants,

		EquationType: req.EquationType,
		UserID:       userID,
	}

	out, err := usecase.CreateTask(r.Context(), in)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	task := out.Task

	response := Response{
		ID:          task.ID,
		Name:        task.Name,
		Description: task.Description,
		Status:      string(task.Status),

		Constants:    task.Constants,
		EquationType: out.Equation.Type,
		CreatedAt:    task.CreatedAt,
	}

	render.JSON(w, response, http.StatusCreated)
}
