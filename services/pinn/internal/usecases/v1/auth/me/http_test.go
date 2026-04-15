package authMe

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AridanWarlock/pinnAutomizer/pinn/internal/domain/fixtures"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHttpHandler_Me(t *testing.T) {
	type fields struct {
		usecase *MockUsecase
	}

	var (
		fixedUser = fixtures.NewUser()
	)

	tests := []struct {
		name           string
		prepare        func(f *fields)
		expectedStatus int
		checkResponse  func(t *testing.T, resp *httptest.ResponseRecorder)
	}{
		{
			name: "success path",
			prepare: func(f *fields) {
				f.usecase.MeFunc = func(ctx context.Context) (Output, error) {
					return Output{
						fixedUser,
					}, nil
				}
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *httptest.ResponseRecorder) {
				var body Response
				err := json.Unmarshal(resp.Body.Bytes(), &body)
				require.NoError(t, err)

				assert.Equal(t, fixedUser.ID, body.ID)
				assert.Equal(t, fixedUser.Login, body.Login)
			},
		},
		{
			name: "error - me not found",
			prepare: func(f *fields) {
				f.usecase.MeFunc = func(ctx context.Context) (Output, error) {
					return Output{}, errs.ErrNotFound
				}
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "error - usecase fail",
			prepare: func(f *fields) {
				f.usecase.MeFunc = func(ctx context.Context) (Output, error) {
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

			ctx := test.ContextWithZeroLogger()

			req := httptest.NewRequest(http.MethodGet, "/auth/me", nil).
				WithContext(ctx)
			w := httptest.NewRecorder()

			handler.Me(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}
