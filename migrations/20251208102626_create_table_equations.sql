-- +goose Up
-- +goose StatementBegin
CREATE TYPE equation_type_enum AS ENUM ('heat', 'wave');
COMMENT ON TYPE equation_type_enum is 'Enum тип уравнения мат. физики';

CREATE TABLE IF NOT EXISTS equations
(
    id    uuid     not null primary key,
    type  equation_type_enum not null
);

COMMENT ON TABLE equations is 'Таблица уравнений мат.физики';
COMMENT ON COLUMN equations.type is 'Enum тип уравнения';
-- +goose StatementEnd