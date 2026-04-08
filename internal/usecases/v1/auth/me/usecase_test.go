package authMe

import (
	"context"
	"testing"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/internal/domain/fixtures"
	"github.com/AridanWarlock/pinnAutomizer/internal/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/test"
	"github.com/stretchr/testify/assert"

	"github.com/google/uuid"
)

func TestUsecase_Me(t *testing.T) {
	type fields struct {
		postgres *MockPostgres
	}

	var (
		fixedID = uuid.New()
		testCtx = test.ContextBackgroundWithZeroLogger()
	)

	tests := []struct {
		name    string
		input   Input
		prepare func(f *fields)
		check   func(t *testing.T, out Output, err error, f *fields)
	}{
		{
			name: "successful path",
			input: Input{
				UserID: fixedID,
			},
			prepare: func(f *fields) {
				f.postgres.GetUserByIDFunc = func(ctx context.Context, id uuid.UUID) (domain.User, error) {
					return fixtures.NewUser(func(user *domain.User) {
						user.ID = fixedID
						user.Login = "Ivan Ivanov"
					}), nil
				}
			},
			check: func(t *testing.T, out Output, err error, f *fields) {
				assert.NoError(t, err)
				assert.Equal(t, "Ivan Ivanov", out.Login)
			},
		},
		{
			name: "error - invalid id",
			input: Input{
				UserID: uuid.Nil,
			},
			prepare: func(f *fields) {
			},
			check: func(t *testing.T, out Output, err error, f *fields) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), errs.ErrInvalidArgument.Error())
				assert.Len(t, f.postgres.GetUserByIDCalls(), 0)
			},
		},
		{
			name: "error - user not found",
			input: Input{
				UserID: fixedID,
			},
			prepare: func(f *fields) {
				f.postgres.GetUserByIDFunc = func(ctx context.Context, id uuid.UUID) (domain.User, error) {
					return domain.User{}, errs.ErrNotFound
				}
			},
			check: func(t *testing.T, out Output, err error, f *fields) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), errs.ErrNotFound.Error())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fields{
				postgres: &MockPostgres{},
			}

			tt.prepare(f)

			uc := New(f.postgres)
			actual, err := uc.Me(testCtx, tt.input)

			tt.check(t, actual, err, f)
		})
	}
}
