package tasksGet

import (
	"github.com/AridanWarlock/pinnAutomizer/pkg/httpin"
	"github.com/AridanWarlock/pinnAutomizer/pkg/httpout"
	"github.com/AridanWarlock/pinnAutomizer/pkg/httpsrv"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/usecases/v1/tasks"
	"net/http"
)

type Response struct {
	Tasks []tasks.TaskDto `json:"tasks"`
	Total int             `json:"total"`
} // @name GetTasksResponse

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
		Path:     "/tasks",
		Handler:  h.GetTasks,
		IsPublic: false,
	}
}

// GetTasks 			godoc
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
func (h *HttpHandler) GetTasks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.FromContext(ctx)
	rh := httpout.NewHandler(w, log)

	options, err := httpin.ParsePaginationOptions(r)
	if err != nil {
		rh.ErrorResponse(err, "failed to decode and validate HTTP request")
		return
	}

	in := Input{
		Pagination: options,
	}

	out, err := h.usecase.GetTasks(ctx, in)
	if err != nil {
		rh.ErrorResponse(err, "failed to get tasks info")
		return
	}

	taskModels := make([]tasks.TaskDto, 0, len(out.Tasks))
	for _, task := range out.Tasks {
		taskModels = append(taskModels, tasks.ToDto(task))
	}

	res := Response{
		Tasks: taskModels,
		Total: out.Total,
	}
	rh.JsonResponse(res, http.StatusOK)
}
