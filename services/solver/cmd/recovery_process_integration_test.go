package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/AridanWarlock/pinnAutomizer/pkg/core"
	"github.com/AridanWarlock/pinnAutomizer/pkg/core/limit"
	"github.com/AridanWarlock/pinnAutomizer/pkg/docker"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/adapter/mlrunner"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/adapter/sqlite"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/config"
	tasksAccept "github.com/AridanWarlock/pinnAutomizer/solver/internal/usecases/v1/tasks/accept"
)

func TestRecoverySurvivesProcessKillWithoutRestartingContainerIntegration(t *testing.T) {
	if os.Getenv("PINN_TEST_DOCKER") != "1" {
		t.Skip("set PINN_TEST_DOCKER=1 for disposable CPU process-crash recovery")
	}
	cfg, db := testConfig(t)
	if image := os.Getenv("PINN_TEST_DOCKER_IMAGE"); image != "" {
		cfg.PinnRunner.Image = image
	}
	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	cli, err := docker.New()
	require.NoError(t, err)
	defer cli.Close()
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
	id, err := cli.Create(ctx, docker.Config{
		Name:    name,
		Image:   cfg.PinnRunner.Image,
		Command: []string{"sh", "-c", "trap 'exit 0' TERM; while :; do sleep 1; done"},
		Labels: map[string]string{
			"pinn.solver.run_id":  runID.String(),
			"pinn.solver.task_id": taskID.String(),
			"pinn.solver.mode":    "predict",
		},
		NetworkMode: "none",
	})
	require.NoError(t, err)
	journal, err := sqlite.New(cfg.SQLite)
	require.NoError(t, err)
	// No Kafka is needed: this test isolates crash persistence and the runtime recovery wiring.
	input := tasksAccept.Input{
		RunID:  runID,
		TaskID: taskID,
		Mode:   "predict",
	}
	_, err = tasksAccept.New(journal).AcceptTask(ctx, input)
	require.NoError(t, err)
	require.NoError(t, journal.Close())

	first := startRecoveryProcess(t, ctx, cfg, runID)
	before, err := cli.Inspect(ctx, id)
	require.NoError(t, err)
	require.True(t, before.IsRunning())
	first.kill(t)
	require.Error(t, first.err, "first recovery process must be killed, not gracefully stopped")
	afterKill, err := cli.Inspect(ctx, id)
	require.NoError(t, err)
	require.True(t, afterKill.IsRunning(), "process death must not imply container termination")
	require.Equal(t, before.StartedAt, afterKill.StartedAt)

	second := startRecoveryProcess(t, ctx, cfg, runID)
	recovered, err := cli.Inspect(ctx, id)
	require.NoError(t, err)
	require.Equal(t, before.StartedAt, recovered.StartedAt, "recovery must adopt the running execution")
	// The test ends its CPU workload; no real PINN contract is involved.
	require.NoError(t, cli.Stop(ctx, id, 2*time.Second))
	require.Eventually(t, func() bool {
		var cleaned *int64
		err := db.QueryRow("SELECT cleaned_at FROM attempts WHERE run_id=?", runID).Scan(&cleaned)
		return err == nil && cleaned != nil
	}, 10*time.Second, 20*time.Millisecond)
	second.kill(t)

	journal, err = sqlite.New(cfg.SQLite)
	require.NoError(t, err, "SIGKILL must release the previous SQLite ownership lock")
	defer journal.Close()
	stored, err := journal.GetAttempt(ctx, runID)
	require.NoError(t, err)
	require.True(t, stored.IsFinished())
	require.NotNil(t, stored.CleanedAt)
	require.NotNil(t, stored.ExitCode)
	require.Zero(t, *stored.ExitCode)
	require.Nil(t, stored.Error)
	// A replay after two process deaths is still the same completed attempt.
	out, err := tasksAccept.New(journal).AcceptTask(ctx, input)
	require.NoError(t, err)
	require.Equal(t, core.Created(false), out.Created)
	events, err := journal.GetPendingEvents(ctx, limit.MustNew(10))
	require.NoError(t, err)
	require.Len(t, events, 2)
	require.Equal(t, "tasks.on.run", events[0].Message.Topic)
	require.Equal(t, "tasks.after.run", events[1].Message.Topic)
	for _, event := range events {
		require.Equal(t, uuid.NewSHA1(runID, []byte(event.Message.Topic)), event.ID)
	}
	pending, err := journal.GetRecoverableAttempts(ctx, uuid.Nil, limit.MustNew(10))
	require.NoError(t, err)
	require.Empty(t, pending)
	_, err = cli.Inspect(ctx, id)
	require.ErrorIs(t, err, errs.ErrNotFound)
}

// This is a subprocess entry point, never part of the normal test run.
func TestSolverRecoveryProcessHelper(t *testing.T) {
	if os.Getenv("PINN_RECOVERY_PROCESS_HELPER") != "1" {
		t.Skip("subprocess helper")
	}
	cfg := config.Config{
		SQLite: sqlite.Config{
			Path:          os.Getenv("PINN_RECOVERY_JOURNAL"),
			MigrationsDir: os.Getenv("PINN_RECOVERY_MIGRATIONS"),
			BusyTimeout:   time.Second,
		},
		Recovery: config.Recovery{
			ActiveLimit:    1,
			BatchSize:      2,
			Interval:       10 * time.Millisecond,
			AttemptTimeout: 5 * time.Second,
		},
		PinnRunner: mlrunner.Config{
			ContainerNamePrefix: "pinn-runtime-test",
			Image:               os.Getenv("PINN_RECOVERY_IMAGE"),
			Timeout:             time.Minute,
		},
		OutboxInterval: time.Second,
	}
	journal, err := sqlite.New(cfg.SQLite)
	require.NoError(t, err)
	defer journal.Close()
	runner, err := mlrunner.NewPinnRunner(cfg.PinnRunner, zerolog.Nop())
	require.NoError(t, err)
	defer runner.Close()
	pipe, err := newPipeline(cfg, zerolog.Nop(), journal, runner, disabledProcessKafka{})
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(t.Context(), 40*time.Second)
	defer cancel()
	stopped := make(chan error, 1)
	go func() { stopped <- pipe.recovery.Run(ctx) }()
	defer func() {
		cancel()
		require.ErrorIs(t, <-stopped, ctx.Err())
	}()
	runID, err := uuid.Parse(os.Getenv("PINN_RECOVERY_RUN"))
	require.NoError(t, err)
	cli, err := docker.New()
	require.NoError(t, err)
	defer cli.Close()
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	ready := false
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if ready {
				continue
			}
			a, err := journal.GetAttempt(ctx, runID)
			require.NoError(t, err)
			if !a.CanObserve() {
				continue
			}
			c, err := cli.Inspect(ctx, a.ContainerID)
			require.NoError(t, err)
			if c.IsRunning() {
				fmt.Println("solver-recovery-ready")
				ready = true
			}
		}
	}
}

// Publishing is deliberately excluded: the parent verifies the two durable outbox records.
type disabledProcessKafka struct{}

func (disabledProcessKafka) WriteMessages(context.Context, ...core.KafkaMessage) error {
	return errors.New("publishing is disabled in the process recovery test")
}

type recoveryProcess struct {
	command *exec.Cmd
	exited  chan struct{}
	err     error
	stderr  bytes.Buffer
}

func startRecoveryProcess(
	t *testing.T,
	ctx context.Context,
	cfg config.Config,
	runID uuid.UUID,
) *recoveryProcess {
	t.Helper()
	p := &recoveryProcess{
		command: exec.Command(os.Args[0], "-test.run=^TestSolverRecoveryProcessHelper$"),
		exited:  make(chan struct{}),
	}
	p.command.Env = append(os.Environ(),
		"PINN_RECOVERY_PROCESS_HELPER=1",
		"PINN_RECOVERY_JOURNAL="+cfg.SQLite.Path,
		"PINN_RECOVERY_MIGRATIONS="+cfg.SQLite.MigrationsDir,
		"PINN_RECOVERY_IMAGE="+cfg.PinnRunner.Image,
		"PINN_RECOVERY_RUN="+runID.String(),
	)
	p.command.Stderr = &p.stderr
	output, err := p.command.StdoutPipe()
	require.NoError(t, err)
	require.NoError(t, p.command.Start())
	t.Cleanup(func() { p.kill(t) })
	go func() {
		p.err = p.command.Wait()
		close(p.exited)
	}()
	ready := make(chan error, 1)
	go func() {
		scanner := bufio.NewScanner(output)
		var lines []string
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "solver-recovery-ready" {
				ready <- nil
				return
			}
			lines = append(lines, line)
			if len(lines) > 8 {
				lines = lines[1:]
			}
		}
		ready <- fmt.Errorf("helper output: %s; read error: %v", strings.Join(lines, "\n"), scanner.Err())
	}()
	select {
	case err := <-ready:
		require.NoError(t, err, "recovery helper must reach a running container")
	case <-p.exited:
		t.Fatalf("recovery helper exited before readiness: %v; %s; %v", p.err, p.stderr.String(), <-ready)
	case <-ctx.Done():
		p.kill(t)
		t.Fatalf("recovery helper not ready: %v; %s", ctx.Err(), p.stderr.String())
	}
	return p
}

func (p *recoveryProcess) kill(t *testing.T) {
	t.Helper()
	select {
	case <-p.exited:
		return
	default:
	}
	require.NoError(t, p.command.Process.Kill())
	select {
	case <-p.exited:
	case <-time.After(10 * time.Second):
		t.Fatal("recovery helper did not exit after Kill")
	}
}
