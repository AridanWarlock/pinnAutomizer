-- +goose Up
-- +goose StatementBegin
CREATE TYPE task_status_enum as ENUM ('created', 'training', 'done');
COMMENT ON TYPE task_status_enum is 'Enum статус выполнения задачи';

CREATE TABLE IF NOT EXISTS tasks
(
    id                 uuid             not null primary key,
    name               text             not null,
    description        text,

    status             task_status_enum not null,
    constants          jsonb            not null,
    training_data_path text,
    results_path       text,

    user_id            uuid             not null,
    equation_id        uuid             not null,

    created_at         timestamptz      not null
);

COMMENT ON TABLE tasks is 'Таблица PINN задач';
COMMENT ON COLUMN tasks.id is 'ID задачи';
COMMENT ON COLUMN tasks.name is 'Название задачи';
COMMENT ON COLUMN tasks.description is 'Описание задачи';
COMMENT ON COLUMN tasks.user_id is 'ID пользователя, создавшего задачу';
COMMENT ON COLUMN tasks.equation_id is 'ID соответствующего уравнения мат. физики';
COMMENT ON COLUMN tasks.created_at is 'Время создания задачи';
-- +goose StatementEnd