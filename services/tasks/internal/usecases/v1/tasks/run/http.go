package tasksRun

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/httpin"
	"github.com/AridanWarlock/pinnAutomizer/pkg/httpout"
	"github.com/AridanWarlock/pinnAutomizer/pkg/httpsrv"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
)

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
		Method:  http.MethodPost,
		Path:    "/tasks/{id}/run",
		Handler: h.RunTask,
	}
}

// RunTask 			godoc
//
//		@Summary		Отправить задачу на исполнение
//		@Description	Отправить PINN задачу на исполнение
//		@Tags			tasks
//		@Accept			json
//	    @Param          id   path      string  true  "ID задачи (UUID)" format(uuid)
//		@Success		204		"Задача успешно отправлена в очередь"
//		@Failure		400		{object}	httpout.ErrorResponse	"Bad request"
//		@Failure		500		{object}	httpout.ErrorResponse	"Internal server error"
//		@Router			/tasks/{id}/solve 	[post]
func (h *HttpHandler) RunTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.FromContext(ctx)
	rh := httpout.NewHandler(w, log)

	taskID, err := httpin.PathUuid(r, "id")
	if err != nil {
		rh.ErrorResponse(err, "failed to decode and validate HTTP request")
		return
	}

	in := Input{
		TaskID: taskID,
	}

	err = h.usecase.RunTask(ctx, in)
	if err != nil {
		if errors.Is(err, domain.ErrTaskAlreadyStarted) {
			err = fmt.Errorf("%w: %v", errs.ErrInvalidArgument, err)
		}

		rh.ErrorResponse(err, "failed to start train task")
		return
	}

	rh.EmptyResponse(http.StatusNoContent)
}
