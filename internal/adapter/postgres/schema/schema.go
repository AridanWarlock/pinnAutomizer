package schema

const AuthTokensTable = "auth_tokens"

const (
	AuthTokensTableColumnUserID       = "user_id"
	AuthTokensTableColumnAccessToken  = "access_token"
	AuthTokensTableColumnRefreshToken = "refresh_token"
)

var AuthTokensColumns = []string{
	AuthTokensTableColumnUserID,
	AuthTokensTableColumnAccessToken,
	AuthTokensTableColumnRefreshToken,
}

const EquationsTable = "equations"

const (
	EquationsTableColumnID   = "id"
	EquationsTableColumnType = "type"
)

var EquationsTableColumns = []string{
	EquationsTableColumnID,
	EquationsTableColumnType,
}

const ScriptsTable = "scripts"

const (
	ScriptsTableColumnID         = "id"
	ScriptsTableColumnFilename   = "filename"
	ScriptsTableColumnPath       = "path"
	ScriptsTableColumnUploadTime = "upload_time"
	ScriptsTableColumnText       = "text"
	ScriptsTableColumnUserID     = "user_id"
)

var ScriptsTableColumns = []string{
	ScriptsTableColumnID,
	ScriptsTableColumnFilename,
	ScriptsTableColumnPath,
	ScriptsTableColumnUploadTime,
	ScriptsTableColumnText,
	ScriptsTableColumnUserID,
}

const TasksTable = "tasks"

const (
	TasksTableColumnID                       = "id"
	TasksTableColumnName                     = "name"
	TasksTableColumnDescription              = "description"
	TasksTableColumnStatus                   = "status"
	TasksTableColumnConstants                = "constants"
	TasksTableColumnTrainingDataPath         = "training_data_path"
	TasksTableColumnTrainingAnalyticSolution = "training_analytic_solution"
	TasksTableColumnResultsPath              = "results_path"
	TasksTableColumnUserId                   = "user_id"
	TasksTableColumnEquationId               = "equation_id"
	TasksTableColumnCreatedAt                = "created_at"
)

var TasksTableColumns = []string{
	TasksTableColumnID,
	TasksTableColumnName,
	TasksTableColumnDescription,
	TasksTableColumnStatus,
	TasksTableColumnConstants,
	TasksTableColumnTrainingDataPath,
	TasksTableColumnTrainingAnalyticSolution,
	TasksTableColumnResultsPath,
	TasksTableColumnUserId,
	TasksTableColumnEquationId,
	TasksTableColumnCreatedAt,
}

const UsersTable = "users"

const (
	UsersTableColumnID           = "id"
	UsersTableColumnLogin        = "login"
	UsersTableColumnPasswordHash = "password_hash"
)

var UsersTableColumns = []string{
	UsersTableColumnID,
	UsersTableColumnLogin,
	UsersTableColumnPasswordHash,
}
