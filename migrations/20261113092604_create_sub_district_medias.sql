-- +goose Up
-- A sub-district links up to 10 medias by position. Each media has a single
-- owner. Deleting a media closes the gap by shifting later medias down one
-- position. The deferrable key is checked at the end of that statement, so the
-- shift does not collide with a position that has not moved yet.
CREATE TABLE SUB_DISTRICT_MEDIAS (
    sub_district_id BIGINT NOT NULL,
    media_id BIGINT NOT NULL UNIQUE,
    position SMALLINT NOT NULL CHECK (position BETWEEN 1 AND 10),
    PRIMARY KEY (sub_district_id, position) DEFERRABLE INITIALLY IMMEDIATE,
    FOREIGN KEY (sub_district_id) REFERENCES SUB_DISTRICTS(id) ON DELETE CASCADE,
    FOREIGN KEY (media_id) REFERENCES MEDIAS(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE SUB_DISTRICT_MEDIAS;
