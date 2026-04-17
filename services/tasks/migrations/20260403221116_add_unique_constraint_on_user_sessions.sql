-- +goose Up
ALTER TABLE user_sessions
    ADD UNIQUE (user_id, fingerprint);