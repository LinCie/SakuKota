-- +goose Up
-- Storage objects that are no longer referenced and must be deleted. A key is
-- queued in the same statement that drops its last reference, and a cleanup
-- worker deletes the object, retrying failed attempts once next_attempt_at
-- passes.
CREATE TABLE STORAGE_DELETIONS (
    key VARCHAR(255) PRIMARY KEY,
    attempts INTEGER NOT NULL DEFAULT 0 CHECK (attempts >= 0),
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE STORAGE_DELETIONS;
