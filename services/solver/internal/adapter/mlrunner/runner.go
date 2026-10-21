package mlrunner

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/AridanWarlock/pinnAutomizer/pkg/docker"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/AridanWarlock/pinnAutomizer/solver/internal/domain"
)

type Docker interface {
	Create(ctx context.Context, cfg docker.Config) (string, error)
	Inspect(ctx context.Context, id string) (docker.Container, error)
	Start(ctx context.Context, id string) error
	Stop(ctx context.Context, id string, grace time.Duration) error
	RemoveStopped(ctx context.Context, id string) error
	Wait(ctx context.Context, id string) (docker.Exit, error)
	Logs(ctx context.Context, id string) (io.ReadCloser, error)
	Close() error
}

type PinnRunner struct {
	docker Docker

	containerNamePrefix string
	pinnImage           string
	hostTasksDataDir    string
	hostTasksOutputDir  string

	timeout time.Duration

	log zerolog.Logger
}

func NewPinnRunner(cfg Config, log zerolog.Logger) (*PinnRunner, error) {
	cli, err := docker.New()
	if err != nil {
		return nil, fmt.Errorf("create docker client: %w", err)
	}

	return &PinnRunner{
		docker: cli,

		containerNamePrefix: cfg.ContainerNamePrefix,
		pinnImage:           cfg.Image,
		hostTasksDataDir:    cfg.HostTasksDataDir,
		hostTasksOutputDir:  cfg.HostTasksOutputDir,

		timeout: cfg.Timeout,

		log: log,
	}, nil
}

func (r *PinnRunner) Run(ctx context.Context, task domain.MlTask) (int, error) {
	if err := task.Validate(); err != nil {
		return 0, fmt.Errorf("%w: validate task: %w", errs.ErrInvalidArgument, err)
	}

	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	args := fmt.Sprintf("--mod=%s", task.Mode)

	command := []string{
		"python",
		"run.py",
		args,
	}

	return r.run(ctx, task, command)
}

func (r *PinnRunner) run(ctx context.Context, task domain.MlTask, command []string) (int, error) {
	createOpts := r.setupContainer(task.TaskID, task.Mode, command)
	id, err := r.docker.Create(ctx, createOpts)
	if err != nil {
		return 0, fmt.Errorf("create container: %w", err)
	}

	go func() {
		time.Sleep(1 * time.Second) // Даем контейнеру время запуститься
		logs, _ := r.docker.Logs(ctx, id)
		if logs != nil {
			defer logs.Close()
			io.Copy(r.log, logs)
		}
	}()

	err = r.docker.Start(ctx, id)
	if err != nil {
		return 0, fmt.Errorf("failed to start container: %w", err)
	}

	res, err := r.docker.Wait(ctx, id)
	if err != nil {
		return -1, fmt.Errorf("run container: %w", err)
	}
	if res.Code != 0 {
		return res.Code, fmt.Errorf("run container: %s", res.Error)
	}
	return 0, nil
}

func (r *PinnRunner) setupContainer(taskID uuid.UUID, mode string, command []string) docker.Config {
	env := []string{
		// Python
		"PYTHONUNBUFFERED=1",
		"PYTHONHASHSEED=42",
		// Временные директории
		"TMPDIR=/tmp",
		"TEMP=/tmp",
		"TMP=/tmp",
		// Matplotlib
		"MPLCONFIGDIR=/tmp/matplotlib",
		"XDG_CACHE_HOME=/tmp/cache",
		// Torch/Lightning
		"TORCH_HOME=/tmp/torch",
		"TORCH_EXTENSIONS_DIR=/tmp/torch_extensions",
		// Lightning
		"LIGHTNING_CACHE_DIR=/tmp/lightning_cache",
		// Hugging Face
		"HF_HOME=/tmp/huggingface",
		"TRANSFORMERS_CACHE=/tmp/transformers",
		"HUGGINGFACE_HUB_CACHE=/tmp/huggingface",
		// TorchMetrics
		"TORCHMETRICS_CACHE_DIR=/tmp/torchmetrics",
		// Conda/Pip
		"PIP_CACHE_DIR=/tmp/pip_cache",
		"CONDA_PKGS_DIRS=/tmp/conda_pkgs",
	}

	return docker.Config{
		Name:    fmt.Sprintf("pinn-solver-%s-%d", taskID, time.Now().Unix()),
		Image:   r.pinnImage,
		Command: command,
		Env:     env,
		Mounts: []docker.Mount{
			{
				Source:   filepath.Join(r.hostTasksDataDir, taskID.String()),
				Target:   "/task_data",
				ReadOnly: true,
			},
			{
				Source: filepath.Join(r.hostTasksOutputDir, taskID.String()),
				Target: "/task_output",
			},
			{
				Target:     "/tmp",
				TmpfsBytes: 1024 * 1024 * 1024,
				TmpfsMode:  0777,
			},
			{
				Target:     "/var/tmp",
				TmpfsBytes: 512 * 1024 * 1024,
				TmpfsMode:  0777,
			},
			{
				Target:     "/usr/tmp",
				TmpfsBytes: 512 * 1024 * 1024,
				TmpfsMode:  0777,
			},
			{
				Target:     "/root/.cache",
				TmpfsBytes: 1024 * 1024 * 1024,
				TmpfsMode:  0777,
			},
			{
				Target:     "/root/.config",
				TmpfsBytes: 100 * 1024 * 1024,
				TmpfsMode:  0777,
			},
			{
				Target:     "/root/.torch",
				TmpfsBytes: 512 * 1024 * 1024,
				TmpfsMode:  0777,
			},
			{
				Target:     "/dev/shm",
				TmpfsBytes: 1024 * 1024 * 1024,
				TmpfsMode:  0777,
			},
		},
		GPU:            mode == domain.MlTaskModeTrain || mode == domain.MlTaskModeRetrain,
		NetworkMode:    "none",
		AutoRemove:     true,
		ReadonlyRootfs: true,
		SecurityOpt:    []string{"no-new-privileges:true"},
		CapDrop:        []string{"ALL"},
	}
}

func (r *PinnRunner) Close() error {
	return r.docker.Close()
}
