package postgres

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/auth/internal/domain"
	authRefresh "github.com/AridanWarlock/pinnAutomizer/auth/internal/usecases/v1/auth/refresh"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/crypt"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/jwt"
	sharedRedis "github.com/AridanWarlock/pinnAutomizer/pkg/redis"
	"github.com/AridanWarlock/pinnAutomizer/pkg/redis/goRedis"
)

type refreshIntegration struct {
	postgres  *Repository
	redis     *trackedRefreshRedis
	input     authRefresh.Input
	token     domain.RefreshToken
	generator *jwt.Generator
}

func refreshIntegrationPostgres(t *testing.T, dsn string) *Repository {
	t.Helper()
	cfg, err := pgx.ParseConfig(dsn)
	require.NoError(t, err)
	admin, err := pgx.ConnectConfig(t.Context(), cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		assert.NoError(t, admin.Close(ctx))
	})
	name := "pinn_refresh_" + uuid.NewString()
	quoted := pgx.Identifier{name}.Sanitize()
	_, err = admin.Exec(t.Context(), "CREATE DATABASE "+quoted)
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := admin.Exec(ctx, "DROP DATABASE "+quoted)
		assert.NoError(t, err)
	})
	copy := cfg.Copy()
	copy.Database = name
	db, err := pgx.ConnectConfig(t.Context(), copy)
	require.NoError(t, err)
	defer func() { assert.NoError(t, db.Close(context.Background())) }()
	for _, migration := range []string{
		"20251130110309_create_table_users.sql",
		"20251208094652_create_table_roles.sql",
		"20251208094904_create_table_users_roles.sql",
		"20260412101853_create_table_refresh_tokens.sql",
		"20261025120000_create_table_session_revocations.sql",
	} {
		sql, err := os.ReadFile("../../../migrations/" + migration)
		require.NoError(t, err)
		_, err = db.Exec(t.Context(), string(sql))
		require.NoError(t, err)
	}
	postgres, err := New(Config{
		User:     cfg.User,
		Password: cfg.Password,
		Host:     cfg.Host,
		Port:     int(cfg.Port),
		DB:       name,
		SslMode:  "disable",
		Timeout:  5 * time.Second,
	})
	require.NoError(t, err)
	t.Cleanup(postgres.Close)
	return postgres
}

// SET failures can occur before the server effect or after Redis applied it.
type trackedRefreshRedis struct {
	*sharedRedis.Redis
	mu         sync.Mutex
	keys       map[string]struct{}
	deleted    []string
	setFailure error
	applySet   bool
}

func (r *trackedRefreshRedis) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	r.mu.Lock()
	r.keys[key] = struct{}{}
	r.mu.Unlock()
	if r.setFailure != nil && !r.applySet {
		return fmt.Errorf("%w: %w", errs.ErrServiceUnavailable, r.setFailure)
	}
	if err := r.Redis.Set(ctx, key, value, ttl); err != nil {
		return err
	}
	if r.setFailure != nil {
		return fmt.Errorf("%w: %w", errs.ErrServiceUnavailable, r.setFailure)
	}
	return nil
}

func (r *trackedRefreshRedis) Delete(ctx context.Context, key string) error {
	r.mu.Lock()
	r.deleted = append(r.deleted, key)
	r.mu.Unlock()
	return r.Redis.Delete(ctx, key)
}

func newRefreshIntegration(t *testing.T) refreshIntegration {
	t.Helper()
	dsn, address := os.Getenv("PINN_TEST_POSTGRES_DSN"), os.Getenv("PINN_TEST_REDIS_ADDR")
	if dsn == "" || address == "" {
		t.Skip("set PINN_TEST_POSTGRES_DSN and PINN_TEST_REDIS_ADDR to disposable services")
	}
	postgres := refreshIntegrationPostgres(t, dsn)
	client, err := goRedis.New(goRedis.Config{
		Addr:         address,
		DialTimeout:  time.Second,
		ReadTimeout:  time.Second,
		WriteTimeout: time.Second,
		PoolTimeout:  time.Second,
		PoolSize:     4,
	})
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, client.Close()) })
	redis := &trackedRefreshRedis{
		Redis: sharedRedis.NewRedis(client),
		keys:  make(map[string]struct{}),
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		for key := range redis.keys {
			assert.NoError(t, redis.Redis.Delete(ctx, key))
		}
	})
	user, err := domain.NewUser(uuid.NewString(), "test-password-hash")
	require.NoError(t, err)
	_, err = postgres.CreateUser(t.Context(), user)
	require.NoError(t, err)
	role, err := core.NewRole(uuid.New(), string(core.RoleTypeUser))
	require.NoError(t, err)
	_, err = postgres.pool.Execx(t.Context(), postgres.sb.Insert(RolesTable).
		Columns(RolesID, RolesTitle).
		Values(role.ID, role.Title))
	require.NoError(t, err)
	_, err = postgres.pool.Execx(t.Context(), postgres.sb.Insert(UsersRolesTable).
		Columns(UsersRolesUserID, UsersRolesRoleID).
		Values(user.ID, role.ID))
	require.NoError(t, err)
	fingerprint, err := core.NewFingerprint(strings.Repeat("a", 64))
	require.NoError(t, err)
	ip, err := core.NewUserIP("127.0.0.1")
	require.NoError(t, err)
	agent, err := core.NewUserAgent("refresh-integration")
	require.NoError(t, err)
	generator := jwt.NewGenerator(jwt.Config{Secret: "refresh-integration-only-secret"})
	_, claims, err := generator.GenerateAndGetClaims(user.ID)
	require.NoError(t, err)
	secret := crypt.GenerateSecureToken()
	token, err := domain.NewRefreshToken(crypt.Sha256(secret), user.ID, claims.Jti, core.NewAuditInfo(fingerprint, ip, agent), time.Hour)
	require.NoError(t, err)
	token, err = postgres.Login(t.Context(), token)
	require.NoError(t, err)
	session, err := core.NewRedisSession(user.ID, []core.Role{role}, fingerprint, claims.IssuedAt)
	require.NoError(t, err)
	require.NoError(t, redis.Set(t.Context(), claims.Jti.ToRedisKey(), session, time.Minute))
	return refreshIntegration{
		postgres:  postgres,
		redis:     redis,
		input:     authRefresh.Input{RefreshTokenString: secret},
		token:     token,
		generator: generator,
	}
}
