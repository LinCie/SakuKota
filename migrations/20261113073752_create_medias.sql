-- +goose Up
-- Every stored image is a MEDIAS row. A place or sub-district links up to 10
-- of them by position, and a sub-district's static map references one. Each
-- media has a single owner. The old image keys hold only dev data and are
-- dropped without being moved over.
CREATE TABLE MEDIAS (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    key VARCHAR(255) NOT NULL UNIQUE,
    content_type VARCHAR(100) NOT NULL,
    size_bytes BIGINT NOT NULL CHECK (size_bytes >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE PLACE_MEDIAS (
    place_id BIGINT NOT NULL,
    media_id BIGINT NOT NULL UNIQUE,
    position SMALLINT NOT NULL CHECK (position BETWEEN 1 AND 10),
    PRIMARY KEY (place_id, position),
    FOREIGN KEY (place_id) REFERENCES PLACES(id) ON DELETE CASCADE,
    FOREIGN KEY (media_id) REFERENCES MEDIAS(id) ON DELETE CASCADE
);

CREATE TABLE SUB_DISTRICT_MEDIAS (
    sub_district_id BIGINT NOT NULL,
    media_id BIGINT NOT NULL UNIQUE,
    position SMALLINT NOT NULL CHECK (position BETWEEN 1 AND 10),
    PRIMARY KEY (sub_district_id, position),
    FOREIGN KEY (sub_district_id) REFERENCES SUB_DISTRICTS(id) ON DELETE CASCADE,
    FOREIGN KEY (media_id) REFERENCES MEDIAS(id) ON DELETE CASCADE
);

ALTER TABLE SUB_DISTRICTS
    ADD COLUMN static_map_media_id BIGINT UNIQUE REFERENCES MEDIAS(id) ON DELETE SET NULL,
    DROP COLUMN image_key,
    DROP COLUMN static_map_image_key;

ALTER TABLE PLACES DROP COLUMN image_key;

-- +goose Down
-- The media links dropped by Down cannot be restored; the key columns come
-- back empty.
ALTER TABLE PLACES ADD COLUMN image_key VARCHAR(255);

ALTER TABLE SUB_DISTRICTS
    ADD COLUMN image_key VARCHAR(255),
    ADD COLUMN static_map_image_key VARCHAR(255),
    DROP COLUMN static_map_media_id;

DROP TABLE SUB_DISTRICT_MEDIAS;
DROP TABLE PLACE_MEDIAS;
DROP TABLE MEDIAS;
