package tasks_solve

import (
	"context"
	"net/http"

	core_http "github.com/AridanWarlock/pinnAutomizer/internal/transport/http"
	core_http_request "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/request"
	core_http_response "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/response"
	core_http_server "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/server"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/google/uuid"
)

type Request struct {
	TaskID    uuid.UUID      `json:"task_id"`
	Constants map[string]any `json:"constants"`
}

type Service interface {
	SolveTask(ctx context.Context, in Input) error
}

type HttpHandler struct {
	service Service
}

func NewHttpHandler(service Service) *HttpHandler {
	return &HttpHandler{
		service: service,
	}
}

func (h *HttpHandler) Route() core_http_server.Route {
	return core_http_server.Route{
		Method:  http.MethodGet,
		Path:    "/tasks/{id}/solve",
		Handler: h.SolveTask,
	}
}

func (h *HttpHandler) SolveTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.FromContext(ctx)
	userClaims := core_http.ClaimsFromContext(ctx)
	rh := core_http_response.NewHandler(w, log)

	var req Request
	if err := core_http_request.DecodeAndValidateRequest(r, &req); err != nil {
		log.Info().Err(err).Msg("parse json error")
		return
	}

	in := Input{
		TaskID:    req.TaskID,
		Constants: req.Constants,
		UserID:    userClaims.UserID,
	}

	err := h.service.SolveTask(ctx, in)
	if err != nil {
		rh.ErrorResponse(err, "failed to solve task")
		return
	}

	rh.EmptyResponse(http.StatusOK)
}
