package authRefresh

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pinn/internal/domain/fixtures"
	"github.com/AridanWarlock/pinnAutomizer/pkg/crypt"
	"github.com/AridanWarlock/pinnAutomizer/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHttpHandler_Refresh(t *testing.T) {
	type fields struct {
		usecase *MockUsecase
	}

	var (
		fixedOldRefreshTokenStr = crypt.GenerateSecureToken()
		fixedNewRefreshTokenStr = crypt.GenerateSecureToken()
		fixedExpiry             = time.Now().Truncate(time.Second).UTC()
		fixedAccess             = fixtures.NewAccessToken()
	)

	tests := []struct {
		name           string
		cookie         *http.Cookie
		prepare        func(f *fields)
		expectedStatus int
		checkResponse  func(t *testing.T, resp *httptest.ResponseRecorder)
	}{
		{
			name:   "success path",
			cookie: &http.Cookie{Name: "refresh_token", Value: fixedOldRefreshTokenStr},
			prepare: func(f *fields) {
				f.usecase.RefreshFunc = func(ctx context.Context, in Input) (Output, error) {
					assert.Equal(t, fixedOldRefreshTokenStr, in.RefreshTokenString)
					return Output{
						AccessToken:           fixedAccess,
						RefreshTokenString:    fixedNewRefreshTokenStr,
						RefreshTokenExpiresAt: fixedExpiry,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *httptest.ResponseRecorder) {
				var body Response
				err := json.Unmarshal(resp.Body.Bytes(), &body)
				require.NoError(t, err)
				assert.Equal(t, string(fixedAccess), body.AccessToken)

				cookies := resp.Result().Cookies()
				require.Len(t, cookies, 1)
				assert.Equal(t, "refresh_token", cookies[0].Name)
				assert.Equal(t, fixedNewRefreshTokenStr, cookies[0].Value)
				assert.True(t, cookies[0].HttpOnly)
				assert.Equal(t, http.SameSiteLaxMode, cookies[0].SameSite)
				assert.Equal(t, fixedExpiry, cookies[0].Expires)
			},
		},
		{
			name:           "error - no cookie",
			cookie:         nil,
			prepare:        func(f *fields) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:   "error - usecase fail",
			cookie: &http.Cookie{Name: "refresh_token", Value: fixedNewRefreshTokenStr},
			prepare: func(f *fields) {
				f.usecase.RefreshFunc = func(ctx context.Context, in Input) (Output, error) {
					return Output{}, sql.ErrConnDone
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

			req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
			req = req.WithContext(test.ContextWithZeroLogger())

			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}

			w := httptest.NewRecorder()

			handler.Refresh(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}
