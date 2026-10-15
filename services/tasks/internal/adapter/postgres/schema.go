package postgres

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
