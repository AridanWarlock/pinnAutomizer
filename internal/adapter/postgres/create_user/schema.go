package create_user

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

const authTokensTable = "auth_tokens"

const (
	authTokensTableColumnUserID       = "user_id"
	authTokensTableColumnAccessToken  = "access_token"
	authTokensTableColumnRefreshToken = "refresh_token"
)

var authTokensColumns = []string{
	authTokensTableColumnUserID,
	authTokensTableColumnAccessToken,
	authTokensTableColumnRefreshToken,
}
