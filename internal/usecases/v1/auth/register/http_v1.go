package auth_register

import (
	"context"
	"net/http"

	core_http_request "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/request"
	core_http_response "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/response"
	core_http_server "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/server"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/google/uuid"
)

type Request struct {
	Login             string `json:"login"`
	Password          string `json:"password"`
	PasswordConfirmed string `json:"password_confirmed"`
}

type Response struct {
	ID    uuid.UUID `json:"ID"`
	Login string    `json:"login"`
}

type Service interface {
	Register(ctx context.Context, in Input) (Output, error)
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
		Method:  http.MethodPost,
		Path:    "/auth/register",
		Handler: h.Register,
	}
}

func (h *HttpHandler) Register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.FromContext(ctx)
	rh := core_http_response.NewHandler(w, log)

	var req Request
	if err := core_http_request.DecodeAndValidateRequest(r, &req); err != nil {
		rh.ErrorResponse(err, "failed to decode and validate HTTP request")
		return
	}

	in := Input{
		Login:             req.Login,
		Password:          req.Password,
		PasswordConfirmed: req.PasswordConfirmed,
	}

	out, err := h.service.Register(ctx, in)
	if err != nil {
		rh.ErrorResponse(err, "failed to register")
		return
	}

	res := Response{
		ID:    out.User.ID,
		Login: out.User.Login,
	}
	rh.JsonResponse(res, http.StatusCreated)
}
