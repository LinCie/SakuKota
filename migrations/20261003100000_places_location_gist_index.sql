-- +goose Up
CREATE INDEX places_location_idx ON PLACES USING GIST (location);

-- +goose Down
DROP INDEX places_location_idx;
