-- +goose Up
-- +goose envsub on
-- +goose StatementBegin
INSERT INTO users (id, login, password_hash)
VALUES (gen_random_uuid(), 'admin', ${ADMIN_PASSWORD_HASH});
-- +goose StatementEnd
