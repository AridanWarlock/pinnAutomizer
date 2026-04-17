-- +goose Up
CREATE TABLE refresh_tokens(
    hash text primary key,

    user_id uuid not null references users,
    jti uuid not null,

    fingerprint text not null,
    agent text not null,
    ip inet not null,

    expires_at timestamptz not null,
    created_at timestamptz not null default now(),

    UNIQUE (user_id, fingerprint)
);

COMMENT ON TABLE refresh_tokens IS 'Таблица токенов обновления';
COMMENT ON COLUMN refresh_tokens.hash IS 'Хеш от сгенерированной refresh-строки';
COMMENT ON COLUMN refresh_tokens.user_id IS 'ID пользователя';
COMMENT ON COLUMN refresh_tokens.jti IS 'ID сессии';
COMMENT ON COLUMN refresh_tokens.fingerprint IS 'Fingerprint сессии';
COMMENT ON COLUMN refresh_tokens.agent IS 'Устройство пользователя';
COMMENT ON COLUMN refresh_tokens.ip IS 'IP пользователя';
COMMENT ON COLUMN refresh_tokens.expires_at IS 'Время истечения токена обновления';
COMMENT ON COLUMN refresh_tokens.created_at IS 'Время создания токена обновления';
