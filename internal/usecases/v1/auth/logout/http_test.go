package authLogout

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/internal/domain/fixtures"
	"github.com/AridanWarlock/pinnAutomizer/pkg/test"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestHttpHandler_Logout(t *testing.T) {
	type fields struct {
		usecase *MockUsecase
	}

	var (
		fixedUserID      = uuid.New()
		fixedFingerprint = fixtures.NewFingerprint()
		testCtx          = test.ContextBackgroundWithZeroLogger()
	)

	tests := []struct {
		name           string
		prepareContext func(ctx context.Context) context.Context
		prepare        func(f *fields)
		expectedStatus int
		shouldPanic    bool
	}{
		{
			name: "success path",
			prepareContext: func(ctx context.Context) context.Context {
				return test.ContextWithUserClaims(ctx, fixtures.NewUserClaims(func(uc *domain.UserClaims) {
					uc.UserID = fixedUserID
					uc.Fingerprint = fixedFingerprint
				}))
			},
			prepare: func(f *fields) {
				f.usecase.LogoutFunc = func(ctx context.Context, in Input) error {
					assert.Equal(t, fixedUserID, in.UserID)
					assert.Equal(t, fixedFingerprint, in.Fingerprint)
					return nil
				}
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "error - usecase fail",
			prepareContext: func(ctx context.Context) context.Context {
				return test.ContextWithUserClaims(ctx, fixtures.NewUserClaims(func(uc *domain.UserClaims) {
					uc.UserID = fixedUserID
					uc.Fingerprint = fixedFingerprint
				}))
			},
			prepare: func(f *fields) {
				f.usecase.LogoutFunc = func(ctx context.Context, in Input) error {
					return errors.New("logout failed")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "critical - panic when claims are missing",
			prepareContext: func(ctx context.Context) context.Context {
				return ctx
			},
			prepare:     func(f *fields) {},
			shouldPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fields{usecase: &MockUsecase{}}
			tt.prepare(f)
			handler := NewHttpHandler(f.usecase)

			req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
			req = req.WithContext(tt.prepareContext(testCtx))

			w := httptest.NewRecorder()

			if tt.shouldPanic {
				assert.Panics(t, func() {
					handler.Logout(w, req)
				}, "Ожидалась паника из-за отсутствия Claims в контексте")
				return
			}

			handler.Logout(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusNoContent {
				checkCookieCleared(t, w)
			}
		})
	}
}

func checkCookieCleared(t *testing.T, w *httptest.ResponseRecorder) {
	cookies := w.Result().Cookies()
	var found bool
	for _, c := range cookies {
		if c.Name == "refreshToken" {
			found = true
			assert.True(t, c.MaxAge < 0, "MaxAge должен быть < 0 для удаления")
			assert.Equal(t, "/api/v1/auth/refresh", c.Path)
			assert.True(t, c.HttpOnly)
			break
		}
	}
	assert.True(t, found, "Кука refreshToken не была найдена в ответе")
}
