-- +goose Up
-- +goose StatementBegin
CREATE TABLE user_sessions
(
    id           uuid primary key,
    user_id      uuid        not null,
    token_sha256 bytea       not null,
    
    created_at   timestamptz not null,
    expires_at   timestamptz not null,

    fingerprint  bytea       not null,

    CONSTRAINT fk_user_sessions_users FOREIGN KEY (user_id) REFERENCES users
);

COMMENT ON TABLE user_sessions IS 'Таблица сессий пользователей';
COMMENT ON COLUMN user_sessions.id IS 'ID сессии';
COMMENT ON COLUMN user_sessions.user_id IS 'ID пользователя';
COMMENT ON COLUMN user_sessions.token_sha256 IS 'SHA256 от refresh токена';
COMMENT ON COLUMN user_sessions.created_at IS 'Timestamp создания сессии';
COMMENT ON COLUMN user_sessions.created_at IS 'Timestamp истечения действия сессии';
COMMENT ON COLUMN user_sessions.fingerprint IS 'Fingerprint устройства, с которого создана сессия';
-- +goose StatementEnd