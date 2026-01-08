package tasks

const tasksTable = "tasks"

var (
	tasksTableColumnID                       = "id"
	tasksTableColumnName                     = "name"
	tasksTableColumnDescription              = "description"
	tasksTableColumnStatus                   = "status"
	tasksTableColumnConstants                = "constants"
	tasksTableColumnTrainingDataPath         = "training_data_path"
	tasksTableColumnTrainingAnalyticSolution = "training_analytic_solution"
	tasksTableColumnResultsPath              = "results_path"
	tasksTableColumnUserId                   = "user_id"
	tasksTableColumnEquationId               = "equation_id"
	tasksTableColumnCreatedAt                = "created_at"
)

var tasksTableColumns = []string{
	tasksTableColumnID,
	tasksTableColumnName,
	tasksTableColumnDescription,
	tasksTableColumnStatus,
	tasksTableColumnConstants,
	tasksTableColumnTrainingDataPath,
	tasksTableColumnTrainingAnalyticSolution,
	tasksTableColumnResultsPath,
	tasksTableColumnUserId,
	tasksTableColumnEquationId,
	tasksTableColumnCreatedAt,
}

const equationsTable = "equations"

var (
	equationsTableColumnID   = "id"
	equationsTableColumnType = "type"
)

var equationsTableColumns = []string{
	equationsTableColumnID,
	equationsTableColumnType,
}
