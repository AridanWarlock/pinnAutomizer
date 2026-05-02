-- +goose Up
ALTER TABLE events RENAME COLUMN id TO id_key;
ALTER TABLE events ALTER COLUMN id_key TYPE text;