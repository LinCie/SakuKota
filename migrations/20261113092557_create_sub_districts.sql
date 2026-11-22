-- +goose Up
CREATE EXTENSION IF NOT EXISTS postgis;

CREATE TABLE SUB_DISTRICTS (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    slug VARCHAR(100) NOT NULL UNIQUE,
    flood_history TEXT,
    safety_crime_information TEXT,
    estimated_daily_spending DECIMAL(10, 2),
    boundary GEOMETRY(MultiPolygon, 4326)
        CONSTRAINT sub_districts_boundary_check CHECK (ST_IsValid(boundary))
);

CREATE UNIQUE INDEX sub_districts_name_lower_key ON SUB_DISTRICTS (LOWER(name));

-- +goose Down
DROP TABLE SUB_DISTRICTS;
