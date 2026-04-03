-- +goose Up
-- +goose StatementBegin
INSERT INTO users (id, login, password_hash)
VALUES (gen_random_uuid(), 'admin', '$2a$12$ZcuWTry6lQS0zA3VzU6wPO3KrY.BCG60.TxTw2/W1m98olgzahm0q');
-- +goose StatementEnd
