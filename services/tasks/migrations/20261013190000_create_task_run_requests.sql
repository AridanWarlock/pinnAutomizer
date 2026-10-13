-- +goose Up
CREATE TABLE task_run_requests
(
    user_id    uuid        NOT NULL,
    id_key     text        NOT NULL CHECK (octet_length(id_key) BETWEEN 1 AND 256),
    task_id    uuid        NOT NULL,
    event_id   uuid        NOT NULL UNIQUE,
    created_at timestamptz NOT NULL,
    PRIMARY KEY (user_id, id_key)
);

COMMENT ON TABLE task_run_requests IS 'Принятые запросы запуска; сохраняются после отправки outbox и удаления задачи';
COMMENT ON COLUMN task_run_requests.task_id IS 'ID исходной задачи без FK: удаление задачи не отменяет принятого запроса';
COMMENT ON COLUMN task_run_requests.event_id IS 'ID события запуска, независимый от пользовательского ключа';

-- +goose Down
DROP TABLE task_run_requests;
