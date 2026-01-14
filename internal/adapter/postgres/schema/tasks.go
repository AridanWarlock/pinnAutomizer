package schema

const TasksTable = "tasks"

const (
	TasksTableColumnID               = "id"
	TasksTableColumnName             = "name"
	TasksTableColumnDescription      = "description"
	TasksTableColumnStatus           = "status"
	TasksTableColumnConstants        = "constants"
	TasksTableColumnTrainingDataPath = "training_data_path"
	TasksTableColumnResultsPath      = "results_path"
	TasksTableColumnUserId           = "user_id"
	TasksTableColumnEquationId       = "equation_id"
	TasksTableColumnCreatedAt        = "created_at"
)

var TasksTableColumns = []string{
	TasksTableColumnID,
	TasksTableColumnName,
	TasksTableColumnDescription,
	TasksTableColumnStatus,
	TasksTableColumnConstants,
	TasksTableColumnTrainingDataPath,
	TasksTableColumnResultsPath,
	TasksTableColumnUserId,
	TasksTableColumnEquationId,
	TasksTableColumnCreatedAt,
}
