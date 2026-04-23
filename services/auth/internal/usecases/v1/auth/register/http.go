package authRegister

import (
	"net/http"

	"github.com/AridanWarlock/pinnAutomizer/pkg/httpin"
	"github.com/AridanWarlock/pinnAutomizer/pkg/httpout"
	"github.com/AridanWarlock/pinnAutomizer/pkg/httpsrv"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/google/uuid"
)

type Request struct {
	Login             string `json:"login" validate:"required"`
	Password          string `json:"password" validate:"required,min=5"`
	PasswordConfirmed string `json:"password_confirmed" validate:"required,eqfield=Password"`
} // @name RegisterRequest

type Response struct {
	ID    uuid.UUID `json:"id"`
	Login string    `json:"login"`
} // @name RegisterResponse

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
		Path:     "/auth/register",
		Handler:  h.Register,
		IsPublic: true,
	}
}

// Register 			godoc
//
//	@Summary		Регистрация в системе
//	@Description	Регистрация в системе PINN Automizer
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		Request						true	"RegisterRequest тело запроса"
//	@Success		201		{object}	Response					"UserResponse новый пользователь"
//	@Failure		400		{object}	httpout.ErrorResponse	"Bad request"
//	@Failure		500		{object}	httpout.ErrorResponse	"Internal server error"
//	@Router			/auth/register 	[post]
func (h *HttpHandler) Register(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := logger.FromContext(ctx)
	rh := httpout.NewHandler(w, log)

	var req Request
	if err := httpin.DecodeAndValidate(w, r, &req); err != nil {
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
