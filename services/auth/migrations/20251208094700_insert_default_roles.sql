-- +goose Up
-- +goose StatementBegin
INSERT INTO roles (id, title)
VALUES (gen_random_uuid(), 'ROLE_ADMIN'),
       (gen_random_uuid(), 'ROLE_USER')
-- +goose StatementEnd
 