package docker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/containerd/errdefs"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/client"

	"github.com/AridanWarlock/pinnAutomizer/pkg/errs"
)

type Client struct {
	cli *client.Client
}

func New() (*Client, error) {
	cli, err := client.New(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("create docker client: %w", err)
	}
	return &Client{cli: cli}, nil
}

func (c *Client) Create(ctx context.Context, cfg Config) (string, error) {
	opts := createOptions(cfg)
	resp, err := c.cli.ContainerCreate(ctx, opts)
	return resp.ID, dockerError(err)
}

func (c *Client) Inspect(ctx context.Context, id string) (Container, error) {
	resp, err := c.cli.ContainerInspect(ctx, id, client.ContainerInspectOptions{})
	if err != nil {
		return Container{}, dockerError(err)
	}
	raw := resp.Container
	if raw.ID == "" || raw.Config == nil || raw.HostConfig == nil || raw.State == nil {
		return Container{}, errors.New("incomplete docker inspect response")
	}
	started, err := time.Parse(time.RFC3339Nano, raw.State.StartedAt)
	if err != nil {
		return Container{}, fmt.Errorf("parse container start time: %w", err)
	}
	return Container{
		ID:            raw.ID,
		Name:          raw.Name,
		Image:         raw.Config.Image,
		Labels:        raw.Config.Labels,
		AutoRemove:    raw.HostConfig.AutoRemove,
		RestartPolicy: string(raw.HostConfig.RestartPolicy.Name),
		State:         string(raw.State.Status),
		Running:       raw.State.Running,
		StartedAt:     started,
	}, nil
}

func (c *Client) Start(ctx context.Context, id string) error {
	_, err := c.cli.ContainerStart(ctx, id, client.ContainerStartOptions{})
	return dockerError(err)
}

func (c *Client) Wait(ctx context.Context, id string) (Exit, error) {
	wait := c.cli.ContainerWait(ctx, id, client.ContainerWaitOptions{Condition: container.WaitConditionNotRunning})
	select {
	case err := <-wait.Error:
		if err == nil {
			return Exit{}, errors.New("docker wait returned no error or result")
		}
		return Exit{}, dockerError(err)
	case res := <-wait.Result:
		out := Exit{Code: int(res.StatusCode)}
		if res.Error != nil {
			out.Error = res.Error.Message
		}
		return out, nil
	}
}

func (c *Client) Logs(ctx context.Context, id string) (io.ReadCloser, error) {
	logs, err := c.cli.ContainerLogs(ctx, id, client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	})
	return logs, dockerError(err)
}

func (c *Client) Remove(ctx context.Context, id string) error {
	_, err := c.cli.ContainerRemove(ctx, id, client.ContainerRemoveOptions{Force: true})
	return dockerError(err)
}

func (c *Client) Close() error {
	return c.cli.Close()
}

func dockerError(err error) error {
	switch {
	case errdefs.IsNotFound(err):
		return fmt.Errorf("%w: %w", errs.ErrNotFound, err)
	case errdefs.IsConflict(err):
		return fmt.Errorf("%w: %w", errs.ErrConflict, err)
	case errs.IsContextErr(err):
		return err
	default:
		return fmt.Errorf("unexpected docker error: %w", err)
	}
}

func createOptions(cfg Config) client.ContainerCreateOptions {
	mounts := make([]mount.Mount, len(cfg.Mounts))
	for i, m := range cfg.Mounts {
		mounts[i] = mount.Mount{
			Type:        mount.TypeBind,
			Source:      m.Source,
			Target:      m.Target,
			ReadOnly:    m.ReadOnly,
			BindOptions: &mount.BindOptions{Propagation: mount.PropagationRPrivate},
		}
		if m.TmpfsBytes > 0 {
			mounts[i].Type = mount.TypeTmpfs
			mounts[i].BindOptions = nil
			mounts[i].TmpfsOptions = &mount.TmpfsOptions{SizeBytes: m.TmpfsBytes, Mode: os.FileMode(m.TmpfsMode)}
		}
	}
	var devices []container.DeviceRequest
	if cfg.GPU {
		devices = []container.DeviceRequest{{
			Driver:       "nvidia",
			Count:        1,
			Capabilities: [][]string{{"gpu"}, {"compute", "utility"}},
		}}
	}
	return client.ContainerCreateOptions{
		Name: cfg.Name,
		Config: &container.Config{
			Image:  cfg.Image,
			Cmd:    cfg.Command,
			Env:    cfg.Env,
			Labels: cfg.Labels,
		},
		HostConfig: &container.HostConfig{
			Mounts:         mounts,
			Resources:      container.Resources{DeviceRequests: devices},
			AutoRemove:     cfg.AutoRemove,
			ReadonlyRootfs: cfg.ReadonlyRootfs,
			NetworkMode:    container.NetworkMode(cfg.NetworkMode),
			SecurityOpt:    cfg.SecurityOpt,
			CapDrop:        cfg.CapDrop,
		},
	}
}
