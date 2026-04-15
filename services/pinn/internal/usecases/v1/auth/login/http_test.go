package authLogin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pinn/internal/domain/fixtures"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/test"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHttpHandler_Login(t *testing.T) {
	type fields struct {
		usecase *MockUsecase
	}

	var (
		fixedAccessToken  = fixtures.NewAccessToken()
		fixedRefreshToken = uuid.NewString()
		fixedExpiry       = time.Now().Truncate(time.Second).UTC()
	)

	tests := []struct {
		name           string
		requestBody    interface{}
		prepare        func(f *fields)
		expectedStatus int
		checkResponse  func(t *testing.T, resp *httptest.ResponseRecorder)
	}{
		{
			name: "success path",
			requestBody: Request{
				Login:    "admin",
				Password: "password123",
			},
			prepare: func(f *fields) {
				f.usecase.LoginFunc = func(ctx context.Context, in Input) (Output, error) {
					assert.Equal(t, "admin", in.Login)

					return Output{
						AccessToken:           fixedAccessToken,
						RefreshTokenString:    fixedRefreshToken,
						RefreshTokenExpiresAt: fixedExpiry,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *httptest.ResponseRecorder) {
				var body Response
				err := json.Unmarshal(resp.Body.Bytes(), &body)
				require.NoError(t, err)
				assert.Equal(t, string(fixedAccessToken), body.AccessToken)

				cookies := resp.Result().Cookies()
				require.Len(t, cookies, 1)
				assert.Equal(t, "refresh_token", cookies[0].Name)
				assert.Equal(t, fixedRefreshToken, cookies[0].Value)
				assert.True(t, cookies[0].HttpOnly)
				assert.Equal(t, http.SameSiteLaxMode, cookies[0].SameSite)
				assert.Equal(t, fixedExpiry, cookies[0].Expires)
			},
		},
		{
			name: "error - usecase credentials error",
			requestBody: Request{
				Login:    "bad-user",
				Password: "bad-password",
			},
			prepare: func(f *fields) {
				f.usecase.LoginFunc = func(ctx context.Context, in Input) (Output, error) {
					return Output{}, errs.ErrInvalidCredentials
				}
			},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "error - empty body",
			requestBody:    nil,
			prepare:        func(f *fields) {},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fields{usecase: &MockUsecase{}}
			tt.prepare(f)
			handler := NewHttpHandler(f.usecase)

			var buf bytes.Buffer
			if tt.requestBody != nil {
				_ = json.NewEncoder(&buf).Encode(tt.requestBody)
			}
			req := httptest.NewRequest(http.MethodPost, "/auth/login", &buf)

			w := httptest.NewRecorder()
			handler.Login(w, req.WithContext(test.ContextWithZeroLogger()))

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}
