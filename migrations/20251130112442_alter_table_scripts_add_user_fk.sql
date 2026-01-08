-- +goose Up
-- +goose StatementBegin
ALTER TABLE scripts
    ADD COLUMN user_id uuid;

UPDATE scripts
SET user_id = (select id
               from users
               where login = 'admin'
               limit 1);

ALTER TABLE scripts
    ALTER COLUMN user_id SET NOT NULL;
-- +goose StatementEnd
