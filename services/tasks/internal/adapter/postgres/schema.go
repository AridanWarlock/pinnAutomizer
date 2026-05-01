package postgres

const EquationsTable = "equations"
const (
	EquationsID   = "id"
	EquationsType = "type"
)

var EquationsColumns = []string{
	EquationsID,
	EquationsType,
}

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
	TasksID               = "id"
	TasksName             = "name"
	TasksDescription      = "description"
	TasksStatus           = "status"
	TasksConstants        = "constants"
	TasksTrainingDataPath = "training_data_path"
	TasksResultsPath      = "results_path"
	TasksUserId           = "user_id"
	TasksEquationId       = "equation_id"
	TasksCreatedAt        = "created_at"
)

var TasksColumns = []string{
	TasksID,
	TasksName,
	TasksDescription,
	TasksStatus,
	TasksConstants,
	TasksTrainingDataPath,
	TasksResultsPath,
	TasksUserId,
	TasksEquationId,
	TasksCreatedAt,
}

var TasksSortColumns = map[string]struct{}{
	TasksName:      {},
	TasksStatus:    {},
	TasksCreatedAt: {},
}
