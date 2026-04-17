package authRegister

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AridanWarlock/pinnAutomizer/auth/internal/domain"
	"github.com/AridanWarlock/pinnAutomizer/auth/internal/domain/domainfixtures"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/test"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHttpHandler_Register(t *testing.T) {
	type fields struct {
		usecase *MockUsecase
	}

	var (
		fixedID    = uuid.New()
		fixedLogin = "new_user"
		testCtx    = test.ContextWithZeroLogger()
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
				Login:             fixedLogin,
				Password:          "password123",
				PasswordConfirmed: "password123",
			},
			prepare: func(f *fields) {
				f.usecase.RegisterFunc = func(ctx context.Context, in Input) (Output, error) {
					assert.Equal(t, fixedLogin, in.Login)
					return Output{
						User: domainfixtures.NewUser(func(user *domain.User) {
							user.Login = fixedLogin
						}),
					}, nil
				}
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, resp *httptest.ResponseRecorder) {
				var body Response
				err := json.Unmarshal(resp.Body.Bytes(), &body)
				require.NoError(t, err)
				assert.Equal(t, fixedID, body.ID)
				assert.Equal(t, fixedLogin, body.Login)
			},
		},
		{
			name: "error - login already taken",
			requestBody: Request{
				Login:             "admin",
				Password:          "password",
				PasswordConfirmed: "password",
			},
			prepare: func(f *fields) {
				f.usecase.RegisterFunc = func(ctx context.Context, in Input) (Output, error) {
					return Output{}, errs.ErrConflict
				}
			},
			expectedStatus: http.StatusConflict,
		},
		{
			name: "error - invalid request body (validation failed)",
			requestBody: Request{
				Login:             "",
				Password:          "1",
				PasswordConfirmed: "2",
			},
			prepare:        func(f *fields) {},
			expectedStatus: http.StatusBadRequest,
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

			req := httptest.NewRequest(http.MethodPost, "/auth/register", &buf)
			w := httptest.NewRecorder()

			handler.Register(w, req.WithContext(testCtx))

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.checkResponse != nil {
				tt.checkResponse(t, w)
			}
		})
	}
}
