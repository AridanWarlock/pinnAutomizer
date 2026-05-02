-- +goose Up
ALTER TYPE task_status_enum RENAME VALUE 'training' TO 'running';