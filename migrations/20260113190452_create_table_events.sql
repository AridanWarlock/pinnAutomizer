-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS events
(
    id         uuid        not null primary key,
    topic      text        not null,
    data       bytea       not null,
    created_at timestamptz not null
);

COMMENT ON TABLE events is 'Таблица событий';
COMMENT ON COLUMN events.id is 'ID события';
COMMENT ON COLUMN events.topic is 'Название топика Kafka';
COMMENT ON COLUMN events.data is 'Payload сообщения';
COMMENT ON COLUMN events.created_at is 'Время создания события';
-- +goose StatementEnd
