package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"pinnAutomizer/config"
	"pinnAutomizer/internal/adapter/kafka_consumer/at_least_once"
	"pinnAutomizer/internal/adapter/kafka_produce"
	postgresAdapter "pinnAutomizer/internal/adapter/postgres"
	"pinnAutomizer/internal/adapter/redis"
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
	"pinnAutomizer/pkg/auth/jwt/access_token"
	"pinnAutomizer/pkg/auth/refresh_token"
	"pinnAutomizer/pkg/httpserver"
	"pinnAutomizer/pkg/log"
	"pinnAutomizer/pkg/password_hasher"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/segmentio/kafka-go"
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

	redisAdapter := redis.New(c.Redis)
	defer redisAdapter.Close()

	kafkaProducer := kafka_produce.New(c.KafkaProducer, log)
	defer kafkaProducer.Close()
	log.Info().Msg("kafka_produce connected")

	kafkaConsumer := at_least_once.New(kafkaProducer, log)

	writer := outbox.New(postgres, kafkaProducer, log)
	defer writer.Close()
	log.Info().Msg("outbox worker started")

	accessTokenGenerator := access_token.New(c.AccessTokenGenerator)
	refreshTokenGenerator := refresh_token.New(c.RefreshTokenGenerator)
	passwordHasher := password_hasher.New()

	//auth
	login.New(postgres, accessTokenGenerator, refreshTokenGenerator, passwordHasher, log)
	logout.New(postgres, log)
	me.New(postgres, log)
	register.New(postgres, passwordHasher, log)
	refresh.New(postgres, accessTokenGenerator, log)

	//tasks
	create_task.New(postgres, redisAdapter, log)
	get_tasks.New(postgres, log)
	solve_task.New(postgres, redisAdapter, log)
	update_task_status_after_train.New(postgres, redisAdapter, log)
	update_task_status_on_train.New(postgres, redisAdapter, log)

	log.Info().Msg("usecases injected")

	//kafka consumers
	//tasks
	updateTaskStatusAfterTrainConsumer := update_task_status_after_train.NewConsumer(log)
	updateTaskStatusOnTrainConsumer := update_task_status_on_train.NewConsumer(log)

	kafkaConsumer.Run(ctx, "dev.tasks.solver.on-train.v1", updateTaskStatusOnTrainConsumer.HandleMessage)
	kafkaConsumer.Run(ctx, "dev.tasks.solver.after-train.v1", updateTaskStatusAfterTrainConsumer.HandleMessage)

	//middleware
	authMiddleware := auth.NewMiddleware(accessTokenGenerator, log)
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

func newKafkaReader(topic string) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers:                nil,
		GroupID:                "",
		GroupTopics:            nil,
		Topic:                  topic,
		Partition:              0,
		Dialer:                 nil,
		QueueCapacity:          0,
		MinBytes:               0,
		MaxBytes:               0,
		MaxWait:                0,
		ReadBatchTimeout:       0,
		ReadLagInterval:        0,
		GroupBalancers:         nil,
		HeartbeatInterval:      0,
		CommitInterval:         0,
		PartitionWatchInterval: 0,
		WatchPartitionChanges:  false,
		SessionTimeout:         0,
		RebalanceTimeout:       0,
		JoinGroupBackoff:       0,
		RetentionTime:          0,
		StartOffset:            0,
		ReadBackoffMin:         0,
		ReadBackoffMax:         0,
		Logger:                 nil,
		ErrorLogger:            nil,
		IsolationLevel:         0,
		MaxAttempts:            0,
		OffsetOutOfRangeError:  false,
	})
}
