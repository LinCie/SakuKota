-- +goose Up
-- Every stored image is a MEDIAS row. Images live in object storage; a row
-- keeps only the object key and the API builds the public URL. A place or
-- sub-district links its medias by position (see PLACE_MEDIAS and
-- SUB_DISTRICT_MEDIAS), and a sub-district's static map references one.
CREATE TABLE MEDIAS (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    key VARCHAR(255) NOT NULL UNIQUE,
    content_type VARCHAR(100) NOT NULL,
    size_bytes BIGINT NOT NULL CHECK (size_bytes >= 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE MEDIAS;
