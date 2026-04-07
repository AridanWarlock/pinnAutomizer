package auth_me

import (
	"net/http"

	http_response "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/response"
	http_server "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/server"
	http_utils "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/utils"
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

func (h *HttpHandler) Route() http_server.Route {
	return http_server.Route{
		Method:  http.MethodGet,
		Path:    "/auth/me",
		Handler: h.Me,
	}
}

func (h *HttpHandler) Me(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.FromContext(ctx)
	userClaims := http_utils.ClaimsFromContext(ctx)
	rh := http_response.NewHandler(w, log)

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
