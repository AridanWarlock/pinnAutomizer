package authRefresh

import (
	"fmt"
	"net/http"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/httpout"
	"github.com/AridanWarlock/pinnAutomizer/pkg/httpsrv"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
)

type Response struct {
	AccessToken string `json:"access_token"`
} // @name RefreshResponse

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
		Method:   http.MethodPost,
		Path:     "/auth/refresh",
		Handler:  h.Refresh,
		IsPublic: true,
	}
}

// Refresh 			godoc
//
//	@Summary		Обновление access токена
//	@Description	Обновление access токена по refresh токену
//	@Tags			auth
//	@Produce		json
//	@Success		200		{object}	Response					"RefreshResponse новый jwt access токен"
//	@Failure		401		{object}	httpout.ErrorResponse	"Unauthorized"
//	@Failure		500		{object}	httpout.ErrorResponse	"Internal server error"
//	@Router			/auth/refresh 	[post]
func (h *HttpHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.FromContext(ctx)
	rh := httpout.NewHandler(w, log)

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
