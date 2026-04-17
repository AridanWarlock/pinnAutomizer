package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/AridanWarlock/pinnAutomizer/gateway/internal/config"
	"github.com/AridanWarlock/pinnAutomizer/gateway/internal/transport/http/middleware"
	"github.com/AridanWarlock/pinnAutomizer/gateway/internal/transport/http/proxy"
	"github.com/AridanWarlock/pinnAutomizer/pkg/adapter/redis"
	"github.com/AridanWarlock/pinnAutomizer/pkg/adapter/redis/goRedis"
	"github.com/AridanWarlock/pinnAutomizer/pkg/jwt"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	coreMiddleware "github.com/AridanWarlock/pinnAutomizer/pkg/transport/http/middleware"
	"github.com/AridanWarlock/pinnAutomizer/pkg/transport/http/server"
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
	// access token generator
	accessTokenGenerator := jwt.NewGenerator(cfg.AccessTokenGenerator)
	// http server
	httpServer := server.NewWithoutDefaultMiddlewares(
		cfg.HTTP,
		log,
		middleware.Cors(),
		coreMiddleware.RequestID(),
		coreMiddleware.Logger(log),
		coreMiddleware.TraceID(),
		coreMiddleware.Recover(),
		middleware.AuditInfo(),
		middleware.Auth(redisAdapter, accessTokenGenerator),
	)

	authProxy, err := proxy.NewServiceProxy(cfg.App.AuthAddr)
	if err != nil {
		return fmt.Errorf("create auth proxy: %w", err)
	}
	tasksProxy, err := proxy.NewServiceProxy(cfg.App.TasksAddr)
	if err != nil {
		return fmt.Errorf("create auth proxy: %w", err)
	}

	httpServer.RegisterHandlers(
		server.HttpHandler{
			Pattern: "/api/v1/auth/",
			Handler: authProxy,
		},
		server.HttpHandler{
			Pattern: "/api/v1/tasks/",
			Handler: tasksProxy,
		},
	)

	// httpServer.RegisterSwagger()

	return httpServer.Run(ctx)
}
