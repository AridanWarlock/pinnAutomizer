package authRefresh

import (
	"fmt"
	"net/http"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/AridanWarlock/pinnAutomizer/pkg/transport/http/response"
	"github.com/AridanWarlock/pinnAutomizer/pkg/transport/http/server"
)

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
		Method:  http.MethodPost,
		Path:    "/auth/refresh",
		Handler: h.Refresh,
	}
}

func (h *HttpHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.FromContext(ctx)
	rh := response.NewHandler(w, log)

	refreshToken, err := r.Cookie("refresh_token")
	if err != nil {
		rh.ErrorResponse(
			fmt.Errorf("%w: %v", errs.ErrAuthorizationFailed, err),
			"not found refresh token in cookies",
		)
		return
	}

	in := Input{
		RefreshTokenString: refreshToken.Value,
	}

	out, err := h.usecase.Refresh(ctx, in)
	if err != nil {
		rh.ErrorResponse(err, "failed to refresh access token")
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
