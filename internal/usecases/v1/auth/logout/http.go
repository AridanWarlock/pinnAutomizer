package authLogout

import (
	"net/http"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	httpResponse "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/response"
	httpServer "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/server"
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

func (h *HttpHandler) Route() httpServer.Route {
	return httpServer.Route{
		Method:  http.MethodPost,
		Path:    "/auth/logout",
		Handler: h.Logout,
	}
}

func (h *HttpHandler) Logout(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.FromContext(ctx)
	rh := httpResponse.NewHandler(w, log)

	_ = domain.AuditInfoFromContext(ctx)
	log.Debug().Msg("in http")

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
