package authMe

import (
	"net/http"

	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/AridanWarlock/pinnAutomizer/pkg/transport/http/response"
	"github.com/AridanWarlock/pinnAutomizer/pkg/transport/http/server"
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

func (h *HttpHandler) Route() server.Route {
	return server.Route{
		Method:   http.MethodGet,
		Path:     "/auth/me",
		Handler:  h.Me,
		IsPublic: false,
	}
}

func (h *HttpHandler) Me(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.FromContext(ctx)
	rh := response.NewHandler(w, log)

	out, err := h.usecase.Me(ctx)
	if err != nil {
		rh.ErrorResponse(err, "failed to get me info")
		return
	}

	res := Response{
		ID:    out.User.ID,
		Login: out.User.Login,
	}
	rh.JsonResponse(res, http.StatusOK)
}
