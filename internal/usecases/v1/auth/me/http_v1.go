package auth_me

import (
	"context"
	"net/http"

	core_http "github.com/AridanWarlock/pinnAutomizer/internal/transport/http"
	core_http_response "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/response"
	core_http_server "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/server"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/google/uuid"
)

type Response struct {
	ID    uuid.UUID `json:"id"`
	Login string    `json:"login"`
}

type Service interface {
	Me(ctx context.Context, in Input) (Output, error)
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
		Path:    "/auth/me",
		Handler: h.Me,
	}
}

func (h *HttpHandler) Me(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.FromContext(ctx)
	userClaims := core_http.ClaimsFromContext(ctx)
	rh := core_http_response.NewHandler(w, log)

	out, err := h.service.Me(ctx, Input{UserID: userClaims.UserID})
	if err != nil {
		rh.ErrorResponse(err, "failed to get me info")
		return
	}

	res := Response{
		ID:    out.UserID,
		Login: out.Login,
	}
	rh.JsonResponse(res, http.StatusOK)
}
