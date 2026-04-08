package authLogin

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain/fixtures"
	"github.com/AridanWarlock/pinnAutomizer/internal/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHttpHandler_Login(t *testing.T) {
	type fields struct {
		usecase *MockUsecase
	}

	var (
		fixedFingerprint       = "66696e6765727072696e74"
		fixedFingerprintRaw, _ = hex.DecodeString(fixedFingerprint)
		fixedAccessToken       = fixtures.NewAccessToken()
		fixedRefreshToken      = "session_id.refresh_token"
		fixedExpiry            = time.Now().Add(time.Hour).Truncate(time.Second)
	)

	tests := []struct {
		name           string
		requestBody    interface{}
		headers        map[string]string
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
			headers: map[string]string{"X-Fingerprint": fixedFingerprint},
			prepare: func(f *fields) {
				f.usecase.LoginFunc = func(ctx context.Context, in Input) (Output, error) {
					assert.Equal(t, "admin", in.Login)
					assert.Equal(t, fixedFingerprintRaw, in.Fingerprint)

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
				assert.Equal(t, "refreshToken", cookies[0].Name)
				assert.Equal(t, fixedRefreshToken, cookies[0].Value)
				assert.True(t, cookies[0].HttpOnly)
				assert.Equal(t, http.SameSiteStrictMode, cookies[0].SameSite)
			},
		},
		{
			name: "error - invalid fingerprint hex",
			requestBody: Request{
				Login:    "admin",
				Password: "password",
			},
			headers:        map[string]string{"X-Fingerprint": "not-hex-value"},
			prepare:        func(f *fields) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "error - usecase credentials error",
			requestBody: Request{
				Login:    "bad-user",
				Password: "bad-password",
			},
			headers: map[string]string{"X-Fingerprint": fixedFingerprint},
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
			headers:        map[string]string{"X-Fingerprint": fixedFingerprint},
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

			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			w := httptest.NewRecorder()
			handler.Login(w, req.WithContext(test.ContextBackgroundWithZeroLogger()))

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}
