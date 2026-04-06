package auth_refresh

import (
	"net/http"

	"github.com/AridanWarlock/pinnAutomizer/internal/transport/http/response"
	"github.com/AridanWarlock/pinnAutomizer/internal/transport/http/server"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
)

type Response struct {
	AccessToken string `json:"accessToken"`
}

type HttpHandler struct {
	usecase Usecase
}

func NewHttpHandler(usecase Usecase) *HttpHandler {
	return &HttpHandler{
		usecase: usecase,
	}
}

func (h *HttpHandler) Route() http_server.Route {
	return http_server.Route{
		Method:  http.MethodPost,
		Path:    "/auth/refresh",
		Handler: h.Refresh,
	}
}

func (h *HttpHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.FromContext(ctx)
	rh := http_response.NewHandler(w, log)

	refreshToken, err := r.Cookie("refreshToken")
	if err != nil {
		rh.ErrorResponse(err, "not found refresh token in cookies")
		return
	}

	in := Input{RefreshTokenString: refreshToken.Value}

	out, err := h.usecase.Refresh(ctx, in)
	if err != nil {
		rh.ErrorResponse(err, "failed to refresh access token")
		return
	}

	res := Response{
		AccessToken: string(out.AccessToken),
	}
	rh.JsonResponse(res, http.StatusOK)
}
