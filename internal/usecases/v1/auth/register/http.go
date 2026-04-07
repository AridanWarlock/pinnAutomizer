package authRegister

import (
	"net/http"

	httpRequest "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/request"
	httpResponse "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/response"
	httpServer "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/server"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/google/uuid"
)

type Request struct {
	Login             string `json:"login"`
	Password          string `json:"password"`
	PasswordConfirmed string `json:"password_confirmed"`
}

type Response struct {
	ID    uuid.UUID `json:"ID"`
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
		Method:  http.MethodPost,
		Path:    "/auth/register",
		Handler: h.Register,
	}
}

func (h *HttpHandler) Register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.FromContext(ctx)
	rh := httpResponse.NewHandler(w, log)

	var req Request
	if err := httpRequest.DecodeAndValidateRequest(w, r, &req); err != nil {
		rh.ErrorResponse(err, "failed to decode and validate HTTP request")
		return
	}

	in := Input{
		Login:             req.Login,
		Password:          req.Password,
		PasswordConfirmed: req.PasswordConfirmed,
	}

	out, err := h.usecase.Register(ctx, in)
	if err != nil {
		rh.ErrorResponse(err, "failed to register")
		return
	}

	res := Response{
		ID:    out.User.ID,
		Login: out.User.Login,
	}
	rh.JsonResponse(res, http.StatusCreated)
}
