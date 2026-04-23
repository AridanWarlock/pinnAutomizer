package authMe

import (
	"net/http"

	"github.com/AridanWarlock/pinnAutomizer/pkg/httpout"
	"github.com/AridanWarlock/pinnAutomizer/pkg/httpsrv"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/google/uuid"
)

type Response struct {
	ID    uuid.UUID `json:"id"`
	Login string    `json:"login"`
} // @name MeResponse

type HttpHandler struct {
	usecase Usecase
}

func NewHttpHandler(usecase Usecase) *HttpHandler {
	return &HttpHandler{
		usecase: usecase,
	}
}

func (h *HttpHandler) Route() httpsrv.Route {
	return httpsrv.Route{
		Method:   http.MethodGet,
		Path:     "/auth/me",
		Handler:  h.Me,
		IsPublic: false,
	}
}

// Me 			godoc
//
//	@Summary		Информация о пользователе
//	@Description	Информация о пользователе из текущей сессии
//	@Tags			auth
//	@Produce		json
//	@Success		200		{object}	Response					"MeResponse информация о пользователе"
//	@Failure		401		{object}	httpout.ErrorResponse	"Unauthorized"
//	@Failure		500		{object}	httpout.ErrorResponse	"Internal server error"
//	@Router			/auth/me 	[get]
func (h *HttpHandler) Me(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.FromContext(ctx)
	rh := httpout.NewHandler(w, log)

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
