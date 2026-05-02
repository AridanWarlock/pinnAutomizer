package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/AridanWarlock/pinnAutomizer/gateway/internal/config"
	"github.com/AridanWarlock/pinnAutomizer/gateway/internal/transport/http/middleware"
	"github.com/AridanWarlock/pinnAutomizer/gateway/internal/transport/http/proxy"
	"github.com/AridanWarlock/pinnAutomizer/pkg/cache"
	"github.com/AridanWarlock/pinnAutomizer/pkg/cacheaside"
	"github.com/AridanWarlock/pinnAutomizer/pkg/httpmv"
	"github.com/AridanWarlock/pinnAutomizer/pkg/httpsrv"
	"github.com/AridanWarlock/pinnAutomizer/pkg/jwt"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/AridanWarlock/pinnAutomizer/pkg/redis"
	"github.com/AridanWarlock/pinnAutomizer/pkg/redis/goRedis"
	"github.com/rs/zerolog"
)

func main() {
	cfg, err := config.InitConfig()
	if err != nil {
		panic(err)
	}

	log, closeLogger, err := logger.New(cfg.Log)
	if err != nil {
		panic(err)
	}
	defer closeLogger()
	log.Info().Msg("logger configured")

	err = AppRun(cfg, log)
	if err != nil {
		panic(err)
	}
}

func AppRun(
	cfg config.Config,
	log zerolog.Logger,
) error {
	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT, syscall.SIGTERM,
	)
	defer cancel()

	// adapters
	// redis
	redisClient, err := goRedis.New(cfg.Redis)
	if err != nil {
		return fmt.Errorf("redis connect: %w", err)
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Error().Err(err).Msg("shutdown redis error")
			return
		}
		log.Info().Msg("redis shutdown gracefully")
	}()
	redisAdapter := redis.NewRedis(redisClient)
	// cache aside
	l1InMemory := cache.NewCache()
	defer l1InMemory.Close()
	cacheAside := cacheaside.NewCacheAside(l1InMemory, redisAdapter)
	// access token generator
	accessTokenGenerator := jwt.NewGenerator(cfg.AccessTokenGenerator)
	// http server
	httpServer := httpsrv.New(
		cfg.HTTP,
		log,
		middleware.Cors(),
		middleware.CleanTrailingSlash(),
		httpmv.RequestID(),
		httpmv.Logger(log),
		httpmv.TraceID(),
		httpmv.Recover(),
		middleware.AuditInfo(),
		middleware.Auth(cacheAside, accessTokenGenerator),
	)

	authProxy, err := proxy.NewServiceProxy(cfg.App.AuthAddr)
	if err != nil {
		return fmt.Errorf("create auth proxy: %w", err)
	}
	tasksProxy, err := proxy.NewServiceProxy(cfg.App.TasksAddr)
	if err != nil {
		return fmt.Errorf("create tasks proxy: %w", err)
	}

	httpServer.RegisterHandlers(
		httpsrv.HttpHandler{
			Pattern: "/api/v1/auth",
			Handler: authProxy,
		},
		httpsrv.HttpHandler{
			Pattern: "/api/v1/auth/",
			Handler: authProxy,
		},

		httpsrv.HttpHandler{
			Pattern: "/api/v1/tasks",
			Handler: tasksProxy,
		},
		httpsrv.HttpHandler{
			Pattern: "/api/v1/tasks/",
			Handler: tasksProxy,
		},
	)

	// httpServer.RegisterSwagger()

	return httpServer.Run(ctx)
}
