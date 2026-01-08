-- +goose Up
-- +goose StatementBegin
WITH admin_role_id as (SELECT id
                       FROM roles
                       WHERE title = 'ROLE_ADMIN'
                       LIMIT 1)
INSERT
INTO users_roles (user_id, role_id)
SELECT u.id, admin_role_id.id
FROM users u, admin_role_id
WHERE login = 'admin'
LIMIT 1;

WITH user_role_id as (SELECT id
                      FROM roles
                      WHERE title = 'ROLE_USER'
                      LIMIT 1)
INSERT
INTO users_roles (user_id, role_id)
SELECT u.id, user_role_id.id
FROM users u, user_role_id;
-- +goose StatementEnd