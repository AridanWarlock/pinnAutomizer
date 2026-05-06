package tasks

import (
	"github.com/AridanWarlock/pinnAutomizer/tasks/internal/domain"
	"github.com/google/uuid"
	"time"
)

type TaskDto struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`

	Mode domain.TaskMode `json:"mode"`

	Status domain.TaskStatus `json:"status"`
	Error  *string           `json:"error,omitempty"`

	HavePlot bool `json:"have_plot"`

	CreatedAt time.Time `json:"created_at"`
} // @name TaskDTO

func ToDto(task domain.Task) TaskDto {
	return TaskDto{
		ID:          task.ID,
		Name:        task.Name,
		Description: task.Description,

		Mode: task.Mode,

		Status: task.Status,
		Error:  task.Error,

		HavePlot: task.PlotPath != nil,

		CreatedAt: task.CreatedAt,
	}
}
