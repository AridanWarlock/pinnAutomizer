-- +goose Up
ALTER TABLE attempts ADD COLUMN cleaned_at INTEGER
 CHECK (cleaned_at IS NULL OR
  (state = 'finished' AND container_id != '' AND cleaned_at >= completed_at));
CREATE INDEX attempts_pending_cleanup ON attempts(run_id)
 WHERE state = 'finished' AND container_id != '' AND cleaned_at IS NULL;
