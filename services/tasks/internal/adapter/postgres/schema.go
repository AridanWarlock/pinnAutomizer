package postgres

import "github.com/google/uuid"

const EventsTable = "events"
const (
	EventsIdKey     = "id_key"
	EventsTopic     = "topic"
	EventsData      = "data"
	EventsCreatedAt = "created_at"
)

var EventsColumns = []string{
	EventsIdKey,
	EventsTopic,
	EventsData,
	EventsCreatedAt,
}

const TasksTable = "tasks"
const (
	TasksID          = "id"
	TasksName        = "name"
	TasksDescription = "description"
	TasksMode        = "mode"
	TasksStatus      = "status"
	TasksError       = "error"
	TasksDataPath    = "data_path"
	TasksOutputPath  = "output_path"
	TasksPlotPath    = "plot_path"
	TasksUserId      = "user_id"
	TasksCreatedAt   = "created_at"
)

var TasksColumns = []string{
	TasksID,
	TasksName,
	TasksDescription,
	TasksMode,
	TasksStatus,
	TasksError,
	TasksDataPath,
	TasksOutputPath,
	TasksPlotPath,
	TasksUserId,
	TasksCreatedAt,
}

var TasksSortColumns = map[string]struct{}{
	TasksName:      {},
	TasksStatus:    {},
	TasksCreatedAt: {},
}

const TaskRunRequestsTable = "task_run_requests"
const (
	TaskRunRequestsUserID    = "user_id"
	TaskRunRequestsIdKey     = "id_key"
	TaskRunRequestsTaskID    = "task_id"
	TaskRunRequestsEventID   = "event_id"
	TaskRunRequestsCreatedAt = "created_at"
)

var TaskRunRequestsColumns = []string{
	TaskRunRequestsUserID,
	TaskRunRequestsIdKey,
	TaskRunRequestsTaskID,
	TaskRunRequestsEventID,
	TaskRunRequestsCreatedAt,
}

const FileCleanupTable = "file_cleanup"

const (
	FileCleanupTaskID     = "task_id"
	FileCleanupDataPath   = "data_path"
	FileCleanupOutputPath = "output_path"
)

var fileCleanupColumns = []string{
	FileCleanupTaskID,
	FileCleanupDataPath,
	FileCleanupOutputPath,
}

type fileCleanupRow struct {
	TaskID     uuid.UUID `db:"task_id"`
	DataPath   string    `db:"data_path"`
	OutputPath string    `db:"output_path"`
}

func (r fileCleanupRow) Values() []any {
	return []any{
		r.TaskID,
		r.DataPath,
		r.OutputPath,
	}
}
