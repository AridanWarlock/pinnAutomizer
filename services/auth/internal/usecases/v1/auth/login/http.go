package authLogin

import (
	"net/http"

	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/AridanWarlock/pinnAutomizer/pkg/transport/http/request"
	"github.com/AridanWarlock/pinnAutomizer/pkg/transport/http/response"
	"github.com/AridanWarlock/pinnAutomizer/pkg/transport/http/server"
)

type Request struct {
	Login    string `json:"login" validate:"required"`
	Password string `json:"password" validate:"required,min=5"`
}

type Response struct {
	AccessToken string `json:"access_token"`
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
		Method:   http.MethodPost,
		Path:     "/auth/login",
		Handler:  h.Login,
		IsPublic: true,
	}
}

func (h *HttpHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.FromContext(ctx)
	rh := response.NewHandler(w, log)

	var req Request
	if err := request.DecodeAndValidate(w, r, &req); err != nil {
		rh.ErrorResponse(err, "failed to decode and validate HTTP request")
		return
	}

	in := Input{
		Login:    req.Login,
		Password: req.Password,
	}

	out, err := h.usecase.Login(ctx, in)
	if err != nil {
		rh.ErrorResponse(err, "failed to login")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Path:     "/api/v1/auth/refresh",
		Value:    out.RefreshTokenString,
		Expires:  out.RefreshTokenExpiresAt,
		SameSite: http.SameSiteLaxMode,
		HttpOnly: true,
	})

	res := Response{
		AccessToken: string(out.AccessToken),
	}

	rh.JsonResponse(res, http.StatusOK)
}
