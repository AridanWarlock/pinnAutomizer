package core_http_response

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	core_errors "github.com/AridanWarlock/pinnAutomizer/internal/errors"
	"github.com/rs/zerolog"
)

type Handler struct {
	rw  http.ResponseWriter
	log zerolog.Logger
}

func NewHandler(w http.ResponseWriter, log zerolog.Logger) *Handler {
	return &Handler{
		rw:  w,
		log: log,
	}
}

func (h *Handler) JsonResponse(body any, statusCode int) {
	h.rw.Header().Set("Content-Type", "application/json")
	h.rw.WriteHeader(statusCode)

	if err := json.NewEncoder(h.rw).Encode(body); err != nil {
		h.log.Error().Err(err).Msg("write HTTP response")
	}
}

func (h *Handler) EmptyResponse(statusCode int) {
	h.rw.WriteHeader(statusCode)
}

func (h *Handler) ErrorResponse(err error, msg string) {
	var (
		statusCode int
		log        *zerolog.Event
	)

	switch {
	case errors.Is(err, core_errors.ErrAuthorizationFailed):
		statusCode = http.StatusUnauthorized
		log = h.log.Debug()

	case errors.Is(err, core_errors.ErrInvalidArgument):
		statusCode = http.StatusBadRequest
		log = h.log.Warn()

	case errors.Is(err, core_errors.ErrNotFound):
		statusCode = http.StatusNotFound
		log = h.log.Debug()

	case errors.Is(err, core_errors.ErrConflict):
		statusCode = http.StatusConflict
		log = h.log.Warn()

	default:
		statusCode = http.StatusInternalServerError
		log = h.log.Error()
	}

	log.Err(err).Msg(msg)
	h.errorResponse(statusCode, err, msg)
}

func (h *Handler) PanicResponse(p any, msg string) {
	statusCode := http.StatusInternalServerError
	err := fmt.Errorf("unexpected panic: %v", p)
	h.log.Error().Err(err).Msg(msg)

	h.errorResponse(statusCode, err, msg)
}

func (h *Handler) errorResponse(statusCode int, err error, msg string) {
	response := domain.NewErrorMessage(msg, err)
	h.JsonResponse(response, statusCode)
}
