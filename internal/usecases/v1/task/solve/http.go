package tasksSolve

import (
	"net/http"

	httpRequest "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/request"
	httpResponse "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/response"
	httpServer "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/server"
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

func (h *HttpHandler) Route() httpServer.Route {
	return httpServer.Route{
		Method:  http.MethodGet,
		Path:    "/tasks/{id}/solve",
		Handler: h.SolveTask,
	}
}

func (h *HttpHandler) SolveTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.FromContext(ctx)
	userClaims := httpRequest.ClaimsFromContext(ctx)
	rh := httpResponse.NewHandler(w, log)

	var req Request
	if err := httpRequest.DecodeAndValidateRequest(w, r, &req); err != nil {
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
