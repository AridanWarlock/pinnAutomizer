-- +goose Up
-- +goose StatementBegin
ALTER TABLE auth_tokens ADD CONSTRAINT fk_auth_tokens_users FOREIGN KEY (user_id) REFERENCES users;
ALTER TABLE users_roles ADD CONSTRAINT fk_users_roles_users FOREIGN KEY (user_id) REFERENCES users;
ALTER TABLE users_roles ADD CONSTRAINT fk_users_roles_roles FOREIGN KEY (role_id) REFERENCES roles;
-- +goose StatementEnd