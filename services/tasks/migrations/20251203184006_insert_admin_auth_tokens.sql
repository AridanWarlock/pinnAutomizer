-- +goose Up
-- +goose StatementBegin
INSERT INTO auth_tokens (user_id)
SELECT id
FROM users
WHERE login = 'admin'
LIMIT 1;
-- +goose StatementEnd
