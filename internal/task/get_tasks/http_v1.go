package get_tasks

import (
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"net/http"
	"pinnAutomizer/internal/middleware/auth"
	"pinnAutomizer/pkg/json"
	"pinnAutomizer/pkg/render"
	"time"
)

type Request struct {
	IDs []uuid.UUID `json:"ids"`
}

type TaskModel struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`

	Status    string         `json:"status"`
	Constants map[string]any `json:"constants"`

	EquationType string `json:"equation_type"`

	CreatedAt time.Time `json:"created_at"`
}

type Response struct {
	Tasks []TaskModel `json:"tasks"`
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
		IDs:    req.IDs,
		UserID: userID,
	}

	out, err := usecase.GetTasks(r.Context(), in)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	tasks := out.TasksToEquation
	taskModels := make([]TaskModel, 0, len(tasks))

	for task, equation := range tasks {
		taskModel := TaskModel{
			ID:          task.ID,
			Name:        task.Name,
			Description: task.Description,
			Status:      string(task.Status),

			Constants:    task.Constants,
			EquationType: equation.Type,
			CreatedAt:    task.CreatedAt,
		}

		taskModels = append(taskModels, taskModel)
	}

	response := Response{
		Tasks: taskModels,
	}

	render.JSON(w, response, http.StatusCreated)
}
