package tasksGet

import (
	"net/http"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pkg/httpin"
	"github.com/AridanWarlock/pinnAutomizer/pkg/httpout"
	"github.com/AridanWarlock/pinnAutomizer/pkg/httpsrv"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/google/uuid"
)

type taskDto struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`

	Status    string         `json:"status"`
	Constants map[string]any `json:"constants"`

	EquationType string `json:"equation_type"`

	CreatedAt time.Time `json:"created_at"`
} // @name TaskDTO

type Response struct {
	Tasks []taskDto `json:"tasks"`
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

	res := Response{
		Tasks: taskModels,
	}
	rh.JsonResponse(res, http.StatusOK)
}
