package docker

import (
	"errors"
	"fmt"
	"time"

	"github.com/moby/moby/api/types/container"
)

// NewContainerFromInspect restores a validated snapshot at the SDK boundary.
func NewContainerFromInspect(raw container.InspectResponse) (Container, error) {
	if raw.ID == "" || raw.Config == nil || raw.HostConfig == nil || raw.State == nil {
		return Container{}, errors.New("incomplete docker inspect response")
	}
	started, err := time.Parse(time.RFC3339Nano, raw.State.StartedAt)
	if err != nil {
		return Container{}, fmt.Errorf("parse container start time: %w", err)
	}
	finished, err := time.Parse(time.RFC3339Nano, raw.State.FinishedAt)
	if err != nil {
		return Container{}, fmt.Errorf("parse container finish time: %w", err)
	}
	return NewContainer(ContainerInput{
		ID:            raw.ID,
		Name:          raw.Name,
		Image:         raw.Config.Image,
		Labels:        raw.Config.Labels,
		AutoRemove:    raw.HostConfig.AutoRemove,
		RestartPolicy: string(raw.HostConfig.RestartPolicy.Name),
		State:         State(raw.State.Status),
		Running:       raw.State.Running,
		StartedAt:     started,
		FinishedAt:    finished,
		ExitCode:      raw.State.ExitCode,
		Error:         raw.State.Error,
	})
}
