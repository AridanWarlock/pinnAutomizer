package authMe

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
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHttpHandler_Me(t *testing.T) {
	type fields struct {
		usecase *MockUsecase
	}

	var (
		fixedUserID = uuid.New()
		fixedLogin  = "admin_user"
		testCtx     = test.ContextBackgroundWithZeroLogger()
	)

	tests := []struct {
		name           string
		prepareContext func(ctx context.Context) context.Context
		prepare        func(f *fields)
		expectedStatus int
		shouldPanic    bool
		checkResponse  func(t *testing.T, resp *httptest.ResponseRecorder)
	}{
		{
			name: "success path",
			prepareContext: func(ctx context.Context) context.Context {
				return test.ContextWithUserClaims(ctx, fixtures.NewUserClaims(func(uc *domain.UserClaims) {
					uc.UserID = fixedUserID
				}))
			},
			prepare: func(f *fields) {
				f.usecase.MeFunc = func(ctx context.Context, in Input) (Output, error) {
					assert.Equal(t, fixedUserID, in.UserID)
					return Output{
						UserID: fixedUserID,
						Login:  fixedLogin,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *httptest.ResponseRecorder) {
				var body Response
				err := json.Unmarshal(resp.Body.Bytes(), &body)
				require.NoError(t, err)

				assert.Equal(t, fixedUserID, body.ID)
				assert.Equal(t, fixedLogin, body.Login)
			},
		},
		{
			name: "error - usecase fail",
			prepareContext: func(ctx context.Context) context.Context {
				return test.ContextWithUserClaims(ctx, fixtures.NewUserClaims(func(uc *domain.UserClaims) {
					uc.UserID = fixedUserID
				}))
			},
			prepare: func(f *fields) {
				f.usecase.MeFunc = func(ctx context.Context, in Input) (Output, error) {
					return Output{}, errors.New("me failed")
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

			req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
			req = req.WithContext(tt.prepareContext(testCtx))

			w := httptest.NewRecorder()

			if tt.shouldPanic {
				assert.Panics(t, func() {
					handler.Me(w, req)
				}, "Ожидалась паника из-за отсутствия Claims в контексте")
				return
			}

			handler.Me(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}
