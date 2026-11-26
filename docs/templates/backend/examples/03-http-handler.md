# HTTP-handler и транспортные helpers

Снимок исходников: `4b0892e9eebf16c3f915c047692f466eee3fdd78`. Handler вызывает тот же register usecase,
который приведён в 02-usecase-and-di.md.

## Поток запроса

```text
net/http → router → middleware → HttpHandler.Register
         → DecodeAndValidate(Request) → Input → Usecase.Register
         → Output → Response → JsonResponse
```

Handler владеет транспортными DTO, ограничениями тела/декодированием, выбором
HTTP-ответа. Он не вызывает Postgres напрямую. Собственные domain-объекты handler
не конструирует вместо usecase и состояние в БД не меняет.

### `services/auth/internal/usecases/v1/auth/register/http.go`

```go
package authRegister

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/pkg/httpin"
	"github.com/AridanWarlock/pinnAutomizer/pkg/httpout"
	"github.com/AridanWarlock/pinnAutomizer/pkg/httpsrv"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
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
```

## Декодирование

Публичный helper ниже использует приватный decodeRequestBody из
`pkg/httpin/decoder.go`: лимит тела 1 MiB, json.Decoder, DisallowUnknownFields,
разбор синтаксических ошибок/типов/пустого тела. Приватный decoder не включён в
выборку. При переносе реализацию нужно написать или взять отдельно вместе с тестами
лимита, пустого/неверного JSON и хвоста после первого документа. Это не готовая
standalone-библиотека с замкнутыми зависимостями.

### `pkg/httpin/decode.go`

```go
package httpin

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"
)

type validatable interface {
	Validate() error
}

func DecodeAndValidate(w http.ResponseWriter, r *http.Request, dst any) error {
	err := decodeRequestBody(w, r, dst)
	switch {
	case err == nil:
	case errors.Is(err, ErrBadJson):
		return fmt.Errorf(
			"decode json: %v: %w",
			err,
			errs.ErrInvalidArgument,
		)
	case errors.Is(err, ErrEntityToLarge):
		return fmt.Errorf("decode json: %w: ", err)
	default:
		return fmt.Errorf("unexpected decode json error: %w", err)
	}

	if v, ok := dst.(validatable); ok {
		err = v.Validate()
	} else {
		err = validate.V.Struct(dst)
	}

	if err != nil {
		return fmt.Errorf(
			"request validation: %w: %v",
			errs.ErrInvalidArgument,
			err,
		)
	}

	return nil
}
```

## Ответы и классификация ошибок

### `pkg/httpout/handler.go`

```go
package httpout

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/rs/zerolog"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
)

type Handler struct {
	rw  http.ResponseWriter
	log zerolog.Logger
}

func NewHandler(w http.ResponseWriter, log zerolog.Logger) *Handler {
	return &Handler{
		rw:  w,
		log: log,
	}
}

func (h *Handler) JsonResponse(body any, statusCode int) {
	h.rw.Header().Set("Content-Type", "application/json")
	h.rw.WriteHeader(statusCode)

	if err := json.NewEncoder(h.rw).Encode(body); err != nil {
		h.log.Error().Err(err).Msg("write HTTP response")
	}
}

func (h *Handler) EmptyResponse(statusCode int) {
	h.rw.WriteHeader(statusCode)
}

func (h *Handler) ErrorResponse(err error, msg string) {
	var (
		statusCode int
		log        *zerolog.Event
	)
	errors.Join()

	switch {
	case errors.Is(err, errs.ErrServiceUnavailable):
		statusCode = http.StatusServiceUnavailable
		log = h.log.Error()

	case errs.OneOf(
		err,
		errs.ErrAuthorizationFailed,
		errs.ErrInvalidCredentials,
	):
		statusCode = http.StatusUnauthorized
		log = h.log.Debug()

	case errors.Is(err, errs.ErrSessionIsCompromised):
		statusCode = http.StatusUnauthorized
		log = h.log.Warn()

	case errs.OneOf(
		err,
		errs.ErrInvalidArgument,
		errs.ErrEntityToLarge,
		errs.ErrInvalidIP,
	):
		statusCode = http.StatusBadRequest
		log = h.log.Warn()

	case errors.Is(err, errs.ErrNotFound):
		statusCode = http.StatusNotFound
		log = h.log.Debug()

	case errors.Is(err, errs.ErrConflict):
		statusCode = http.StatusConflict
		log = h.log.Warn()

	default:
		statusCode = http.StatusInternalServerError
		log = h.log.Error()
	}

	log.Err(err).Msg(msg)
	h.errorResponse(statusCode, err, msg)
}

func (h *Handler) PanicResponse(p any, msg string) {
	statusCode := http.StatusInternalServerError
	err := fmt.Errorf("unexpected panic: %v", p)
	h.log.Error().Err(err).Msg(msg)

	h.errorResponse(statusCode, err, msg)
}

func (h *Handler) errorResponse(statusCode int, err error, msg string) {
	response := NewErrorResponse(err, msg)
	h.JsonResponse(response, statusCode)
}
```

### `pkg/httpout/writer.go`

```go
package httpout

import "net/http"

const StatusCodeUninitialized = -1

type ResponseWriter struct {
	w http.ResponseWriter

	statusCode int
}

func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	return &ResponseWriter{
		w:          w,
		statusCode: StatusCodeUninitialized,
	}
}

func (rw *ResponseWriter) Header() http.Header {
	return rw.w.Header()
}

func (rw *ResponseWriter) Write(b []byte) (int, error) {
	if rw.statusCode == StatusCodeUninitialized {
		rw.statusCode = http.StatusOK
	}
	return rw.w.Write(b)
}

func (rw *ResponseWriter) WriteHeader(statusCode int) {
	rw.w.WriteHeader(statusCode)
	rw.statusCode = statusCode
}

func (rw *ResponseWriter) GetStatusCode() int {
	if rw.statusCode == StatusCodeUninitialized {
		return http.StatusOK
	}

	return rw.statusCode
}
```

### `pkg/httpout/error.go`

```go
package httpout

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
} // @name ErrorResponse

func NewErrorResponse(err error, msg string) ErrorResponse {
	return ErrorResponse{
		Error:   err.Error(),
		Message: msg,
	}
}
```

**Legacy, которое не следует повторять:** NewErrorResponse сейчас отправляет
err.Error() клиенту. Для нового проекта полная внутренняя цепочка ошибок остаётся
в логах; клиент получает безопасное сообщение/стабильный код. Таблица статусов
выше показывает текущую политику проекта, не универсальное продуктовое решение.

## Регистрация маршрутов

### `pkg/httpsrv/route.go`

```go
package httpsrv

import (
	"net/http"

	"github.com/AridanWarlock/pinnAutomizer/pkg/httpmv"
)

type Route struct {
	Method             string
	Path               string
	Handler            http.HandlerFunc
	Middlewares        []httpmv.Middleware
	IsPublic           bool
	NeedIdempotencyKey bool
}

func (r Route) WithMiddleware() http.Handler {
	middlewares := r.Middlewares

	if r.NeedIdempotencyKey {
		middlewares = append(middlewares, httpmv.IdKey())
	}
	if !r.IsPublic {
		middlewares = append(middlewares, httpmv.AuthInfo())
	}

	return httpmv.ChainMiddleware(
		r.Handler,
		middlewares...,
	)
}
```

### `pkg/httpsrv/router.go`

```go
package httpsrv

import (
	"fmt"
	"net/http"

	"github.com/AridanWarlock/pinnAutomizer/pkg/httpmv"
)

type ApiVersion int

type ApiVersionRouter struct {
	mux         *http.ServeMux
	apiVersion  ApiVersion
	middlewares []httpmv.Middleware
}

func NewApiVersionRouter(apiVersion ApiVersion, middlewares ...httpmv.Middleware) *ApiVersionRouter {
	return &ApiVersionRouter{
		mux:         http.NewServeMux(),
		apiVersion:  apiVersion,
		middlewares: middlewares,
	}
}

func (r *ApiVersionRouter) RegisterHandlers(handlers ...HttpHandler) {
	for _, handler := range handlers {
		r.mux.Handle(handler.Pattern, handler.Handler)
	}
}

func (r *ApiVersionRouter) RegisterRoutes(routes ...Route) {
	for _, route := range routes {
		r.registerRoute(route)
	}
}

func (r *ApiVersionRouter) registerRoute(route Route) {
	pattern := fmt.Sprintf("%s %s", route.Method, route.Path)

	r.mux.Handle(pattern, route.WithMiddleware())
}

func (r *ApiVersionRouter) WithMiddleware() http.Handler {
	return httpmv.ChainMiddleware(r.mux, r.middlewares...)
}
```

`IsPublic: false` в исходном проекте подключает AuthInfo middleware, которое читает
информацию из headers доверенного gateway. В односервисном приложении нельзя
превращать это в доверие произвольным клиентским заголовкам: аутентификация должна
проверяться в принятой модели безопасности. Показанная регистрация — public route.

HTTP-сервер (`pkg/httpsrv/server.go`), logging middleware, lifecycle и logger не
включены полностью. Их контракты в примере нужны для wiring, не как скрытая
обязанность другого агента найти исходный репозиторий. Реализуй их в новом модуле
по AGENTS.md: timeout сервера, graceful shutdown, обработка panic и логирование.

## Проверки транспорта

Используй httptest и сгенерированный MockUsecase: корректный JSON → точный Input,
успешный Output → 201/Response, невалидный JSON → отказ без вызова usecase,
ошибка сценария → согласованный HTTP-статус. Для SQL в этих тестах моки не нужны: SQL
вообще не должен вызываться handler-ом. Тесты ошибок writer/decoder — у shared helper.
