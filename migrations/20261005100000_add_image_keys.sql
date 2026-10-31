-- +goose Up
-- Images live in object storage. Rows keep only the object key; the API builds
-- the public URL. The static map moves to storage too, so its URL column is
-- replaced by a key.
ALTER TABLE PLACES ADD COLUMN image_key VARCHAR(255);

ALTER TABLE SUB_DISTRICTS
    ADD COLUMN image_key VARCHAR(255),
    ADD COLUMN static_map_image_key VARCHAR(255),
    DROP COLUMN static_map_image_url;

-- +goose Down
-- The static map URLs dropped by Up cannot be restored.
ALTER TABLE SUB_DISTRICTS
    ADD COLUMN static_map_image_url VARCHAR(255),
    DROP COLUMN static_map_image_key,
    DROP COLUMN image_key;

ALTER TABLE PLACES DROP COLUMN image_key;
