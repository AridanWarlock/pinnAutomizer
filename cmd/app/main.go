package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"pinnAutomizer/config"
	"pinnAutomizer/internal/adapter/kafka_produce"
	postgresAdapter "pinnAutomizer/internal/adapter/postgres"
	"pinnAutomizer/internal/controller/http_v1"
	"pinnAutomizer/internal/middleware/auth"
	"pinnAutomizer/internal/outbox"
	"pinnAutomizer/internal/usecases/auth/login"
	"pinnAutomizer/internal/usecases/auth/logout"
	"pinnAutomizer/internal/usecases/auth/me"
	"pinnAutomizer/internal/usecases/auth/refresh"
	"pinnAutomizer/internal/usecases/auth/register"
	"pinnAutomizer/internal/usecases/task/create_task"
	"pinnAutomizer/internal/usecases/task/get_tasks"
	"pinnAutomizer/internal/usecases/task/solve_task"
	"pinnAutomizer/internal/usecases/task/update_task_status_after_train"
	"pinnAutomizer/internal/usecases/task/update_task_status_on_train"
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
	postgres, err := postgresAdapter.New(ctx, c.Postgres, log)
	if err != nil {
		return fmt.Errorf("db connect error, %w", err)
	}
	defer postgres.Close()
	log.Info().Msg("postgres connected")

	kafka := kafka_produce.New(c.KafkaProducer, log)
	defer kafka.Close()
	log.Info().Msg("kafka_produce connected")

	writer := outbox.New(postgres, kafka, log)
	defer writer.Close()
	log.Info().Msg("outbox worker started")

	jwtService := jwt.New(c.Jwt, postgres)
	log.Info().Msg("jwt service started")
	passwordHasher := password_hasher.New()

	//auth
	login.New(postgres, jwtService, passwordHasher, log)
	logout.New(postgres, log)
	me.New(postgres, log)
	register.New(postgres, passwordHasher, log)
	refresh.New(postgres, jwtService, log)

	//tasks
	create_task.New(postgres, log)
	get_tasks.New(postgres, log)
	solve_task.New(postgres, log)
	update_task_status_after_train.New(postgres, log)
	update_task_status_on_train.New(postgres, log)

	log.Info().Msg("usecases injected")

	//kafka consumers
	//tasks
	consumerAfterTrain := update_task_status_after_train.NewConsumer(c.KafkaConsumerAfterTrain, log)
	defer consumerAfterTrain.Close()
	consumerOnTrain := update_task_status_on_train.NewConsumer(c.KafkaConsumerOnTrain, log)
	defer consumerOnTrain.Close()

	//middleware
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
