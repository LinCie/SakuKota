-- +goose Up
CREATE TABLE SUB_DISTRICTS (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    slug VARCHAR(100) NOT NULL UNIQUE,
    flood_history TEXT,
    safety_crime_information TEXT,
    estimated_daily_spending DECIMAL(10, 2),
    static_map_media_id BIGINT UNIQUE REFERENCES MEDIAS(id) ON DELETE SET NULL
);

CREATE UNIQUE INDEX sub_districts_name_lower_key ON SUB_DISTRICTS (LOWER(name));

-- +goose Down
DROP TABLE SUB_DISTRICTS;
