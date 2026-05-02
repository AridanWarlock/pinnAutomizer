package tasksCreate

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/httpout"
	"github.com/AridanWarlock/pinnAutomizer/pkg/httpsrv"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
	"github.com/google/uuid"
)

const maxUploadSize = 200 << 20 // 200 MB

type TaskRequest struct {
	Name        string  `json:"name" example:"Теплопроводность стержня"`
	Description *string `json:"description,omitempty" example:"Теплопроводность железного стержня"`

	Mode string `json:"mode" example:"train"`
}

type Response struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`

	Mode string `json:"mode"`

	Status string `json:"status"`

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

func (h *HttpHandler) Route() httpsrv.Route {
	return httpsrv.Route{
		Method:   http.MethodPost,
		Path:     "/tasks",
		Handler:  h.CreateTask,
		IsPublic: false,
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
//	@Failure		400		{object}	response.ErrorResponse	"Bad request"
//	@Failure		500		{object}	response.ErrorResponse	"Internal server error"
//	@Router			/tasks 	[post]
func (h *HttpHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.FromContext(ctx)
	rh := httpout.NewHandler(w, log)

	jsonStr := strings.TrimSpace(r.FormValue("data"))
	if jsonStr == "" {
		rh.ErrorResponse(errs.ErrInvalidArgument, "missing 'data' field with JSON metadata")
		return
	}

	var taskReq TaskRequest
	if err := json.Unmarshal([]byte(jsonStr), &taskReq); err != nil {
		rh.ErrorResponse(
			fmt.Errorf(
				"%w: invalid JSON in 'data' field: %v",
				errs.ErrInvalidArgument,
				err,
			),
			"failed to decode and validate HTTP request",
		)

		return
	}

	mode, err := domain.NewTaskMode(taskReq.Mode)
	if err != nil {
		rh.ErrorResponse(
			fmt.Errorf("%w: %v", errs.ErrInvalidArgument, err),
			"failed to decode and validate HTTP request",
		)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		rh.ErrorResponse(
			fmt.Errorf("%w: failed to parse multipart form: %v", errs.ErrInvalidArgument, err),
			"request body must be multipart/form-data",
		)
		return
	}
	defer func() {
		if err := r.MultipartForm.RemoveAll(); err != nil {
			log.Err(err).Msg("remove multipart error")
		}
	}()

	files, err := h.collectFiles(r, mode)
	if err != nil {
		rh.ErrorResponse(
			err,
			"failed to collect files from multipart/form-data",
		)
		return
	}
	defer func() {
		for _, file := range files {
			_ = file.File.Close()
		}
	}()

	in := Input{
		Name:        taskReq.Name,
		Description: taskReq.Description,

		Mode: mode,

		Files: files,
	}

	out, err := h.usecase.CreateTask(ctx, in)
	if err != nil {
		rh.ErrorResponse(err, "failed to create task")
		return
	}

	task := out.Task

	res := Response{
		ID:          task.ID,
		Name:        task.Name,
		Description: task.Description,

		Mode: string(task.Mode),

		Status: string(task.Status),

		CreatedAt: task.CreatedAt,
	}
	rh.JsonResponse(res, http.StatusCreated)
}

func (h *HttpHandler) collectFiles(r *http.Request, mode domain.TaskMode) ([]domain.TaskFile, error) {
	var files []domain.TaskFile

	for _, filename := range mode.RequiredFiles() {
		file, _, err := r.FormFile(filename)
		if err != nil {
			return nil, fmt.Errorf(
				"%w: failed to get requred file=%s: %v",
				errs.ErrInvalidArgument,
				filename,
				err,
			)
		}

		files = append(files, domain.TaskFile{
			Name: filename,
			File: file,
		})
	}

	return files, nil
}
