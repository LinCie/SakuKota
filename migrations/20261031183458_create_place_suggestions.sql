-- +goose Up
-- A place a user suggests for the catalog. It stays pending until an admin
-- approves it, which creates a place from the details the admin checked, or
-- rejects it with an optional reason.
CREATE TABLE PLACE_SUGGESTIONS (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL,
    sub_district_id BIGINT NOT NULL,
    category_id BIGINT NOT NULL,
    name VARCHAR(150) NOT NULL,
    address TEXT,
    location GEOMETRY(Point, 4326) NOT NULL,
    operating_hours TEXT,
    estimated_price_range TEXT,
    short_description TEXT,
    attributes JSONB NOT NULL DEFAULT '{}',
    note TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    rejection_reason TEXT,
    place_id BIGINT,
    reviewed_by BIGINT,
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    FOREIGN KEY (user_id) REFERENCES USERS(id) ON DELETE CASCADE,
    FOREIGN KEY (sub_district_id) REFERENCES SUB_DISTRICTS(id) ON DELETE CASCADE,
    FOREIGN KEY (category_id) REFERENCES CATEGORIES(id),
    FOREIGN KEY (place_id) REFERENCES PLACES(id) ON DELETE SET NULL,
    FOREIGN KEY (reviewed_by) REFERENCES USERS(id) ON DELETE SET NULL
);

-- +goose Down
DROP TABLE PLACE_SUGGESTIONS;
