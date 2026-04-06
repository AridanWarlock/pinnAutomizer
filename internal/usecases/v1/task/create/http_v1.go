package tasks_create

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
		Method:  http.MethodPost,
		Path:    "/tasks",
		Handler: h.CreateTask,
	}
}

// CreateTask 			godoc
// @Summary 	Создать задачу
// @Description Создать новую PINN задачу
// @Tags 		tasks
// @Accept 		json
// @Produce 	json
// @Param		request body 		Request 	true "CreateTask тело запроса"
// @Success 	201 	{object}	Response 	"Успешно созданная PINN задача"
// @Failure 	400		{object} 	http_v1.ErrorResponse 	"Bad request"
// @Failure		500 	{object} 	http_v1.ErrorResponse 	"Internal server error"
// @Router 		/tasks 	[post]
func (h *HttpHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
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
		Name: req.Name,

		Description: req.Description,
		Constants:   req.Constants,

		EquationType: req.EquationType,
		UserID:       userClaims.UserID,
	}

	out, err := h.usecase.CreateTask(ctx, in)
	if err != nil {
		rh.ErrorResponse(err, "failed to create task")
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
	rh.JsonResponse(response, http.StatusCreated)
}
