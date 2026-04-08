package authRefresh

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AridanWarlock/pinnAutomizer/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/internal/domain/fixtures"
	"github.com/AridanWarlock/pinnAutomizer/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHttpHandler_Refresh(t *testing.T) {
	type fields struct {
		usecase *MockUsecase
	}

	var (
		fixedFingerprint = fixtures.NewFingerprint()
		fixedTokenStr    = "session_id.refresh_token_value"
		fixedAccess      = fixtures.NewAccessToken()
		testCtx          = test.ContextBackgroundWithZeroLogger()
	)

	tests := []struct {
		name           string
		prepareContext func(ctx context.Context) context.Context
		cookie         *http.Cookie
		prepare        func(f *fields)
		expectedStatus int
		shouldPanic    bool
		checkResponse  func(t *testing.T, resp *httptest.ResponseRecorder)
	}{
		{
			name: "success path",
			prepareContext: func(ctx context.Context) context.Context {
				return test.ContextWithUserClaims(ctx, fixtures.NewUserClaims(func(uc *domain.UserClaims) {
					uc.Fingerprint = fixedFingerprint
				}))
			},
			cookie: &http.Cookie{Name: "refreshToken", Value: fixedTokenStr},
			prepare: func(f *fields) {
				f.usecase.RefreshFunc = func(ctx context.Context, in Input) (Output, error) {
					assert.Equal(t, fixedTokenStr, in.RefreshTokenString)
					assert.Equal(t, fixedFingerprint, in.Fingerprint)
					return Output{AccessToken: fixedAccess}, nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *httptest.ResponseRecorder) {
				var body Response
				err := json.Unmarshal(resp.Body.Bytes(), &body)
				require.NoError(t, err)
				assert.Equal(t, string(fixedAccess), body.AccessToken)
			},
		},
		{
			name: "error - no cookie",
			prepareContext: func(ctx context.Context) context.Context {
				return test.ContextWithUserClaims(ctx, fixtures.NewUserClaims(func(uc *domain.UserClaims) {
					uc.Fingerprint = fixedFingerprint
				}))
			},
			cookie:         nil,
			prepare:        func(f *fields) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "error - usecase fail",
			prepareContext: func(ctx context.Context) context.Context {
				return test.ContextWithUserClaims(ctx, fixtures.NewUserClaims(func(uc *domain.UserClaims) {
					uc.Fingerprint = fixedFingerprint
				}))
			},
			cookie: &http.Cookie{Name: "refreshToken", Value: fixedTokenStr},
			prepare: func(f *fields) {
				f.usecase.RefreshFunc = func(ctx context.Context, in Input) (Output, error) {
					return Output{}, errors.New("invalid refresh token")
				}
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "critical - panic when claims missing",
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

			req := httptest.NewRequest(http.MethodPost, "/auth/refresh", nil)
			req = req.WithContext(tt.prepareContext(testCtx))

			if tt.cookie != nil {
				req.AddCookie(tt.cookie)
			}

			w := httptest.NewRecorder()

			if tt.shouldPanic {
				assert.Panics(t, func() {
					handler.Refresh(w, req)
				})
				return
			}

			handler.Refresh(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}
