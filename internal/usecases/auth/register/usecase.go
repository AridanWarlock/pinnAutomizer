package register

import (
	"context"
	"pinnAutomizer/internal/domain"
	"pinnAutomizer/pkg/tx"

	"github.com/rs/zerolog"
)

type Postgres interface {
	GetRoleByTitle(ctx context.Context, title string) (domain.Role, error)
	CreateUser(ctx context.Context, user domain.User) (domain.User, error)
	CreateUsersRolesBatch(ctx context.Context, usersRoles []domain.UsersRoles) ([]domain.UsersRoles, error)

	tx.Wrapper
}

type PasswordHasher interface {
	HashPassword(password string) (string, error)
}

type Usecase struct {
	postgres       Postgres
	passwordHasher PasswordHasher

	log zerolog.Logger
}

var usecase *Usecase

func New(
	postgres Postgres,
	passwordHasher PasswordHasher,
	log zerolog.Logger,
) *Usecase {
	u := &Usecase{
		postgres:       postgres,
		passwordHasher: passwordHasher,

		log: log.With().Str("component", "usecase: auth.Register").Logger(),
	}

	usecase = u

	return u
}

func (u *Usecase) Register(ctx context.Context, in Input) error {
	log := u.log.With().Ctx(ctx).Logger()

	if err := in.Validate(); err != nil {
		log.Info().Err(err).Msg("input validation error")
		return err
	}

	passwordHash, err := u.passwordHasher.HashPassword(in.Password)
	if err != nil {
		log.Error().Err(err).Msg("hash password error")
		return err
	}

	user, err := domain.NewUser(in.Login, passwordHash)
	if err != nil {
		log.Error().Err(err).Msg("user domain model creating error")
		return err
	}

	role, err := u.postgres.GetRoleByTitle(ctx, string(domain.RoleTypeUser))
	if err != nil {
		log.Error().Err(err).Msg("usecase: postgres.GetRoleByTitle")
		return err
	}

	err = u.postgres.Wrap(ctx, func(ctx context.Context) error {
		user, err = u.createUser(ctx, user, []domain.Role{role})
		return err
	})

	if err != nil {
		log.Error().
			Err(err).
			Msg("saving user in postgres error")
		return err
	}

	return nil
}

func (u *Usecase) createUser(ctx context.Context, user domain.User, roles []domain.Role) (domain.User, error) {
	user, err := u.postgres.CreateUser(ctx, user)
	if err != nil {
		u.log.Error().Err(err).Msg("usecase: postgres.CreateUser")
		return domain.User{}, err
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
		u.log.Error().Err(err).Msg("usecase: postgres.CreateUsersRolesBatch")
		return domain.User{}, err
	}

	return user, nil
}
