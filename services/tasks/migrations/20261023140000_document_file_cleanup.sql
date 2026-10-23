-- +goose Up
COMMENT ON TABLE file_cleanup IS 'Durable file removal jobs retained after task deletion; acknowledged after both directories are removed';
COMMENT ON COLUMN file_cleanup.task_id IS 'Deleted task UUID; unique cleanup job key and keyset pagination cursor';
COMMENT ON COLUMN file_cleanup.data_path IS 'Absolute input directory owned by the deleted task';
COMMENT ON COLUMN file_cleanup.output_path IS 'Absolute output directory owned by the deleted task';
