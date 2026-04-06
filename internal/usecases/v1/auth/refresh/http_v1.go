package auth_refresh

import (
	"context"
	"net/http"

	core_http_response "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/response"
	core_http_server "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/server"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
)

type Response struct {
	AccessToken string `json:"accessToken"`
}

type Service interface {
	Refresh(ctx context.Context, in Input) (Output, error)
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
		Path:    "/auth/refresh",
		Handler: h.Refresh,
	}
}

func (h *HttpHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.FromContext(ctx)
	rh := core_http_response.NewHandler(w, log)

	refreshToken, err := r.Cookie("refreshToken")
	if err != nil {
		rh.ErrorResponse(err, "not found refresh token in cookies")
		return
	}

	in := Input{RefreshTokenString: refreshToken.Value}

	out, err := h.service.Refresh(ctx, in)
	if err != nil {
		rh.ErrorResponse(err, "failed to refresh access token")
		return
	}

	res := Response{
		AccessToken: string(out.AccessToken),
	}
	rh.JsonResponse(res, http.StatusOK)
}
