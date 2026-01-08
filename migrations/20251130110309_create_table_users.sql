-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS users
(
    id            uuid not null,
    login         text not null,
    password_hash text not null,

    PRIMARY KEY (id),
    UNIQUE (login)
);

COMMENT ON TABLE users is 'Таблица пользователей';

COMMENT ON COLUMN users.id is 'Уникальный идентификатор пользователя';
COMMENT ON COLUMN users.login is 'Логин пользователя';
COMMENT ON COLUMN users.password_hash is 'Хеш пароля пользователя';
-- +goose StatementEnd