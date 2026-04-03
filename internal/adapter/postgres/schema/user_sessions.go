package schema

const UserSessionsTable = "user_sessions"

const (
	UserSessionsTableColumnID          = "id"
	UserSessionsTableColumnUserID      = "user_id"
	UserSessionsTableColumnTokenSha256 = "token_sha256"
	UserSessionsTableColumnCreatedAt   = "created_at"
	UserSessionsTableColumnExpiresAt   = "expires_at"
	UserSessionsTableColumnFingerprint = "fingerprint"
)

var UserSessionsTableColumns = []string{
	UserSessionsTableColumnID,
	UserSessionsTableColumnUserID,
	UserSessionsTableColumnTokenSha256,
	UserSessionsTableColumnCreatedAt,
	UserSessionsTableColumnExpiresAt,
	UserSessionsTableColumnFingerprint,
}
