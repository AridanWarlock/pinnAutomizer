-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS roles
(
    id    uuid not null,
    title text not null unique,

    PRIMARY KEY (id)
);

COMMENT ON TABLE roles is 'Таблица ролей';
COMMENT ON COLUMN roles.id is 'ID роли';
COMMENT ON COLUMN roles.title is 'Название роли';
-- +goose StatementEnd