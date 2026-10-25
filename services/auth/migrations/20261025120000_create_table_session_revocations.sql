-- +goose Up
CREATE TABLE session_revocations (
    jti uuid PRIMARY KEY CHECK (jti <> '00000000-0000-0000-0000-000000000000'),
    user_id uuid NOT NULL,
    fingerprint text NOT NULL
);

CREATE INDEX session_revocations_session_idx ON session_revocations (user_id, fingerprint);
COMMENT ON TABLE session_revocations IS 'Устойчивая очередь удаления Redis-сессий после отзыва refresh';
