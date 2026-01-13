-- +goose Up
-- +goose StatementBegin

create table if not exists scripts
(
    id
    uuid
    not
    null
    default
    gen_random_uuid
(
),
    filename text not null,
    path text not null,
    upload_time timestamp with time zone not null default now(),
    text text,
    primary key
(
    id
)
    );

comment
on table scripts is 'Таблица скриптов';

comment
on column scripts.id is 'Уникальный идентификатор скрипта';
comment
on column scripts.filename is 'Имя аудио файла скрипта';
comment
on column scripts.path is 'Абсолютный путь до аудио файла скрипта';
comment
on column scripts.upload_time is 'Время загрузки аудио файла скрипта';
comment
on column scripts.text is 'Переведённые скрипты в отформатированном виде';
-- +goose StatementEnd