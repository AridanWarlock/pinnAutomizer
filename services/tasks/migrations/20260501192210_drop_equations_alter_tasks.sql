-- +goose Up
ALTER TABLE tasks DROP COLUMN constants;
ALTER TABLE tasks DROP COLUMN training_data_path;
ALTER TABLE tasks DROP COLUMN results_path;
ALTER TABLE tasks DROP COLUMN equation_id;

DROP TABLE equations;

ALTER TABLE tasks ADD COLUMN error text;
ALTER TABLE tasks ADD COLUMN mode text;
ALTER TABLE tasks ALTER COLUMN mode SET not null;
ALTER TABLE tasks ADD COLUMN data_path text;
ALTER TABLE tasks ALTER COLUMN data_path SET not null;
ALTER TABLE tasks ADD COLUMN output_path text;
ALTER TABLE tasks ALTER COLUMN output_path SET not null;

COMMENT ON COLUMN tasks.mode IS 'Вид PINN задачи';
COMMENT ON COLUMN tasks.data_path IS 'Путь до директории input файлов';
COMMENT ON COLUMN tasks.output_path IS 'Путь до директории output файлов';