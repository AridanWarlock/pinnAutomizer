package authLogout

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AridanWarlock/pinnAutomizer/pkg/test"
	"github.com/stretchr/testify/assert"
)

func TestHttpHandler_Logout(t *testing.T) {
	type fields struct {
		usecase *MockUsecase
	}

	tests := []struct {
		name           string
		prepare        func(f *fields)
		expectedStatus int
	}{
		{
			name: "success path",
			prepare: func(f *fields) {
				f.usecase.LogoutFunc = func(ctx context.Context) error {
					return nil
				}
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "error - usecase fail",
			prepare: func(f *fields) {
				f.usecase.LogoutFunc = func(ctx context.Context) error {
					return errors.New("logout failed")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fields{usecase: &MockUsecase{}}
			tt.prepare(f)

			handler := NewHttpHandler(f.usecase)
			ctx := test.ContextWithZeroLogger()

			req := httptest.NewRequest(http.MethodPost, "/auth/logout", nil).
				WithContext(ctx)
			w := httptest.NewRecorder()

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
		if c.Name == "refresh_token" {
			found = true
			assert.True(t, c.MaxAge < 0, "MaxAge должен быть < 0 для удаления")
			assert.Equal(t, "/api/v1/auth/refresh", c.Path)
			assert.True(t, c.HttpOnly)
			break
		}
	}
	assert.True(t, found, "Кука refreshToken не была найдена в ответе")
}
