package auth_logout

import (
	"net/http"

	core_http "github.com/AridanWarlock/pinnAutomizer/internal/transport/http"
	core_http_response "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/response"
	"github.com/AridanWarlock/pinnAutomizer/internal/transport/http/server"
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

func (h *HttpHandler) Route() core_http_server.Route {
	return core_http_server.Route{
		Method:  http.MethodPost,
		Path:    "/auth/logout",
		Handler: h.Logout,
	}
}

func (h *HttpHandler) Logout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.FromContext(ctx)
	userClaims := core_http.ClaimsFromContext(ctx)
	rh := core_http_response.NewHandler(w, log)

	err := h.usecase.Logout(ctx, Input{ID: userClaims.UserID})
	if err != nil {
		rh.ErrorResponse(err, "failed to logout")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refreshToken",
		Path:     "/api/v1/auøth/refresh",
		MaxAge:   -1,
		HttpOnly: true,
	})

	rh.EmptyResponse(http.StatusNoContent)
}
