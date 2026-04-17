package schema

const RefreshTokensTable = "refresh_tokens"

const (
	RefreshTokensTableColumnHash        = "hash"
	RefreshTokensTableColumnUserID      = "user_id"
	RefreshTokensTableColumnJti         = "jti"
	RefreshTokensTableColumnFingerprint = "fingerprint"
	RefreshTokensTableColumnAgent       = "agent"
	RefreshTokensTableColumnIP          = "ip"
	RefreshTokensTableColumnCreatedAt   = "created_at"
	RefreshTokensTableColumnExpiresAt   = "expires_at"
)

var RefreshTokensTableColumns = []string{
	RefreshTokensTableColumnHash,
	RefreshTokensTableColumnUserID,
	RefreshTokensTableColumnJti,
	RefreshTokensTableColumnFingerprint,
	RefreshTokensTableColumnAgent,
	RefreshTokensTableColumnIP,
	RefreshTokensTableColumnCreatedAt,
	RefreshTokensTableColumnExpiresAt,
}
