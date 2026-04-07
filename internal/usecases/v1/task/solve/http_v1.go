package tasks_solve

import (
	"net/http"

	http_request "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/request"
	http_response "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/response"
	http_server "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/server"
	http_utils "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/utils"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/google/uuid"
)

type Request struct {
	TaskID    uuid.UUID      `json:"task_id"`
	Constants map[string]any `json:"constants"`
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
		Path:    "/tasks/{id}/solve",
		Handler: h.SolveTask,
	}
}

func (h *HttpHandler) SolveTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.FromContext(ctx)
	userClaims := http_utils.ClaimsFromContext(ctx)
	rh := http_response.NewHandler(w, log)

	var req Request
	if err := http_request.DecodeAndValidateRequest(w, r, &req); err != nil {
		log.Info().Err(err).Msg("parse json error")
		return
	}

	in := Input{
		TaskID:    req.TaskID,
		Constants: req.Constants,
		UserID:    userClaims.UserID,
	}

	err := h.usecase.SolveTask(ctx, in)
	if err != nil {
		rh.ErrorResponse(err, "failed to solve task")
		return
	}

	rh.EmptyResponse(http.StatusOK)
}
