package users

const usersTable = "users"

const (
	usersTableColumnID           = "id"
	usersTableColumnLogin        = "login"
	usersTableColumnPasswordHash = "password_hash"
)

var usersTableColumns = []string{
	usersTableColumnID,
	usersTableColumnLogin,
	usersTableColumnPasswordHash,
}
