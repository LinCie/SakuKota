-- +goose Up
-- `attributes` holds the values of the category's custom fields. The rating
-- columns summarize the place's reviews and are updated in the same statement
-- that stores a review. The sum is kept instead of the average so it stays
-- exact.
CREATE TABLE PLACES (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    sub_district_id BIGINT NOT NULL,
    category_id BIGINT NOT NULL,
    name VARCHAR(150) NOT NULL,
    address VARCHAR(255),
    location GEOMETRY(Point, 4326),
    operating_hours VARCHAR(255),
    estimated_price_range VARCHAR(100),
    short_description TEXT,
    attributes JSONB NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(attributes) = 'object'),
    rating_count INTEGER NOT NULL DEFAULT 0 CHECK (rating_count >= 0),
    rating_sum INTEGER NOT NULL DEFAULT 0 CHECK (rating_sum >= 0),
    FOREIGN KEY (sub_district_id) REFERENCES SUB_DISTRICTS(id),
    FOREIGN KEY (category_id) REFERENCES CATEGORIES(id)
);

CREATE INDEX places_location_idx ON PLACES USING GIST (location);

-- Postgres does not index the referencing side of a foreign key. These serve
-- the GET /places filters and the foreign key checks that scan this table when
-- a sub-district or category is deleted.
CREATE INDEX places_sub_district_id_idx ON PLACES (sub_district_id);
CREATE INDEX places_category_id_idx ON PLACES (category_id);

-- +goose Down
DROP TABLE PLACES;
