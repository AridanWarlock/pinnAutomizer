-- +goose Up
ALTER TYPE task_status_enum RENAME VALUE 'queue' TO 'in_queue';