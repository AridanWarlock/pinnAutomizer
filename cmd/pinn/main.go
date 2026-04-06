package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/AridanWarlock/pinnAutomizer/config"
	"github.com/AridanWarlock/pinnAutomizer/internal/adapter/kafka_produce"
	"github.com/AridanWarlock/pinnAutomizer/internal/adapter/postgres"
	"github.com/AridanWarlock/pinnAutomizer/internal/adapter/redis"
	"github.com/AridanWarlock/pinnAutomizer/internal/outbox"
	core_http_middleware "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/middleware"
	core_http_server "github.com/AridanWarlock/pinnAutomizer/internal/transport/http/server"
	auth_login "github.com/AridanWarlock/pinnAutomizer/internal/usecases/v1/auth/login"
	auth_logout "github.com/AridanWarlock/pinnAutomizer/internal/usecases/v1/auth/logout"
	auth_me "github.com/AridanWarlock/pinnAutomizer/internal/usecases/v1/auth/me"
	auth_refresh "github.com/AridanWarlock/pinnAutomizer/internal/usecases/v1/auth/refresh"
	auth_register "github.com/AridanWarlock/pinnAutomizer/internal/usecases/v1/auth/register"
	tasks_after_train "github.com/AridanWarlock/pinnAutomizer/internal/usecases/v1/task/after_train"
	tasks_create "github.com/AridanWarlock/pinnAutomizer/internal/usecases/v1/task/create"
	tasks_get "github.com/AridanWarlock/pinnAutomizer/internal/usecases/v1/task/get"
	tasks_on_train "github.com/AridanWarlock/pinnAutomizer/internal/usecases/v1/task/on_train"
	tasks_solve "github.com/AridanWarlock/pinnAutomizer/internal/usecases/v1/task/solve"
	"github.com/AridanWarlock/pinnAutomizer/pkg/auth/jwt/access_token"
	"github.com/AridanWarlock/pinnAutomizer/pkg/auth/refresh_token"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/AridanWarlock/pinnAutomizer/pkg/password_hasher"
	"github.com/rs/zerolog"
)

// @title 		PINN Automizer App
// @version 	1.0
// @host 		127.0.0.1:8080
// @BasePath 	/api/v1
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

	//adapters
	//postgres
	postgresAdapter, err := postgres.New(cfg.Postgres)
	if err != nil {
		return fmt.Errorf("postgres connect: %w", err)
	}
	defer func() {
		postgresAdapter.Close()
		log.Info().Msg("postgres pool shutdown gracefully")
	}()
	log.Info().Msg("postgres connected")
	//redis
	redisAdapter, err := redis.New(cfg.Redis)
	if err != nil {
		return fmt.Errorf("redis connect: %w", err)
	}
	defer func() {
		if err := redisAdapter.Close(); err != nil {
			log.Error().Err(err).Msg("shutdown redis error")
			return
		}
		log.Info().Msg("redis shutdown gracefully")
	}()
	//kafka producer
	kafkaProducer := kafka_produce.New(cfg.KafkaProducer)
	defer func() {
		if err := kafkaProducer.Close(); err != nil {
			log.Error().Err(err).Msg("kafka producer shutdown error")
			return
		}
		log.Info().Msg("kafka producer shutdown gracefully")
	}()
	log.Info().Msg("kafka_produce connected")
	//outbox writer
	writer := outbox.New(postgresAdapter, kafkaProducer, log)
	defer func() {
		writer.Close()
		log.Info().Msg("outbox writer shutdown gracefully")
	}()
	log.Info().Msg("outbox worker started")
	//access token generator
	accessTokenGenerator := access_token.New(cfg.AccessTokenGenerator)
	//refresh token generator
	refreshTokenGenerator := refresh_token.New(cfg.RefreshTokenGenerator)
	//password hasher
	passwordHasher := password_hasher.New()

	//usecases
	//auth
	authLoginUsecase := auth_login.New(postgresAdapter, accessTokenGenerator, refreshTokenGenerator, passwordHasher)
	authLogoutUsecase := auth_logout.New(postgresAdapter)
	authMeUsecase := auth_me.New(postgresAdapter)
	authRegisterUsecase := auth_register.New(postgresAdapter, passwordHasher)
	authRefreshUsecase := auth_refresh.New(postgresAdapter, accessTokenGenerator)
	//tasks
	tasksCreateUsecase := tasks_create.New(postgresAdapter, redisAdapter)
	tasksGetUsecase := tasks_get.New(postgresAdapter)
	tasksSolveUsecase := tasks_solve.New(postgresAdapter, redisAdapter)
	_ = tasks_after_train.New(postgresAdapter, redisAdapter)
	_ = tasks_on_train.New(postgresAdapter, redisAdapter)

	//http handlers
	//auth
	authLoginHandler := auth_login.NewHttpHandler(authLoginUsecase)
	authLogoutHandler := auth_logout.NewHttpHandler(authLogoutUsecase)
	authMeHandler := auth_me.NewHttpHandler(authMeUsecase)
	authRegisterHandler := auth_register.NewHttpHandler(authRegisterUsecase)
	authRefreshHandler := auth_refresh.NewHttpHandler(authRefreshUsecase)
	//tasks
	tasksCreateHandler := tasks_create.NewHttpHandler(tasksCreateUsecase)
	tasksGetHandler := tasks_get.NewHttpHandler(tasksGetUsecase)
	tasksSolveHandler := tasks_solve.NewHttpHandler(tasksSolveUsecase)

	apiV1Router := core_http_server.NewApiVersionRouter(core_http_server.ApiVersion1)
	apiV1Router.RegisterHandlers(
		//auth
		authLoginHandler,
		authLogoutHandler,
		authMeHandler,
		authRegisterHandler,
		authRefreshHandler,
		//tasks
		tasksCreateHandler,
		tasksGetHandler,
		tasksSolveHandler,
	)

	httpServer := core_http_server.New(
		cfg.HTTP,
		log,
		core_http_middleware.NewChiCORS(core_http_middleware.DefaultCORSConfig()),
		core_http_middleware.RequestID(),
		core_http_middleware.Logger(log),
		core_http_middleware.Recover(),
		core_http_middleware.TraceID(),
		core_http_middleware.Authentication(accessTokenGenerator),
	)
	httpServer.RegisterApiRouters(apiV1Router)

	return httpServer.Run(ctx)
}
