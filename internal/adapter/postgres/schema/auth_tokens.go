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
