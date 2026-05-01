package authLogin

import (
	"net/http"

	"github.com/AridanWarlock/pinnAutomizer/pkg/httpin"
	"github.com/AridanWarlock/pinnAutomizer/pkg/httpout"
	"github.com/AridanWarlock/pinnAutomizer/pkg/httpsrv"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
)

type Request struct {
	Login    string `json:"login" validate:"required"`
	Password string `json:"password" validate:"required,min=5"`
} // @name LoginRequest

type Response struct {
	AccessToken string `json:"access_token"`
} // @name LoginResponse

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
		Path:     "/auth/login",
		Handler:  h.Login,
		IsPublic: true,
	}
}

// Login 			godoc
//
//	@Summary		Вход в систему
//	@Description	Вход в систему PINN Automizer
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		Request						true	"LoginRequest тело запроса"
//	@Success		200		{object}	Response					"LoginResponse jwt access токен"
//	@Failure		400		{object}	httpout.ErrorResponse	"Bad request"
//	@Failure		401		{object}	httpout.ErrorResponse	"Unauthorized"
//	@Failure		500		{object}	httpout.ErrorResponse	"Internal server error"
//	@Router			/auth/login 	[post]
func (h *HttpHandler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.FromContext(ctx)
	rh := httpout.NewHandler(w, log)

	var req Request
	if err := httpin.DecodeAndValidate(w, r, &req); err != nil {
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
