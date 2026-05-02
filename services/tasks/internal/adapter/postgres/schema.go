package postgres

const EventsTable = "events"
const (
	EventsID        = "id"
	EventsTopic     = "topic"
	EventsData      = "data"
	EventsCreatedAt = "created_at"
)

var EventsColumns = []string{
	EventsID,
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
	TasksUserId,
	TasksCreatedAt,
}

var TasksSortColumns = map[string]struct{}{
	TasksName:      {},
	TasksStatus:    {},
	TasksCreatedAt: {},
}
