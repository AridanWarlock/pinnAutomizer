package schema

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
