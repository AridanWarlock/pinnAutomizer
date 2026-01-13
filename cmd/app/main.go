package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"pinnAutomizer/config"
	"pinnAutomizer/internal/adapter/kafka"
	postgresAdapter "pinnAutomizer/internal/adapter/postgres"
	"pinnAutomizer/internal/adapter/translator"
	"pinnAutomizer/internal/auth/login"
	"pinnAutomizer/internal/auth/logout"
	"pinnAutomizer/internal/auth/me"
	"pinnAutomizer/internal/auth/refresh"
	"pinnAutomizer/internal/auth/register"
	"pinnAutomizer/internal/controller/http_v1"
	"pinnAutomizer/internal/middleware/auth"
	"pinnAutomizer/internal/script/create_script"
	"pinnAutomizer/internal/script/search_scripts"
	"pinnAutomizer/internal/script/update_script_after_translate"
	"pinnAutomizer/internal/task/create_task"
	"pinnAutomizer/internal/task/get_tasks"
	"pinnAutomizer/pkg/httpserver"
	"pinnAutomizer/pkg/jwt"
	"pinnAutomizer/pkg/log"
	"pinnAutomizer/pkg/password_hasher"
	"syscall"

	"github.com/rs/zerolog"
)

func main() {
	cfg, err := config.InitConfig()
	if err != nil {
		panic(err)
	}

	logger, err := log.New(cfg.App.Env, cfg.Log)
	if err != nil {
		panic(err)
	}
	logger.Info().Msg("zerolog logger configured")

	err = AppRun(context.Background(), cfg, logger)
	if err != nil {
		panic(err)
	}
}

func AppRun(
	ctx context.Context,
	c config.Config,
	log zerolog.Logger,
) error {
	postgres, err := postgresAdapter.New(ctx, c.Postgres)
	if err != nil {
		return fmt.Errorf("db connect error, %w", err)
	}
	defer postgres.Close()
	log.Info().Msg("postgres connected")

	kf, err := kafka.New(c.Kafka)
	if err != nil {
		return fmt.Errorf("kafka connection error, %w", err)
	}
	defer kf.Close()
	log.Info().Msg("kafka connected")

	jwtService := jwt.New(c.Jwt, postgres)
	log.Info().Msg("jwt service started")
	translatorService := translator.New(c.Translator, log)
	log.Info().Msg("translator service started")
	passwordHasher := password_hasher.New()

	//auth
	login.New(postgres, jwtService, passwordHasher, log)
	logout.New(postgres, log)
	me.New(postgres, log)
	register.New(postgres, passwordHasher, log)
	refresh.New(postgres, jwtService, log)

	//script
	create_script.New(postgres, translatorService, log)
	search_scripts.New(postgres, log)
	update_script_after_translate.New(postgres, log)

	//tasks
	create_task.New(postgres, kf, log)
	get_tasks.New(postgres, log)

	log.Info().Msg("use cases injected")

	authMiddleware := auth.NewMiddleware(jwtService, log)
	router := http_v1.Router(authMiddleware, log)
	httpServer := httpserver.New(router, c.HTTP, log)
	defer httpServer.Shutdown(ctx)

	go func() {
		log.Info().Msg("http server listening...")
		if err := httpServer.ListenAndServer(); err != nil {
			log.Error().Msg("server stopped")
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	return nil
}
