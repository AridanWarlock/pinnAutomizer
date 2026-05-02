package mlrunner

import (
	"bufio"
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/client"
)

type PinnRunner struct {
	cli *client.Client

	pinnImage          string
	hostTasksDataDir   string
	hostTasksOutputDir string

	timeout time.Duration

	mx sync.Mutex
}

func NewPinnRunner(cfg Config) (*PinnRunner, error) {
	cli, err := client.New(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("create docker client: %w", err)
	}

	return &PinnRunner{
		cli: cli,

		pinnImage:          cfg.Image,
		hostTasksDataDir:   cfg.HostTasksDataDir,
		hostTasksOutputDir: cfg.HostTasksOutputDir,

		timeout: cfg.Timeout,

		mx: sync.Mutex{},
	}, nil
}

func (r *PinnRunner) Run(ctx context.Context, task domain.MlTask) (int, error) {
	args := fmt.Sprintf("mod=%s", task.Mode)

	switch task.Mode {
	case domain.MlTaskModeTrain, domain.MlTaskModeRetrain:
	case domain.MlTaskModePredict:
		args += fmt.Sprintf("checkpoint=%s", task.CheckpointFile)
	default:
		return 0, domain.ErrInvalidMLTaskMode
	}

	command := []string{
		"python",
		"run.py",
		args,
	}

	return r.run(ctx, task, command)
}

func (r *PinnRunner) run(ctx context.Context, task domain.MlTask, command []string) (int, error) {
	if !r.mx.TryLock() {
		return 0, domain.ErrPinnBusy
	}
	defer r.mx.Unlock()

	containerConfig := &container.Config{
		Image: r.pinnImage,
		Cmd:   command,
		Tty:   false,
	}

	hostDataPath := filepath.Join(r.hostTasksDataDir, task.TaskID.String())
	hostOutputPath := filepath.Join(r.hostTasksOutputDir, task.TaskID.String())

	mounts := []mount.Mount{
		{
			Type:     mount.TypeBind,
			Source:   hostDataPath,
			Target:   "/task_data",
			ReadOnly: true,
			BindOptions: &mount.BindOptions{
				Propagation: mount.PropagationRPrivate,
			},
		},
		{
			Type:     mount.TypeBind,
			Source:   hostOutputPath,
			Target:   "/task_output",
			ReadOnly: false,
			BindOptions: &mount.BindOptions{
				Propagation: mount.PropagationRPrivate,
			},
		},
	}

	deviceRequest := container.DeviceRequest{
		Driver: "nvidia",
		Count:  1,
		Capabilities: [][]string{
			{"gpu"},
			{"compute", "utility"},
		},
	}

	hostConfig := &container.HostConfig{
		NetworkMode: "none",
		Mounts:      mounts,
		Resources: container.Resources{
			DeviceRequests: []container.DeviceRequest{deviceRequest},
		},
		AutoRemove:     false,
		ReadonlyRootfs: true,
		SecurityOpt: []string{
			"no-new-privileges:true",
		},
		CapDrop: []string{"ALL"},
	}

	createOptions := client.ContainerCreateOptions{
		Config:     containerConfig,
		HostConfig: hostConfig,
		Name:       fmt.Sprintf("pinn-solver-%s-%d", task.TaskID, time.Now().Unix()),
	}

	resp, err := r.cli.ContainerCreate(ctx, createOptions)
	if err != nil {
		return 0, fmt.Errorf("failed to create container: %w", err)
	}

	logOptions := client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true, // Следить за логами в реальном времени
		Since:      "",   // Все логи с начала
		Timestamps: true, // Добавить временные метки
	}

	// Создаём каналы для передачи вывода
	stdoutCh := make(chan string, 100)
	stderrCh := make(chan string, 100)
	errCh := make(chan error, 2)

	// Запускаем горутину для чтения логов
	logsReader, err := r.cli.ContainerLogs(ctx, resp.ID, logOptions)
	if err != nil {
		return 0, fmt.Errorf("failed to get container logs: %w", err)
	}
	defer logsReader.Close()

	// Читаем логи в фоне
	go func() {
		scanner := bufio.NewScanner(logsReader)
		for scanner.Scan() {
			line := scanner.Text()
			// Docker logs возвращает строки с префиксами (01 для stdout, 02 для stderr)
			// Можно разобрать, но проще использовать отдельный метод
			fmt.Printf("[CONTAINER LOG] %s\n", line)
			stdoutCh <- line
		}
		if err := scanner.Err(); err != nil {
			errCh <- err
		}
		close(stdoutCh)
		close(stderrCh)
	}()

	_, err = r.cli.ContainerStart(ctx, resp.ID, client.ContainerStartOptions{})
	if err != nil {
		return 0, fmt.Errorf("failed to start container: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	wait := r.cli.ContainerWait(
		ctx,
		resp.ID,
		client.ContainerWaitOptions{Condition: container.WaitConditionNotRunning},
	)

	select {
	case err := <-wait.Error:
		return -1, fmt.Errorf("run container: %w", err)
	case res := <-wait.Result:
		code := int(res.StatusCode)
		if code != 0 {
			return code, fmt.Errorf("run container: %s", res.Error)
		}

		return 0, nil
	}
}

func (r *PinnRunner) Close() error {
	return r.cli.Close()
}
