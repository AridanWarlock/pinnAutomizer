-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS auth_tokens
(
    user_id
    uuid
    not
    null,
    access_token
    text,
    refresh_token
    text,

    PRIMARY
    KEY
(
    user_id
)
    );

COMMENT
ON TABLE auth_tokens is 'Токены авторизации';
COMMENT
ON COLUMN auth_tokens.user_id is 'ID пользователя';
COMMENT
ON COLUMN auth_tokens.access_token is 'Access token пользователя';
COMMENT
ON COLUMN auth_tokens.refresh_token is 'Refresh token пользователя';
-- +goose StatementEnd