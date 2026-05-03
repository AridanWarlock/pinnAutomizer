package tasksResults

import (
	"net/http"

	"github.com/AridanWarlock/pinnAutomizer/pkg/httpin"
	"github.com/AridanWarlock/pinnAutomizer/pkg/httpout"
	"github.com/AridanWarlock/pinnAutomizer/pkg/httpsrv"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
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
		Method:   http.MethodGet,
		Path:     "/tasks/{id}/download",
		Handler:  h.DownloadTaskResults,
		IsPublic: false,
	}
}

// DownloadTaskResults 			godoc
//
//		@Summary		Получить статус задач
//		@Description	Получить статус  PINN задач по id
//		@Tags			tasks
//		@Accept			json
//		@Produce		json
//	 @Param          limit   query     int     false  "Количество записей"  default(100) minimum(1) maximum(100)
//	 @Param          offset  query     int     false  "Смещение"            default(0)   minimum(0)
//	 @Param          sort    query     string  false  "Поле сортировки"     Enums(created_at, name) default(created_at)
//	 @Param          order   query     string  false  "Направление сортировки"         Enums(asc, desc) default(desc)
//		@Success		200		{object}	Response					"GetTasksResponse информация о задачах"
//		@Failure		400		{object}	httpout.ErrorResponse	"Bad request"
//		@Failure		500		{object}	httpout.ErrorResponse	"Internal server error"
//		@Router			/tasks 	[get]
func (h *HttpHandler) DownloadTaskResults(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.FromContext(ctx)
	rh := httpout.NewHandler(w, log)

	taskID, err := httpin.PathUuid(r, "id")
	if err != nil {
		rh.ErrorResponse(err, "failed to get task id from path")
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", "attachment; filename=\"results.zip\"")

	in := Input{
		TaskID:       taskID,
		Writer:       w,
		DownloadType: DownloadType(r.URL.Query().Get("type")),
	}

	err = h.usecase.DownloadTaskResults(ctx, in)
	if err != nil {
		rh.ErrorResponse(err, "failed to download task results")
		return
	}

	rh.EmptyResponse(http.StatusOK)
}
