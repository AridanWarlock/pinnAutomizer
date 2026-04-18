package postgres

const RefreshTokensTable = "refresh_tokens"
const (
	RefreshTokensHash        = "hash"
	RefreshTokensUserID      = "user_id"
	RefreshTokensJti         = "jti"
	RefreshTokensFingerprint = "fingerprint"
	RefreshTokensAgent       = "agent"
	RefreshTokensIP          = "ip"
	RefreshTokensCreatedAt   = "created_at"
	RefreshTokensExpiresAt   = "expires_at"
)

var RefreshTokensColumns = []string{
	RefreshTokensHash,
	RefreshTokensUserID,
	RefreshTokensJti,
	RefreshTokensFingerprint,
	RefreshTokensAgent,
	RefreshTokensIP,
	RefreshTokensCreatedAt,
	RefreshTokensExpiresAt,
}

const RolesTable = "roles"
const (
	RolesID    = "id"
	RolesTitle = "title"
)

var RolesColumns = []string{
	RolesID,
	RolesTitle,
}

const UsersTable = "users"
const (
	UsersID           = "id"
	UsersLogin        = "login"
	UsersPasswordHash = "password_hash"
)

var UsersColumns = []string{
	UsersID,
	UsersLogin,
	UsersPasswordHash,
}

const UsersRolesTable = "users_roles"
const (
	UsersRolesUserID = "user_id"
	UsersRolesRoleID = "role_id"
)

var UsersRolesColumns = []string{
	UsersRolesUserID,
	UsersRolesRoleID,
}
