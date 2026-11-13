-- +goose Up
-- A user's review of a place: a star rating and an optional comment. Each
-- user reviews a place at most once.
CREATE TABLE PLACE_REVIEWS (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    place_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL,
    rating SMALLINT NOT NULL CHECK (rating BETWEEN 1 AND 5),
    comment TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (place_id, user_id),
    FOREIGN KEY (place_id) REFERENCES PLACES(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES USERS(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE PLACE_REVIEWS;
