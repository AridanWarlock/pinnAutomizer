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

const EventsTable = "events"

const (
	EventsTableColumnID        = "id"
	EventsTableColumnTopic     = "topic"
	EventsTableColumnData      = "data"
	EventsTableColumnCreatedAt = "created_at"
)

var EventsTableColumns = []string{
	EventsTableColumnID,
	EventsTableColumnTopic,
	EventsTableColumnData,
	EventsTableColumnCreatedAt,
}

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
