-- +goose Up
COMMENT ON TABLE file_cleanup IS 'Durable cleanup for incomplete task creation or task deletion; removed atomically when creation commits, otherwise acknowledged after directory removal';
COMMENT ON COLUMN file_cleanup.task_id IS 'Task UUID reserved by an incomplete creation or belonging to a deleted task; unique cleanup key and pagination cursor';
COMMENT ON COLUMN file_cleanup.data_path IS 'Absolute input directory owned by the task; file writes and cleanup use the same advisory filesystem lock';
COMMENT ON COLUMN file_cleanup.output_path IS 'Absolute output directory owned by the task';
