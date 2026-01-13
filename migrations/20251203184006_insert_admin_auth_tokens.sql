-- +goose Up
-- +goose StatementBegin
INSERT INTO auth_tokens (user_id)
select id
from users
where login = 'admin' limit 1;
-- +goose StatementEnd
