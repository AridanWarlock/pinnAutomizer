package auth_login

import (
	"context"
	"encoding/hex"
	"net/http"

	core_http_request "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/request"
	core_http_response "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/response"
	core_http_server "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/server"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
)

type Request struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type Response struct {
	AccessToken string `json:"access_token"`
}

type Service interface {
	Login(ctx context.Context, in Input) (Output, error)
}

type HttpHandler struct {
	service Service
}

func NewHttpHandler(service Service) *HttpHandler {
	return &HttpHandler{
		service: service,
	}
}

func (h *HttpHandler) Route() core_http_server.Route {
	return core_http_server.Route{
		Method:  http.MethodPost,
		Path:    "/auth/login",
		Handler: h.Login,
	}
}

func (h *HttpHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.FromContext(ctx)
	rh := core_http_response.NewHandler(w, log)

	var req Request
	if err := core_http_request.DecodeAndValidateRequest(r, &req); err != nil {
		rh.ErrorResponse(err, "failed to decode and validate HTTP request")
		return
	}

	fingerprint, err := hex.DecodeString(r.Header.Get("X-Fingerprint"))
	if err != nil {
		rh.ErrorResponse(err, "failed to decode and fingerprint")
		return
	}

	in := Input{
		Login:       req.Login,
		Password:    req.Password,
		Fingerprint: fingerprint,
	}

	out, err := h.service.Login(ctx, in)
	if err != nil {
		rh.ErrorResponse(err, "failed to login")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refreshToken",
		Path:     "/api/v1/auth/refresh",
		Value:    out.RefreshTokenString,
		Expires:  out.RefreshTokenExpiresAt,
		SameSite: http.SameSiteStrictMode,
		HttpOnly: true,
	})

	response := Response{
		AccessToken: out.AccessTokenString,
	}

	rh.JsonResponse(response, http.StatusOK)
}
