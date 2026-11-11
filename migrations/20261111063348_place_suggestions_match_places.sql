-- +goose Up
-- An approved suggestion becomes a place, so its columns take the limits and
-- checks of PLACES.
ALTER TABLE PLACE_SUGGESTIONS
    ALTER COLUMN address TYPE VARCHAR(255),
    ALTER COLUMN operating_hours TYPE VARCHAR(255),
    ALTER COLUMN estimated_price_range TYPE VARCHAR(100),
    ADD CONSTRAINT place_suggestions_attributes_check
        CHECK (jsonb_typeof(attributes) = 'object');

-- +goose Down
ALTER TABLE PLACE_SUGGESTIONS
    DROP CONSTRAINT place_suggestions_attributes_check,
    ALTER COLUMN estimated_price_range TYPE TEXT,
    ALTER COLUMN operating_hours TYPE TEXT,
    ALTER COLUMN address TYPE TEXT;
