-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS users_roles
(
    user_id uuid not null,
    role_id uuid not null
);

COMMENT ON TABLE users_roles is 'Таблица связки Many-to-many Пользователь-Роль';
COMMENT ON COLUMN users_roles.user_id is 'ID пользователя';
COMMENT ON COLUMN users_roles.role_id is 'ID роли';
-- +goose StatementEnd
