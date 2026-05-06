-- +goose Up
ALTER TABLE tasks ADD COLUMN plot_path text;
COMMENT ON COLUMN tasks.plot_path IS 'Путь до графика с результатами';