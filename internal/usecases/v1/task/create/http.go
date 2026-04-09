package tasksCreate

import (
	"net/http"
	"time"

	httpRequest "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/request"
	httpResponse "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/response"
	httpServer "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/server"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/google/uuid"
)

type Request struct {
	Name         string         `json:"name" example:"Теплопроводность стержня"`
	Description  string         `json:"description" example:"Теплопроводность железного стержня"`
	Constants    map[string]any `json:"constants"`
	EquationType string         `json:"equation_type" example:"heat"`
} //	@name	CreateTaskRequest

type Response struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`

	Status    string         `json:"status"`
	Constants map[string]any `json:"constants"`

	EquationType string `json:"equation_type"`

	CreatedAt time.Time `json:"created_at"`
} //	@name	CreateTaskResponse

type HttpHandler struct {
	usecase Usecase
}

func NewHttpHandler(usecase Usecase) *HttpHandler {
	return &HttpHandler{
		usecase: usecase,
	}
}

func (h *HttpHandler) Route() httpServer.Route {
	return httpServer.Route{
		Method:  http.MethodPost,
		Path:    "/tasks",
		Handler: h.CreateTask,
	}
}

// CreateTask 			godoc
//
//	@Summary		Создать задачу
//	@Description	Создать новую PINN задачу
//	@Tags			tasks
//	@Accept			json
//	@Produce		json
//	@Param			request	body		Request						true	"CreateTask тело запроса"
//	@Success		201		{object}	Response					"Успешно созданная PINN задача"
//	@Failure		400		{object}	http_response.ErrorResponse	"Bad request"
//	@Failure		500		{object}	http_response.ErrorResponse	"Internal server error"
//	@Router			/tasks 	[post]
func (h *HttpHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.FromContext(ctx)
	userClaims := httpRequest.ClaimsFromContext(ctx)
	rh := httpResponse.NewHandler(w, log)

	var req Request
	if err := httpRequest.DecodeAndValidateRequest(w, r, &req); err != nil {
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
