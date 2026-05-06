package mlrunner

import (
	"context"
	"fmt"
	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
	"github.com/rs/zerolog"
	"io"
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

	log zerolog.Logger
}

func NewPinnRunner(cfg Config, log zerolog.Logger) (*PinnRunner, error) {
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
	if !r.mx.TryLock() {
		return 0, domain.ErrPinnBusy
	}
	defer r.mx.Unlock()

	createOpts := r.setupContainer(task, command)
	resp, err := r.cli.ContainerCreate(ctx, createOpts)
	if err != nil {
		return 0, fmt.Errorf("create container: %w", err)
	}

	go func() {
		time.Sleep(1 * time.Second) // Даем контейнеру время запуститься
		logs, _ := r.cli.ContainerLogs(ctx, resp.ID, client.ContainerLogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Follow:     true,
		})
		if logs != nil {
			defer logs.Close()
			io.Copy(r.log, logs)
		}
	}()

	_, err = r.cli.ContainerStart(ctx, resp.ID, client.ContainerStartOptions{})
	if err != nil {
		return 0, fmt.Errorf("failed to start container: %w", err)
	}

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
			return code, fmt.Errorf("run container: %v", res.Error)
		}

		return 0, nil
	}
}

func (r *PinnRunner) setupContainer(task domain.MlTask, command []string) client.ContainerCreateOptions {
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

	containerConfig := &container.Config{
		Image: r.pinnImage,
		Cmd:   command,
		Tty:   false,

		Env: env,
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
		{
			Type:   mount.TypeTmpfs,
			Target: "/tmp",
			TmpfsOptions: &mount.TmpfsOptions{
				SizeBytes: 1024 * 1024 * 1024, // 1 GB
				Mode:      0777,
			},
		},
		{
			Type:   mount.TypeTmpfs,
			Target: "/var/tmp",
			TmpfsOptions: &mount.TmpfsOptions{
				SizeBytes: 512 * 1024 * 1024, // 512 MB
				Mode:      0777,
			},
		},
		{
			Type:   mount.TypeTmpfs,
			Target: "/usr/tmp",
			TmpfsOptions: &mount.TmpfsOptions{
				SizeBytes: 512 * 1024 * 1024,
				Mode:      0777,
			},
		},
		{
			Type:   mount.TypeTmpfs,
			Target: "/root/.cache",
			TmpfsOptions: &mount.TmpfsOptions{
				SizeBytes: 1024 * 1024 * 1024, // 1 GB
				Mode:      0777,
			},
		},
		{
			Type:   mount.TypeTmpfs,
			Target: "/root/.config",
			TmpfsOptions: &mount.TmpfsOptions{
				SizeBytes: 100 * 1024 * 1024, // 100 MB
				Mode:      0777,
			},
		},
		{
			Type:   mount.TypeTmpfs,
			Target: "/root/.torch",
			TmpfsOptions: &mount.TmpfsOptions{
				SizeBytes: 512 * 1024 * 1024,
				Mode:      0777,
			},
		},
		{
			Type:   mount.TypeTmpfs,
			Target: "/dev/shm",
			TmpfsOptions: &mount.TmpfsOptions{
				SizeBytes: 1024 * 1024 * 1024, // 1 GB для shared memory
				Mode:      0777,
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
		AutoRemove:     true,
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

	return createOptions
}

func (r *PinnRunner) Close() error {
	return r.cli.Close()
}
