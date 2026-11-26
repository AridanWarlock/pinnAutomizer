# Usecase и явный DI

Снимок исходников: `4b0892e9eebf16c3f915c047692f466eee3fdd78`. Сквозной пример — регистрация,
`services/auth/internal/usecases/v1/auth/register`. Это пример организации кода,
а не требование создавать регистрацию/роли в другой предметной области.

## Публичный контракт и реализация

- interface.go содержит Input, Output, Usecase. HTTP Request и Response туда не входят.
- В usecase.go перед структурой объявлены узкие Postgres и PasswordHasher.
  Конкретные SDK и SQL-запросы сюда не импортируются.
- Нью-конструктор принимает интерфейсы, возвращает публичный Usecase и сохраняет
  зависимости в приватной реализации. Сервисы берутся из аргументов, не из глобального реестра.
- HashPassword выполняется до начала SQL-транзакции. В callback остаются операции БД.
- При наличии бизнес-параметров используется Input, даже для одного поля. Без
  бизнес-параметров допустим только ctx. Бизнес-результаты возвращаются через Output.

### `services/auth/internal/usecases/v1/auth/register/interface.go`

```go
package authRegister

import (
	"context"

	"github.com/AridanWarlock/pinnAutomizer/auth/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"
)

type Input struct {
	Login             string `validate:"required"`
	Password          string `validate:"required,min=5"`
	PasswordConfirmed string `validate:"required,eqfield=Password"`
}

func (i Input) Validate() error {
	return validate.V.Struct(i)
}

type Output struct {
	User domain.User `json:"user"`
}

type Usecase interface {
	Register(ctx context.Context, in Input) (Output, error)
}
```

### `services/auth/internal/usecases/v1/auth/register/usecase.go`

```go
package authRegister

import (
	"context"
	"fmt"

	"github.com/AridanWarlock/pinnAutomizer/auth/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/postgres/poolx"
)

type Postgres interface {
	GetRoleByTitle(ctx context.Context, title string) (core.Role, error)
	CreateUser(ctx context.Context, user domain.User) (domain.User, error)
	CreateUsersRolesBatch(ctx context.Context, usersRoles []domain.UsersRoles) ([]domain.UsersRoles, error)

	poolx.TxManager
}

type PasswordHasher interface {
	HashPassword(password string) (string, error)
}

type usecase struct {
	postgres       Postgres
	passwordHasher PasswordHasher
}

func New(
	postgres Postgres,
	passwordHasher PasswordHasher,
) Usecase {
	return &usecase{
		postgres:       postgres,
		passwordHasher: passwordHasher,
	}
}

func (u *usecase) Register(ctx context.Context, in Input) (Output, error) {
	if err := in.Validate(); err != nil {
		return Output{}, fmt.Errorf("%w: %v", errs.ErrInvalidArgument, err)
	}

	passwordHash, err := u.passwordHasher.HashPassword(in.Password)
	if err != nil {
		return Output{}, fmt.Errorf("hash password: %w", err)
	}

	user, err := domain.NewUser(in.Login, passwordHash)
	if err != nil {
		return Output{}, fmt.Errorf("user model create: %w", err)
	}

	role, err := u.postgres.GetRoleByTitle(ctx, string(core.RoleTypeUser))
	if err != nil {
		return Output{}, fmt.Errorf("get role by title from postgres: %w", err)
	}

	err = u.postgres.InTransaction(ctx, func(ctx context.Context) error {
		user, err = u.createUser(ctx, user, []core.Role{role})
		return err
	})

	if err != nil {
		return Output{}, fmt.Errorf("create user transaction: %w", err)
	}

	return Output{
		User: user,
	}, nil
}

func (u *usecase) createUser(ctx context.Context, user domain.User, roles []core.Role) (domain.User, error) {
	user, err := u.postgres.CreateUser(ctx, user)
	if err != nil {
		return domain.User{}, fmt.Errorf("create user in postgres: %w", err)
	}

	usersRolesBatch := make([]domain.UsersRoles, len(roles))
	for i, role := range roles {
		usersRolesBatch[i] = domain.UsersRoles{
			UserID: user.ID,
			RoleID: role.ID,
		}
	}

	_, err = u.postgres.CreateUsersRolesBatch(ctx, usersRolesBatch)
	if err != nil {
		return domain.User{}, fmt.Errorf("create user roles in postgres: %w", err)
	}

	return user, nil
}
```

## Конструктор предметной сущности

### `services/auth/internal/domain/user.go`

```go
package domain

import (
	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/pkg/validate"
)

type User struct {
	ID           uuid.UUID `json:"id" validate:"required,uuid"`
	Login        string    `json:"login" validate:"required"`
	PasswordHash string    `json:"password_hash" validate:"required"`
}

func NewUser(login string, passwordHash string) (User, error) {
	u := User{
		ID:           uuid.New(),
		Login:        login,
		PasswordHash: passwordHash,
	}

	if err := u.Validate(); err != nil {
		return User{}, err
	}

	return u, nil
}

func (s *User) Validate() error {
	return validate.V.Struct(s)
}
```

**Legacy, которое не следует повторять:** User имеет экспортируемые поля;
createUser создаёт UsersRoles литералом. В новом проекте сущности имеют проверяющие
конструкторы, методы переходов и приватное состояние. Сами Input/Output — DTO.
JSON-тег в Output не делает его HTTP Response: наружу handler строит отдельный DTO.

## Composition root: как связываются реальные зависимости

Ниже адаптированный фрагмент `services/auth/cmd/main.go`: оставлены только
создание адаптера, hasher, usecase и handler. Это часть AppRun, не полный main.go;
`ctx`, `cfg` и `log` создаёт вызывающий composition root.

```go
postgresAdapter, err := postgres.New(cfg.Postgres)
if err != nil {
	return fmt.Errorf("postgres connect: %w", err)
}
defer postgresAdapter.Close()

hasher := password.NewHasher()
registerUsecase := authRegister.New(postgresAdapter, hasher)
registerHandler := authRegister.NewHttpHandler(registerUsecase)

apiRouter := httpsrv.NewApiVersionRouter(httpsrv.ApiVersion(1))
apiRouter.RegisterRoutes(registerHandler.Route())

server := httpsrv.NewWithDefaultMiddlewares(cfg.HTTP, log)
server.RegisterApiRouters(apiRouter)

return lifecycle.Run(ctx,
	lifecycle.Worker("http server", server.Run),
)
```

Интерфейс реализуется структурно: Repository удовлетворяет Postgres usecase-а,
поскольку предоставляет его методы; декларация implements и регистрация в DI
не нужны. Аналогично PasswordHasher удовлетворяется конкретным hasher.
Вся цепочка освобождает пул после завершения lifecycle, а не после создания handler.

## Тесты usecase

Мокируй только Postgres/PasswordHasher. Проверь отказ Input, отказ hashing, ошибку
хранения, успешный Output и границы транзакции. Интеграцией отдельно проверь
отсутствие частично записанных связей при rollback. Фикстуры сущностей — через
конструкторы. Моки генерируются Mockery и коммитятся с интерфейсом.
