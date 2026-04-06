package tasks_get

import (
	"net/http"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/internal/transport/http/request"
	"github.com/AridanWarlock/pinnAutomizer/internal/transport/http/response"
	"github.com/AridanWarlock/pinnAutomizer/internal/transport/http/server"
	"github.com/AridanWarlock/pinnAutomizer/internal/transport/http/utils"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/google/uuid"
)

type Request struct {
	IDs []uuid.UUID `json:"ids"`
}

type taskDto struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`

	Status    string         `json:"status"`
	Constants map[string]any `json:"constants"`

	EquationType string `json:"equation_type"`

	CreatedAt time.Time `json:"created_at"`
}

type Response struct {
	Tasks []taskDto `json:"tasks"`
}

type HttpHandler struct {
	usecase Usecase
}

func NewHttpHandler(usecase Usecase) *HttpHandler {
	return &HttpHandler{
		usecase: usecase,
	}
}

func (h *HttpHandler) Route() http_server.Route {
	return http_server.Route{
		Method:  http.MethodGet,
		Path:    "/tasks",
		Handler: h.GetTasks,
	}
}

func (h *HttpHandler) GetTasks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.FromContext(ctx)
	userClaims := http_utils.ClaimsFromContext(ctx)
	rh := http_response.NewHandler(w, log)

	var req Request
	if err := http_request.DecodeAndValidateRequest(w, r, &req); err != nil {
		rh.ErrorResponse(err, "failed to decode and validate HTTP request")
		return
	}

	in := Input{
		IDs:    req.IDs,
		UserID: userClaims.UserID,
	}

	out, err := h.usecase.GetTasks(ctx, in)
	if err != nil {
		rh.ErrorResponse(err, "failed to get tasks info")
		return
	}

	tasks := out.TasksToEquation
	taskModels := make([]taskDto, 0, len(tasks))

	for task, equation := range tasks {
		taskModel := taskDto{
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
	rh.JsonResponse(response, http.StatusOK)
}
