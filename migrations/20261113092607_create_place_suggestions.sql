-- +goose Up
-- A place a user suggests for the catalog. It stays pending until an admin
-- approves it, which creates a place from the details the admin checked, or
-- rejects it with an optional reason. Its columns take the limits and checks
-- of PLACES.
CREATE TABLE PLACE_SUGGESTIONS (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL,
    sub_district_id BIGINT NOT NULL,
    category_id BIGINT NOT NULL,
    name VARCHAR(150) NOT NULL,
    address VARCHAR(255),
    location GEOMETRY(Point, 4326) NOT NULL,
    operating_hours JSONB CHECK (jsonb_typeof(operating_hours) = 'array'),
    price_min INTEGER CHECK (price_min >= 0),
    price_max INTEGER,
    short_description TEXT,
    attributes JSONB NOT NULL DEFAULT '{}' CHECK (jsonb_typeof(attributes) = 'object'),
    note TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    rejection_reason TEXT,
    place_id BIGINT,
    reviewed_by BIGINT,
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK ((price_min IS NULL) = (price_max IS NULL) AND price_max >= price_min),
    FOREIGN KEY (user_id) REFERENCES USERS(id) ON DELETE CASCADE,
    FOREIGN KEY (sub_district_id) REFERENCES SUB_DISTRICTS(id) ON DELETE CASCADE,
    FOREIGN KEY (category_id) REFERENCES CATEGORIES(id),
    FOREIGN KEY (place_id) REFERENCES PLACES(id) ON DELETE SET NULL,
    FOREIGN KEY (reviewed_by) REFERENCES USERS(id) ON DELETE SET NULL
);

-- Serve the pending-suggestion count made on every submit, and the foreign key
-- cascades that scan this table when a user or place is deleted.
CREATE INDEX place_suggestions_user_id_status_idx ON PLACE_SUGGESTIONS (user_id, status);
CREATE INDEX place_suggestions_place_id_idx ON PLACE_SUGGESTIONS (place_id);

-- +goose Down
DROP TABLE PLACE_SUGGESTIONS;
