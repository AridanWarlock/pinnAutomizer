package tasksSolve

import (
	"net/http"

	"github.com/AridanWarlock/pinnAutomizer/pkg/httpin"
	"github.com/AridanWarlock/pinnAutomizer/pkg/httpout"
	"github.com/AridanWarlock/pinnAutomizer/pkg/httpsrv"
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

func (h *HttpHandler) Route() httpsrv.Route {
	return httpsrv.Route{
		Method:  http.MethodGet,
		Path:    "/tasks/{id}/solve",
		Handler: h.SolveTask,
	}
}

func (h *HttpHandler) SolveTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.FromContext(ctx)
	rh := httpout.NewHandler(w, log)

	var req Request
	if err := httpin.DecodeAndValidate(w, r, &req); err != nil {
		rh.ErrorResponse(err, "failed to decode and validate HTTP request")
		return
	}

	in := Input{
		TaskID:    req.TaskID,
		Constants: req.Constants,
	}

	err := h.usecase.SolveTask(ctx, in)
	if err != nil {
		rh.ErrorResponse(err, "failed to solve task")
		return
	}

	rh.EmptyResponse(http.StatusOK)
}
