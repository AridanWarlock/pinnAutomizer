package authLogin

import (
	"fmt"
	"net/http"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/internal/errs"
	httpRequest "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/request"
	httpResponse "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/response"
	httpServer "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/server"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
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

func (h *HttpHandler) Route() httpServer.Route {
	return httpServer.Route{
		Method:  http.MethodPost,
		Path:    "/auth/login",
		Handler: h.Login,
	}
}

func (h *HttpHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.FromContext(ctx)
	rh := httpResponse.NewHandler(w, log)

	var req Request
	if err := httpRequest.DecodeAndValidateRequest(w, r, &req); err != nil {
		rh.ErrorResponse(err, "failed to decode and validate HTTP request")
		return
	}

	fingerprint, err := domain.NewFingerprintFromHex(r.Header.Get("X-Fingerprint"))
	if err != nil {
		rh.ErrorResponse(
			fmt.Errorf("%w: decode fingerprint header", errs.ErrInvalidArgument),
			"failed to decode and fingerprint",
		)
		return
	}

	in := Input{
		Login:       req.Login,
		Password:    req.Password,
		Fingerprint: fingerprint,
	}

	out, err := h.usecase.Login(ctx, in)
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
		AccessToken: string(out.AccessToken),
	}

	rh.JsonResponse(response, http.StatusOK)
}
