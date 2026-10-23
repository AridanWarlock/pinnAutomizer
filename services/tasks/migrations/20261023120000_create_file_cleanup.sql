-- +goose Up
CREATE TABLE file_cleanup (
 task_id uuid PRIMARY KEY,
 data_path text NOT NULL CHECK (data_path <> ''),
 output_path text NOT NULL CHECK (output_path <> '')
);
