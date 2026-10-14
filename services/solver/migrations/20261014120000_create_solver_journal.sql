-- +goose Up
CREATE TABLE attempts (
 run_id TEXT PRIMARY KEY,
 task_id TEXT NOT NULL UNIQUE,
 mode TEXT NOT NULL CHECK (mode IN ('train','retrain','predict')),
 state TEXT NOT NULL CHECK (state IN ('registered','created','start_requested','running','finished')),
 container_id TEXT NOT NULL DEFAULT '',
 created_at INTEGER NOT NULL,
 deadline INTEGER,
 completed_at INTEGER,
 exit_code INTEGER,
 error TEXT,
 CHECK ((state = 'finished') = (completed_at IS NOT NULL)),
 CHECK (state NOT IN ('start_requested','running') OR deadline IS NOT NULL),
 CHECK (state NOT IN ('created','start_requested','running') OR container_id != ''),
 CHECK (state != 'finished' OR COALESCE(exit_code = 0, 0) OR error IS NOT NULL)
);
CREATE TABLE outbox (
 sequence INTEGER PRIMARY KEY AUTOINCREMENT,
 event_id TEXT NOT NULL UNIQUE,
 run_id TEXT NOT NULL REFERENCES attempts(run_id),
 topic TEXT NOT NULL,
 message_key BLOB,
 value BLOB NOT NULL,
 headers TEXT NOT NULL
);
