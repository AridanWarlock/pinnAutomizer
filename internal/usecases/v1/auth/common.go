package auth

import "time"

const (
	AccessTokenTTL  = 15 * time.Minute
	RefreshTokenTTL = 7 * 27 * time.Hour
)
