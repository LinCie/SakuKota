-- +goose Up
-- A user's report of an abusive place, or of a review when review_id is set.
-- It stays pending until an admin removes the content or dismisses the report.
-- place_id and review_id have no foreign keys, so a report outlives the
-- content it names and keeps the history; ids are never reused.
CREATE TABLE REPORTS (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL,
    place_id BIGINT NOT NULL,
    review_id BIGINT,
    reason VARCHAR(20) NOT NULL CHECK (reason IN ('spam', 'offensive', 'fake', 'other')),
    note TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'removed', 'dismissed')),
    resolved_by BIGINT,
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (user_id) REFERENCES USERS(id) ON DELETE CASCADE,
    FOREIGN KEY (resolved_by) REFERENCES USERS(id) ON DELETE SET NULL
);

-- A user reports the same content once while it is pending. The index also
-- serves the pending count made on every report.
CREATE UNIQUE INDEX reports_pending_user_target_idx ON REPORTS (user_id, place_id, review_id)
    NULLS NOT DISTINCT WHERE status = 'pending';
-- Serves closing every pending report of removed content.
CREATE INDEX reports_pending_place_id_idx ON REPORTS (place_id) WHERE status = 'pending';

-- +goose Down
DROP TABLE REPORTS;
