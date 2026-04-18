package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/AridanWarlock/pinnAutomizer/auth/internal/adapter/postgres"
	"github.com/AridanWarlock/pinnAutomizer/auth/internal/config"
	authLogin "github.com/AridanWarlock/pinnAutomizer/auth/internal/usecases/v1/auth/login"
	authLogout "github.com/AridanWarlock/pinnAutomizer/auth/internal/usecases/v1/auth/logout"
	authMe "github.com/AridanWarlock/pinnAutomizer/auth/internal/usecases/v1/auth/me"
	authRefresh "github.com/AridanWarlock/pinnAutomizer/auth/internal/usecases/v1/auth/refresh"
	authRegister "github.com/AridanWarlock/pinnAutomizer/auth/internal/usecases/v1/auth/register"
	"github.com/AridanWarlock/pinnAutomizer/pkg/httpsrv"
	"github.com/AridanWarlock/pinnAutomizer/pkg/jwt"
	"github.com/AridanWarlock/pinnAutomizer/pkg/logger"
	"github.com/AridanWarlock/pinnAutomizer/pkg/password"
	"github.com/AridanWarlock/pinnAutomizer/pkg/redis"
	"github.com/AridanWarlock/pinnAutomizer/pkg/redis/goRedis"
	"github.com/rs/zerolog"
)

// @title		PINN Automizer App
// @version	1.0
// @host		127.0.0.1:8080
// @BasePath	/api/v1
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
	// postgres
	postgresAdapter, err := postgres.New(cfg.Postgres)
	if err != nil {
		return fmt.Errorf("postgres connect: %w", err)
	}
	defer func() {
		postgresAdapter.Close()
		log.Info().Msg("postgres pool shutdown gracefully")
	}()
	log.Info().Msg("postgres connected")
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
	// kafka producer
	// access token generator
	accessTokenGenerator := jwt.NewGenerator(cfg.AccessTokenGenerator)
	// password hasher
	hasher := password.NewHasher()

	// usecases
	// auth
	authLoginUsecase := authLogin.New(
		postgresAdapter,
		redisAdapter,
		accessTokenGenerator,
		hasher,
	)
	authLogoutUsecase := authLogout.New(postgresAdapter, redisAdapter)
	authMeUsecase := authMe.New(postgresAdapter)
	authRegisterUsecase := authRegister.New(postgresAdapter, hasher)
	authRefreshUsecase := authRefresh.New(postgresAdapter, redisAdapter, accessTokenGenerator)

	// http handlers
	// auth
	authLoginHandler := authLogin.NewHttpHandler(authLoginUsecase)
	authLogoutHandler := authLogout.NewHttpHandler(authLogoutUsecase)
	authMeHandler := authMe.NewHttpHandler(authMeUsecase)
	authRegisterHandler := authRegister.NewHttpHandler(authRegisterUsecase)
	authRefreshHandler := authRefresh.NewHttpHandler(authRefreshUsecase)
	// routers
	apiV1Router := httpsrv.NewApiVersionRouter(httpsrv.ApiVersion(1))
	apiV1Router.RegisterRoutes(
		// auth
		authLoginHandler.Route(),
		authLogoutHandler.Route(),
		authMeHandler.Route(),
		authRegisterHandler.Route(),
		authRefreshHandler.Route(),
	)
	// http server
	server := httpsrv.NewWithDefaultMiddlewares(
		cfg.HTTP,
		log,
	)
	server.RegisterApiRouters(apiV1Router)
	// httpServer.RegisterSwagger()

	return server.Run(ctx)
}
