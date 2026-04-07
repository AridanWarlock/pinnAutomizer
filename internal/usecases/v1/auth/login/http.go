package authLogin

import (
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/AridanWarlock/pinnAutomizer/internal/errs"
	httpRequest "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/request"
	httpResponse "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/response"
	httpServer "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/server"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
)

type Request struct {
	Login    string `json:"login" example:"Ivan Ivanov"`
	Password string `json:"password" example:"12345678"`
}

type Response struct {
	AccessToken string `json:"access_token" example:"very.long.access_token"`
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

// Login godoc
//
//	@Summary		Авторизация в системе
//	@Description	Авторизация в системе PINN Automizer
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request			body		Request						true	"Login тело запроса"
//	@Param			X-Fingerprint	header		string						true	"Sha-256 fingerprint"
//	@Success		200				{object}	Response					"Успешная авторизация"
//	@Failure		400				{object}	httpResponse.ErrorResponse	"Bad request"
//	@Failure		404				{object}	httpResponse.ErrorResponse	"Not found"
//	@Failure		500				{object}	httpResponse.ErrorResponse	"Internal server error"
//	@Router			/auth/login [post]
func (h *HttpHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.FromContext(ctx)
	rh := httpResponse.NewHandler(w, log)

	var req Request
	if err := httpRequest.DecodeAndValidateRequest(w, r, &req); err != nil {
		rh.ErrorResponse(err, "failed to decode and validate HTTP request")
		return
	}

	fingerprint, err := hex.DecodeString(r.Header.Get("X-Fingerprint"))
	if err != nil {
		rh.ErrorResponse(
			fmt.Errorf("%w: %v", errs.ErrInvalidArgument, err),
			"failed to decode fingerprint header",
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
		AccessToken: out.AccessTokenString,
	}

	rh.JsonResponse(response, http.StatusOK)
}
