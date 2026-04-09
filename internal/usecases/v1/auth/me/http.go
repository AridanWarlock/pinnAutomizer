package authMe

import (
	"net/http"

	httpRequest "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/request"
	httpResponse "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/response"
	httpServer "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/server"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/google/uuid"
)

type Response struct {
	ID    uuid.UUID `json:"id"`
	Login string    `json:"login"`
}

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
		Method:  http.MethodGet,
		Path:    "/auth/me",
		Handler: h.Me,
	}
}

func (h *HttpHandler) Me(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.FromContext(ctx)
	userClaims := httpRequest.ClaimsFromContext(ctx)
	rh := httpResponse.NewHandler(w, log)

	out, err := h.usecase.Me(ctx, Input{UserID: userClaims.UserID})
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
