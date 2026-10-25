package middleware

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	sharedRedis "github.com/AridanWarlock/pinnAutomizer/pkg/redis"
	"github.com/AridanWarlock/pinnAutomizer/pkg/redis/goRedis"
)

func redisTestAddress(t *testing.T) string {
	t.Helper()
	address := os.Getenv("PINN_TEST_REDIS_ADDR")
	if address == "" {
		t.Skip("set PINN_TEST_REDIS_ADDR to a disposable Redis")
	}
	return address
}

func authRedis(t *testing.T, address string) *sharedRedis.Redis {
	t.Helper()
	client, err := goRedis.New(goRedis.Config{
		Addr:         address,
		DialTimeout:  200 * time.Millisecond,
		ReadTimeout:  200 * time.Millisecond,
		WriteTimeout: 200 * time.Millisecond,
		PoolTimeout:  time.Second,
		PoolSize:     2,
	})
	require.NoError(t, err)
	t.Cleanup(func() { assert.NoError(t, client.Close()) })
	return sharedRedis.NewRedis(client)
}

func storeAuthSession(t *testing.T, redis *sharedRedis.Redis, fixture authFixture, ttl time.Duration) {
	t.Helper()
	key := fixture.claims.Jti.ToRedisKey()
	require.NoError(t, redis.Set(t.Context(), key, fixture.session, ttl))
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		assert.NoError(t, redis.Delete(ctx, key))
	})
}

func TestAuthRejectsRevocationAcrossTwoGatewaysIntegration(t *testing.T) {
	address := redisTestAddress(t)
	fixture := newAuthFixture(t)
	control := authRedis(t, address)
	storeAuthSession(t, control, fixture, time.Minute)
	first := authHandler(authRedis(t, address), fixture)
	second := authHandler(authRedis(t, address), fixture)
	for _, handler := range []http.Handler{first, second} {
		require.Equal(t, http.StatusNoContent, authRequest(t, handler, fixture).Code)
	}
	// Auth logout performs this DEL through a different Redis client/process.
	require.NoError(t, control.Delete(t.Context(), fixture.claims.Jti.ToRedisKey()))
	for _, handler := range []http.Handler{first, second} {
		response := authRequest(t, handler, fixture)
		require.Equal(t, http.StatusUnauthorized, response.Code)
		require.Empty(t, response.Header().Get("X-Test-User"))
	}
}

func TestAuthRejectsRedisTTLExpiryIntegration(t *testing.T) {
	address := redisTestAddress(t)
	fixture := newAuthFixture(t)
	control := authRedis(t, address)
	handler := authHandler(authRedis(t, address), fixture)
	storeAuthSession(t, control, fixture, time.Second)
	require.Equal(t, http.StatusNoContent, authRequest(t, handler, fixture).Code)
	require.Eventually(t, func() bool {
		var session core.RedisSession
		return errors.Is(control.Get(t.Context(), fixture.claims.Jti.ToRedisKey(), &session), errs.ErrKeyNotFound)
	}, 3*time.Second, 5*time.Millisecond)
	_, err := fixture.parser.GetClaims(fixture.token)
	require.NoError(t, err, "JWT itself has not expired: Redis must enforce session expiry")
	require.Equal(t, http.StatusUnauthorized, authRequest(t, handler, fixture).Code)
}

// redisNetworkProxy forwards real Redis traffic until stop cuts existing and new
// connections. It does not stop or flush the Redis supplied by the test environment.
func redisNetworkProxy(t *testing.T, address string) (string, func()) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	done := make(chan struct{})
	var connections []net.Conn
	var copies sync.WaitGroup
	go func() {
		defer close(done)
		for {
			incoming, err := listener.Accept()
			if err != nil {
				return
			}
			outgoing, err := net.DialTimeout("tcp", address, time.Second)
			if err != nil {
				_ = incoming.Close()
				continue
			}
			connections = append(connections, incoming, outgoing)
			copies.Add(2)
			forward := func(destination, source net.Conn) {
				defer copies.Done()
				defer destination.Close()
				defer source.Close()
				_, _ = io.Copy(destination, source)
			}
			go forward(incoming, outgoing)
			go forward(outgoing, incoming)
		}
	}()
	var once sync.Once
	stop := func() {
		once.Do(func() {
			_ = listener.Close()
			<-done // No more appends or WaitGroup.Add calls can occur.
			for _, connection := range connections {
				_ = connection.Close()
			}
			copies.Wait()
		})
	}
	t.Cleanup(stop)
	return listener.Addr().String(), stop
}

func TestAuthReturns503AfterRedisConnectionLossIntegration(t *testing.T) {
	address := redisTestAddress(t)
	fixture := newAuthFixture(t)
	control := authRedis(t, address)
	storeAuthSession(t, control, fixture, time.Minute)
	proxyAddress, disconnect := redisNetworkProxy(t, address)
	handler := authHandler(authRedis(t, proxyAddress), fixture)
	require.Equal(t, http.StatusNoContent, authRequest(t, handler, fixture).Code)
	disconnect()
	response := authRequest(t, handler, fixture)
	require.Equal(t, http.StatusServiceUnavailable, response.Code)
	require.Empty(t, response.Header().Get("X-Test-User"))
}
