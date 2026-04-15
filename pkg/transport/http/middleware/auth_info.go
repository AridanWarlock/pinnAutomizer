package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/AridanWarlock/pinnAutomizer/pkg/transport/http/response"
	"github.com/google/uuid"
)

func AuthInfo() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			auth, err := authInfoFromHeaders(r.Header)
			if err == nil {
				ctx = auth.WithContext(ctx)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			log := logger.FromContext(ctx)
			rh := response.NewHandler(w, log)
			rh.ErrorResponse(
				fmt.Errorf("auth info from headers: %w", err),
				"failed on parse auth info from headers",
			)
		})
	}
}

func authInfoFromHeaders(headers http.Header) (core.AuthInfo, error) {
	jtiUuid, err := uuid.Parse(headers.Get(core.JtiHeader))
	if err != nil {
		return core.AuthInfo{}, fmt.Errorf("parse uuid jti from headers: %w", err)
	}
	jti, err := core.NewJti(jtiUuid)
	if err != nil {
		return core.AuthInfo{}, err
	}

	userID, err := uuid.Parse(headers.Get(core.UserIDHeader))
	if err != nil {
		return core.AuthInfo{}, fmt.Errorf("invalid uuid in user id header")
	}

	var roles []core.Role
	err = json.Unmarshal([]byte(headers.Get(core.RolesHeader)), &roles)
	if err != nil {
		return core.AuthInfo{}, fmt.Errorf("parse roles from headers: %w", err)
	}

	issuedAt, err := time.Parse(time.RFC3339, headers.Get(core.IssuedAtHeader))
	if err != nil {
		return core.AuthInfo{}, fmt.Errorf("parse issued at from headers: %w", err)
	}

	return core.NewAuthInfo(jti, userID, roles, issuedAt)
}
