# PostgreSQL: клиент, пул, транзакции и адаптер

Снимок исходников: `4b0892e9eebf16c3f915c047692f466eee3fdd78`. Имена предметной области сохранены для
прослеживаемости. Нормативные правила нового проекта — в соседнем AGENTS.md;
отмеченные ниже legacy-места не являются рекомендуемым образцом.

## Цепочка зависимостей

```text
cmd/main → postgres.New → poolx.New → pgxpool.NewWithConfig → Ping
usecase.Postgres ← реализует postgres.Repository
postgres.Repository → poolx.Pool → executor(ctx): собственная tx или pgxpool
```

`Postgres` у usecase — узкий контракт бизнес-операций хранения. `poolx.Pool` —
технический контракт SQL-клиента. Это разные интерфейсы разных уровней.
Конкретный `Repository` создаётся только в composition root.

## Общий SQL-клиент

- `pgxpool` владеет соединениями; `poolx` добавляет Squirrel/Sqlizer, сканирование,
  timeout, перевод ошибок и выбор исполнителя транзакции.
- New действительно проверяет подключение через Ping. Ошибка проверки закрывает
  уже созданный пул; успешный возврат передаёт обязанность Close вызывающему коду.
- Начальная проверка ограничена 5s; операции и вся транзакция — `Config.Timeout`.
- MaxConns/MinConns здесь явно не настроены: действуют значения pgxpool. Не выводи
  из этого кода выдуманных гарантий размера пула.
- Ключ transaction context содержит владельца `*Poolx`: другой клиент не заимствует
  чужую транзакцию. Вложенная транзакция того же клиента отклоняется.
- В callback нужно передавать именно полученный `ctx`. Использование внешнего ctx
  может выполнить запрос вне транзакции. После callback выполняется commit;
  defer пытается выполнить rollback и сохраняет значимую ошибку очистки.
- Ошибка commit допускает неизвестный исход. Эти методы не дают exactly-once
  и не включают Redis/HTTP/файлы в PostgreSQL-транзакцию.

### `pkg/postgres/poolx/config.go`

```go
package poolx

import "time"

type Config struct {
	User     string
	Password string
	Host     string
	Port     int
	DB       string
	SslMode  string
	Timeout  time.Duration
}
```

### `pkg/postgres/poolx/pool.go`

```go
package poolx

import (
	"context"
)

type Sqlizer interface {
	ToSql() (sql string, args []any, err error)
}

type Tag interface {
	RowsAffected() int
	ExactlyOneRowAffected() bool
}

type Pool interface {
	Getx(ctx context.Context, dst any, sqlizer Sqlizer) error
	Selectx(ctx context.Context, dst any, sqlizer Sqlizer) error
	Execx(ctx context.Context, sqlizer Sqlizer) (Tag, error)
	InTransaction(ctx context.Context, inTx func(ctx context.Context) error) error

	Close()
}
```

### `pkg/postgres/poolx/tagx.go`

```go
package poolx

import "github.com/jackc/pgx/v5/pgconn"

type Tagx struct {
	rowsAffected int
}

func newTagxFromPgx(tag pgconn.CommandTag) Tagx {
	return Tagx{
		rowsAffected: int(tag.RowsAffected()),
	}
}

func (t Tagx) RowsAffected() int {
	return t.rowsAffected
}

func (t Tagx) ExactlyOneRowAffected() bool {
	return t.rowsAffected == 1
}
```

### `pkg/postgres/poolx/poolx.go`

```go
package poolx

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
)

type TxManager interface {
	InTransaction(
		ctx context.Context,
		inTx func(context.Context) error,
	) error
}

type txKey struct {
	postgres *Poolx
}

func toSqlErr(err error) error {
	return fmt.Errorf("postgres: to sql: %w", err)
}

type executor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (commandTag pgconn.CommandTag, err error)
	Query(ctx context.Context, query string, args ...any) (pgx.Rows, error)
}

type Postgres interface {
	executor
	Begin(ctx context.Context) (pgx.Tx, error)
	Ping(ctx context.Context) error
	Close()
}

type Poolx struct {
	postgres Postgres

	timeout time.Duration
}

func New(cfg Config) (Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	connString := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.DB,
		cfg.SslMode,
	)
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("parse pgxconfig: %w", err)
	}

	pgxPool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("pgxpool connect: %w", err)
	}

	return newPoolx(ctx, pgxPool, cfg.Timeout)
}

// newPoolx takes ownership: failed verification closes the pool, success transfers
// it to the returned client, whose owner must call Close.
func newPoolx(
	ctx context.Context,
	postgres Postgres,
	timeout time.Duration,
) (client Pool, err error) {
	defer func() {
		if err != nil {
			postgres.Close()
		}
	}()
	if err := postgres.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return &Poolx{
		postgres: postgres,
		timeout:  timeout,
	}, nil
}

func (p *Poolx) Getx(ctx context.Context, dst any, sqlizer Sqlizer) error {
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	query, args, err := sqlizer.ToSql()
	if err != nil {
		return toSqlErr(err)
	}

	err = pgxscan.Get(ctx, p.executor(ctx), dst, query, args...)
	if err != nil {
		return p.handleErr(err)
	}
	return nil
}

func (p *Poolx) Selectx(ctx context.Context, dst any, sqlizer Sqlizer) error {
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	query, args, err := sqlizer.ToSql()
	if err != nil {
		return toSqlErr(err)
	}

	err = pgxscan.Select(ctx, p.executor(ctx), dst, query, args...)
	if err != nil {
		return p.handleErr(err)
	}
	return nil
}

func (p *Poolx) Execx(ctx context.Context, sqlizer Sqlizer) (Tag, error) {
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	query, args, err := sqlizer.ToSql()
	if err != nil {
		return Tagx{}, toSqlErr(err)
	}

	tag, err := p.executor(ctx).Exec(ctx, query, args...)
	if err != nil {
		return Tagx{}, p.handleErr(err)
	}
	return newTagxFromPgx(tag), nil
}

func (p *Poolx) InTransaction(
	ctx context.Context,
	inTx func(context.Context) error,
) (err error) {
	if inTx == nil {
		return fmt.Errorf("%w: nil transaction callback", errs.ErrInvalidArgument)
	}
	if _, ok := ctx.Value(txKey{postgres: p}).(pgx.Tx); ok {
		return fmt.Errorf("%w: nested PostgreSQL transaction", errs.ErrInvalidArgument)
	}
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	tx, err := p.postgres.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		rollbackErr := tx.Rollback(ctx)
		if rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			err = errors.Join(err, fmt.Errorf("tx.Rollback: %w", rollbackErr))
		}
	}()

	ctx = context.WithValue(ctx, txKey{postgres: p}, tx)

	err = inTx(ctx)
	if err != nil {
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("tx.Commit: %w", err)
	}
	return nil
}

func (p *Poolx) executor(ctx context.Context) executor {
	tx, ok := ctx.Value(txKey{postgres: p}).(pgx.Tx)
	if ok {
		return tx
	}
	return p.postgres
}

func (p *Poolx) handleErr(err error) error {
	var pgErr *pgconn.PgError

	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return errs.ErrNotFound
	case errors.As(err, &pgErr):
		switch pgErr.Code {
		case "23505", "23503":
			return errs.ErrConflict
		}
	case errs.IsContextErr(err):
		return err
	}

	return fmt.Errorf("postgres err: %w", err)
}

func (p *Poolx) Close() {
	p.postgres.Close()
}
```

## Адаптер приложения

В новом односервисном проекте эти файлы размещаются в `internal/adapter/postgres`.
Общий клиент остаётся в `pkg/postgres/poolx` того же корневого Go-модуля.

### `services/auth/internal/adapter/postgres/postgres.go`

```go
package postgres

import (
	"context"
	"fmt"
	"time"

	sq "github.com/Masterminds/squirrel"

	"github.com/AridanWarlock/pinnAutomizer/pkg/postgres/poolx"
)

type Config struct {
	User     string        `env:"USER,required"`
	Password string        `env:"PASSWORD,required"`
	Host     string        `env:"HOST,required"`
	Port     int           `env:"PORT,required"`
	DB       string        `env:"DB,required"`
	SslMode  string        `env:"SSLMODE" default:"disable"`
	Timeout  time.Duration `env:"TIMEOUT,required"`
}

type Repository struct {
	pool poolx.Pool
	sb   sq.StatementBuilderType
}

func New(cfg Config) (*Repository, error) {
	p, err := poolx.New(poolx.Config{
		User:     cfg.User,
		Password: cfg.Password,
		Host:     cfg.Host,
		Port:     cfg.Port,
		DB:       cfg.DB,
		SslMode:  cfg.SslMode,
		Timeout:  cfg.Timeout,
	})

	if err != nil {
		return nil, fmt.Errorf("postgres: poolx create: %w", err)
	}

	return &Repository{
		pool: p,
		sb:   sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}, nil
}

func (r *Repository) InTransaction(ctx context.Context, inTx func(ctx context.Context) error) error {
	return r.pool.InTransaction(ctx, inTx)
}

func (r *Repository) Close() {
	r.pool.Close()
}
```

### `services/auth/internal/adapter/postgres/schema.go` — фрагмент

```go
const RolesTable = "roles"
const (
	RolesID    = "id"
	RolesTitle = "title"
)

var RolesColumns = []string{
	RolesID,
	RolesTitle,
}

const UsersTable = "users"
const (
	UsersID           = "id"
	UsersLogin        = "login"
	UsersPasswordHash = "password_hash"
)

var UsersColumns = []string{
	UsersID,
	UsersLogin,
	UsersPasswordHash,
}

const UsersRolesTable = "users_roles"
const (
	UsersRolesUserID = "user_id"
	UsersRolesRoleID = "role_id"
)

var UsersRolesColumns = []string{
	UsersRolesUserID,
	UsersRolesRoleID,
}
```

### `services/auth/internal/adapter/postgres/users_converters.go`

```go
package postgres

import (
	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/auth/internal/domain"
)

type UserRow struct {
	ID           uuid.UUID `db:"id"`
	Login        string    `db:"login"`
	PasswordHash string    `db:"password_hash"`
}

func (r UserRow) Values() []any {
	return []any{
		r.ID,
		r.Login,
		r.PasswordHash,
	}
}

func toUserModel(r UserRow) domain.User {
	return domain.User{
		ID:           r.ID,
		Login:        r.Login,
		PasswordHash: r.PasswordHash,
	}
}

func fromUserModel(u domain.User) UserRow {
	return UserRow{
		ID:           u.ID,
		Login:        u.Login,
		PasswordHash: u.PasswordHash,
	}
}
```

### `services/auth/internal/adapter/postgres/users.go`

```go
package postgres

import (
	"context"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/auth/internal/domain"
)

func (r *Repository) CreateUser(ctx context.Context, user domain.User) (domain.User, error) {
	row := fromUserModel(user)
	query := r.sb.
		Insert(UsersTable).
		Columns(UsersColumns...).
		Values(row.Values()...).
		Suffix("RETURNING " + strings.Join(UsersColumns, ","))

	var outRow UserRow
	if err := r.pool.Getx(ctx, &outRow, query); err != nil {
		return domain.User{}, err
	}
	return toUserModel(outRow), nil
}

func (r *Repository) GetUserByID(ctx context.Context, id uuid.UUID) (domain.User, error) {
	query := r.sb.
		Select(UsersColumns...).
		From(UsersTable).
		Where(sq.Eq{UsersID: id})

	var row UserRow
	if err := r.pool.Getx(ctx, &row, query); err != nil {
		return domain.User{}, err
	}
	return toUserModel(row), nil
}

func (r *Repository) GetUserByLogin(ctx context.Context, login string) (domain.User, error) {
	query := r.sb.
		Select(UsersColumns...).
		From(UsersTable).
		Where(sq.Eq{UsersLogin: login})

	var outRow UserRow
	if err := r.pool.Getx(ctx, &outRow, query); err != nil {
		return domain.User{}, err
	}

	return toUserModel(outRow), nil
}
```

### `services/auth/internal/adapter/postgres/roles_converters.go`

```go
package postgres

import (
	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
)

type RoleRaw struct {
	ID    uuid.UUID `db:"id"`
	Title string    `db:"title"`
}

func FromRoleModel(r core.Role) RoleRaw {
	return RoleRaw{
		ID:    r.ID,
		Title: r.Title,
	}
}

func ToRoleModel(r RoleRaw) core.Role {
	return core.Role{
		ID:    r.ID,
		Title: r.Title,
	}
}
```

### `services/auth/internal/adapter/postgres/roles.go`

```go
package postgres

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
)

func (r *Repository) GetRoleByTitle(ctx context.Context, title string) (core.Role, error) {
	query := r.sb.
		Select(RolesColumns...).
		From(RolesTable).
		Where(sq.Eq{RolesTitle: title})

	var outRow RoleRaw
	if err := r.pool.Getx(ctx, &outRow, query); err != nil {
		return core.Role{}, err
	}
	return ToRoleModel(outRow), nil
}

func (r *Repository) GetRolesByUserID(ctx context.Context, userID uuid.UUID) ([]core.Role, error) {
	q := r.sb.Select(fmt.Sprintf("r.%s, r.%s", RolesID, RolesTitle)).
		From(UsersRolesTable + " ur").
		InnerJoin(fmt.Sprintf("%s r on r.%s = ur.%s", RolesTable, RolesID, UsersRolesRoleID)).
		Where(sq.Eq{"ur." + UsersRolesUserID: userID})

	var rows []RoleRaw
	if err := r.pool.Selectx(ctx, &rows, q); err != nil {
		return nil, err
	}

	roles := make([]core.Role, len(rows))
	for i, row := range rows {
		roles[i] = ToRoleModel(row)
	}
	return roles, nil
}
```

### `services/auth/internal/adapter/postgres/users_roles_converters.go`

```go
package postgres

import (
	"github.com/google/uuid"

	"github.com/AridanWarlock/pinnAutomizer/auth/internal/domain"
)

type UsersRolesRow struct {
	UserID uuid.UUID `db:"user_id"`
	RoleID uuid.UUID `db:"role_id"`
}

func (r UsersRolesRow) Values() []any {
	return []any{
		r.UserID,
		r.RoleID,
	}
}

func fromUsersRolesModel(ur domain.UsersRoles) UsersRolesRow {
	return UsersRolesRow{
		UserID: ur.UserID,
		RoleID: ur.RoleID,
	}
}

func toUsersRolesModel(r UsersRolesRow) domain.UsersRoles {
	return domain.UsersRoles{
		UserID: r.UserID,
		RoleID: r.RoleID,
	}
}
```

### `services/auth/internal/adapter/postgres/users_roles.go`

```go
package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/AridanWarlock/pinnAutomizer/auth/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
)

func (r *Repository) CreateUsersRolesBatch(ctx context.Context, usersRoles []domain.UsersRoles) ([]domain.UsersRoles, error) {
	batchSize := len(usersRoles)
	if batchSize == 0 || batchSize > 100 {
		return nil, fmt.Errorf(
			"%w: invalid batch size=%d",
			errs.ErrInvalidArgument,
			batchSize,
		)
	}

	query := r.sb.
		Insert(UsersRolesTable).
		Columns(UsersRolesColumns...)

	for _, ur := range usersRoles {
		values := fromUsersRolesModel(ur).Values()
		query = query.Values(values...)
	}

	query = query.Suffix("RETURNING " + strings.Join(UsersRolesColumns, ","))

	var outRows []UsersRolesRow
	if err := r.pool.Selectx(ctx, &outRows, query); err != nil {
		return nil, err
	}

	res := make([]domain.UsersRoles, batchSize)
	for i, row := range outRows {
		res[i] = toUsersRolesModel(row)
	}
	return res, nil
}
```

## Что переносить с поправкой на текущие правила

- Row, Values, список колонок, query builder и узкий DI — образцы устройства слоя.
- `toUserModel`, `toRoleModel`, `toUsersRolesModel` восстанавливают legacy-сущности
  литералами. Для нового проекта нужен проверяющий Restore-конструктор; нарушение
  persistence-контракта — fail-fast, а не возврат некорректной сущности.
- Рабочие domain-объекты нельзя создавать литералами и при подготовке связи сущностей.
- New собирает DSN через fmt.Sprintf. При переносе нужно корректно кодировать
  специальные символы credentials, например при помощи net/url; DSN не логируется.
- Текстовые SQLSTATE 23505/23503 сейчас сводятся к ErrConflict. Это фактическая
  политика клиента, а не универсальная классификация ошибок для любого продукта.
- Схему создаёт внешняя Up-миграция Goose через Make/Compose. Конструктор адаптера
  не применяет миграции и не добавляет собственную проверку их наличия.

## Проверки этого слоя

Источники тестов: `pkg/postgres/poolx/poolx_test.go`,
`pkg/postgres/poolx/poolx_integration_test.go`. Они намеренно не продублированы
полностью: сначала агент должен понять контракт, затем написать проверки своего
продукта. Сохрани проверки Ping failure/Close, rollback/commit, owner транзакции,
запрета вложенности, SQL round-trip и конкурентного изменения. Моки не заменяют
настоящий PostgreSQL. При отсутствии integration-конфигурации пропуск не доказывает
SQL-гарантии.
