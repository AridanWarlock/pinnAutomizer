-- +goose Up
ALTER TABLE tasks DROP CONSTRAINT fk_tasks_users;

DROP TABLE refresh_tokens;
DROP TABLE users_roles;
DROP TABLE roles;
DROP TABLE users;
