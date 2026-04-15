package authLogout

import (
	"net/http"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/AridanWarlock/pinnAutomizer/pkg/transport/http/response"
	"github.com/AridanWarlock/pinnAutomizer/pkg/transport/http/server"
)

type HttpHandler struct {
	usecase Usecase
}

func NewHttpHandler(usecase Usecase) *HttpHandler {
	return &HttpHandler{
		usecase: usecase,
	}
}

func (h *HttpHandler) Route() server.Route {
	return server.Route{
		Method:   http.MethodPost,
		Path:     "/auth/logout",
		Handler:  h.Logout,
		IsPublic: false,
	}
}

func (h *HttpHandler) Logout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.FromContext(ctx)
	rh := response.NewHandler(w, log)

	err := h.usecase.Logout(ctx)
	if err != nil {
		rh.ErrorResponse(err, "failed to logout")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Path:     "/api/v1/auth/refresh",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	rh.EmptyResponse(http.StatusNoContent)
}
