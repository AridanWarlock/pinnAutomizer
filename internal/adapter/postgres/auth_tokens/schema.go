package auth_tokens

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
