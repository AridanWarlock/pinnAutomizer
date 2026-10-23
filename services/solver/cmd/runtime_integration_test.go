package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/docker"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/pkg/kafka"
	"github.com/AridanWarlock/pinnAutomizer/pkg/kafka/segmentio"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/adapter/sqlite"
)

// Requires a fresh isolated broker with the two queue/retry topics and two result topics.
func TestRuntimeKafkaDockerAcceptRecoveryOutboxAndShutdownIntegration(t *testing.T) {
	broker := os.Getenv("PINN_TEST_KAFKA_BROKER")
	if broker == "" || os.Getenv("PINN_TEST_DOCKER") != "1" {
		t.Skip("set PINN_TEST_KAFKA_BROKER and PINN_TEST_DOCKER=1 for an isolated Kafka/Docker runtime test")
	}
	cfg, db := testConfig(t)
	cfg.KafkaConsumer = kafka.ConsumerConfig{
		Broker:  broker,
		GroupID: "solver-runtime-" + uuid.NewString(),
	}
	cfg.KafkaProducer = kafka.ProducerConfig{Broker: broker}
	if image := os.Getenv("PINN_TEST_DOCKER_IMAGE"); image != "" {
		cfg.PinnRunner.Image = image
	}
	cli, err := docker.New()
	require.NoError(t, err)
	defer cli.Close()
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	runID, taskID := uuid.New(), uuid.New()
	name := cfg.PinnRunner.ContainerNamePrefix + "-" + runID.String()
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := cli.Remove(cleanup, name)
		if !errors.Is(err, errs.ErrNotFound) {
			require.NoError(t, err)
		}
	}()
	// The production runner adopts this owned container and sends its only Start.
	// Its CPU command replaces PINN computation for the integration test.
	id, err := cli.Create(ctx, docker.Config{
		Name:    name,
		Image:   cfg.PinnRunner.Image,
		Command: []string{"sh", "-c", "exit 0"},
		Labels: map[string]string{
			"pinn.solver.run_id":  runID.String(),
			"pinn.solver.task_id": taskID.String(),
			"pinn.solver.mode":    "predict",
		},
		NetworkMode: "none",
	})
	require.NoError(t, err)
	before, err := cli.Inspect(ctx, id)
	require.NoError(t, err)
	require.True(t, before.IsCreated())

	done := make(chan struct{})
	var runErr error
	go func() {
		defer close(done)
		runErr = run(ctx, cfg, zerolog.Nop())
	}()
	defer func() {
		cancel()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Error("runtime did not join workers on cancellation")
		}
	}()
	producer := kafka.NewProducer(cfg.KafkaProducer, zerolog.Nop())
	defer producer.Close()
	command, err := core.NewProduceKafkaMessage(
		"tasks.run.queue",
		[]byte(taskID.String()),
		fmt.Appendf(nil, `{"task_id":%q,"mode":"predict"}`, taskID.String()),
		core.KafkaHeaders{core.KafkaIdempotencyHeader: runID.String()},
	)
	require.NoError(t, err)
	// Both queue consumers must use the same durable identity.
	duplicate := command
	duplicate.Topic = "tasks.predict.queue"
	require.NoError(t, producer.WriteMessages(ctx, command, duplicate))
	require.Eventually(t, func() bool {
		select {
		case <-done:
			return true
		default:
		}
		var state string
		var cleaned *int64
		if err := db.QueryRow("SELECT state,cleaned_at FROM attempts WHERE run_id=?", runID).Scan(&state, &cleaned); err != nil {
			return false
		}
		var pending int
		if err := db.QueryRow("SELECT COUNT(*) FROM outbox").Scan(&pending); err != nil {
			return false
		}
		return state == "finished" && cleaned != nil && pending == 0
	}, 40*time.Second, 20*time.Millisecond, "runtime must persist, execute, publish and clean")
	select {
	case <-done:
		require.NoError(t, runErr, "runtime stopped before explicit cancellation")
		t.Fatal("runtime stopped before completing the scenario")
	default:
	}
	for _, topic := range []string{"tasks.on.run", "tasks.after.run"} {
		msg := readTaskEvent(t, ctx, broker, topic, taskID)
		require.JSONEq(t, fmt.Sprintf(`{"task_id":%q}`, taskID.String()), string(msg.Value))
		require.Equal(t, uuid.NewSHA1(runID, []byte(topic)).String(), msg.Headers[core.KafkaIdempotencyHeader])
	}
	cancel()
	<-done
	require.NoError(t, runErr)
	_, err = cli.Inspect(t.Context(), id)
	require.ErrorIs(t, err, errs.ErrNotFound)
	// New ownership after Run returns proves dependencies were closed after workers joined.
	journal, err := sqlite.New(cfg.SQLite)
	require.NoError(t, err)
	defer journal.Close()
	stored, err := journal.GetAttempt(t.Context(), runID)
	require.NoError(t, err)
	require.True(t, stored.IsFinished())
	require.NotNil(t, stored.CleanedAt)
	var attempts int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM attempts").Scan(&attempts))
	require.Equal(t, 1, attempts, "duplicate delivery across queues must share a single attempt")
}

func readTaskEvent(t *testing.T, ctx context.Context, broker, topic string, taskID uuid.UUID) segmentio.Message {
	t.Helper()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	reader := segmentio.NewConsumer(segmentio.ConsumerConfig{
		Broker:  broker,
		GroupID: "runtime-result-" + uuid.NewString(),
	}, topic, kafka.ConsumeMessageMaxBytes)
	defer func() { require.NoError(t, reader.Close()) }()
	for {
		msg, err := reader.FetchMessage(ctx)
		require.NoError(t, err)
		if string(msg.Key) == taskID.String() {
			return msg
		}
	}
}
