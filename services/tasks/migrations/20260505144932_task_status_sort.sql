-- +goose Up
ALTER TYPE task_status_enum RENAME TO task_status_enum_old;
CREATE TYPE task_status_enum AS ENUM ('created', 'in_queue', 'running', 'error', 'done');

ALTER TABLE tasks
    ALTER COLUMN status TYPE task_status_enum
    USING status::text::task_status_enum;

DROP TYPE task_status_enum_old;

COMMENT ON TYPE task_status_enum IS 'Enum статус выполнения задачи';