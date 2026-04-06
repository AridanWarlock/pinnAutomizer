package auth_register

import (
	"context"
	"fmt"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/pkg/tx"
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
}

func New(
	postgres Postgres,
	passwordHasher PasswordHasher,
) *Usecase {
	return &Usecase{
		postgres:       postgres,
		passwordHasher: passwordHasher,
	}
}

func (u *Usecase) Register(ctx context.Context, in Input) (Output, error) {
	if err := in.Validate(); err != nil {
		return Output{}, domain.ErrInputValidation
	}

	passwordHash, err := u.passwordHasher.HashPassword(in.Password)
	if err != nil {
		return Output{}, fmt.Errorf("hash password: %w", err)
	}

	user, err := domain.NewUser(in.Login, passwordHash)
	if err != nil {
		return Output{}, fmt.Errorf("user model create: %w", err)
	}

	role, err := u.postgres.GetRoleByTitle(ctx, string(domain.RoleTypeUser))
	if err != nil {
		return Output{}, fmt.Errorf("get role by title from postgres: %w", err)
	}

	err = u.postgres.Wrap(ctx, func(ctx context.Context) error {
		user, err = u.createUser(ctx, user, []domain.Role{role})
		return err
	})

	if err != nil {
		return Output{}, fmt.Errorf("create user transaction: %w", err)
	}

	return Output{
		User: user,
	}, nil
}

func (u *Usecase) createUser(ctx context.Context, user domain.User, roles []domain.Role) (domain.User, error) {
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
