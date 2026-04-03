package refresh

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"pinnAutomizer/internal/domain"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type Postgres interface {
	GetUserSessionById(ctx context.Context, id uuid.UUID) (domain.UserSession, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (domain.User, error)
	GetRolesByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Role, error)
}

type AccessTokenGenerator interface {
	Generate(user domain.User, roles []domain.Role) (domain.AccessToken, error)
}

type Usecase struct {
	postgres             Postgres
	accessTokenGenerator AccessTokenGenerator

	log zerolog.Logger
}

var usecase *Usecase

func New(
	postgres Postgres,
	accessTokenGenerator AccessTokenGenerator,
	log zerolog.Logger,
) *Usecase {
	uc := &Usecase{
		postgres:             postgres,
		accessTokenGenerator: accessTokenGenerator,

		log: log.With().Str("component", "usecase: auth.Refresh").Logger(),
	}

	usecase = uc

	return uc
}

func (u *Usecase) Refresh(ctx context.Context, in Input) (Output, error) {
	log := u.log.With().Ctx(ctx).Logger()

	if err := in.Validate(); err != nil {
		log.Info().Err(err).Msg("input validation error")
		return Output{}, err
	}

	sessionID, tokenSha256, err := parseRefreshTokenFromString(in.RefreshTokenString)
	if err != nil {
		log.Info().Err(err).Msg("refresh token validating error")
		return Output{}, err
	}

	session, err := u.postgres.GetUserSessionById(ctx, sessionID)
	if err != nil {
		log.Error().Err(err).Msg("postgres: getting session error")
		return Output{}, err
	}

	if subtle.ConstantTimeCompare(session.TokenSha256, tokenSha256) != 1 {
		log.Error().Err(err).Msg("session is no longer valid")
		return Output{}, domain.ErrSessionCompromised
	}

	if session.ExpiresAt.Before(time.Now()) {
		log.Info().Err(domain.ErrRefreshTokenExpired).Msg("refresh token expired")
		return Output{}, domain.ErrRefreshTokenExpired
	}

	user, err := u.postgres.GetUserByID(ctx, session.UserID)
	if err != nil {
		log.Error().Err(err).Msg("postgres: getting user by id error")
		return Output{}, err
	}

	roles, err := u.postgres.GetRolesByUserID(ctx, user.ID)
	if err != nil {
		log.Error().Err(err).Msg("postgres: getting user roles error")
		return Output{}, err
	}

	accessToken, err := u.accessTokenGenerator.Generate(user, roles)
	if err != nil {
		log.Info().Err(err).Msg("accessTokenGenerator: generate access token error")
		return Output{}, err
	}

	return Output{AccessToken: accessToken}, nil
}

func parseRefreshTokenFromString(token string) (uuid.UUID, []byte, error) {
	tokenSplit := strings.Split(token, ".")
	if len(tokenSplit) != 2 {
		return uuid.UUID{}, nil, domain.ErrParseRefreshTokenFailed
	}

	sessionIDString := tokenSplit[0]
	sessionID, err := uuid.Parse(sessionIDString)
	if err != nil {
		return uuid.UUID{}, nil, domain.ErrParseRefreshTokenFailed
	}

	randomBytesString := tokenSplit[1]
	sha256Bytes := sha256.Sum256([]byte(randomBytesString))

	return sessionID, sha256Bytes[:], nil
}
