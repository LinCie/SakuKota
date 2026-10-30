-- +goose Up
CREATE EXTENSION IF NOT EXISTS postgis;

CREATE TABLE SUB_DISTRICTS (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    slug VARCHAR(100) NOT NULL UNIQUE,
    flood_history TEXT,
    safety_crime_information TEXT,
    estimated_daily_spending DECIMAL(10, 2),
    static_map_image_url VARCHAR(255)
);

-- Categories are defined here, not through the API. `code`, `label` and
-- `color` are how clients draw a category; `color` is a palette key that the
-- web app maps to its styles.
CREATE TABLE CATEGORIES (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    slug VARCHAR(100) NOT NULL UNIQUE,
    code VARCHAR(1) NOT NULL,
    label VARCHAR(100) NOT NULL,
    color VARCHAR(20) NOT NULL CHECK (color IN ('chili', 'plum', 'cat', 'sign', 'board'))
);

INSERT INTO CATEGORIES (name, slug, code, label, color)
VALUES
    ('Affordable Food & Warung', 'affordable-food-warung', 'W', 'Warung', 'chili'),
    ('Tailor', 'tailor', 'J', 'Penjahit', 'plum'),
    ('Local Clinic', 'local-clinic', 'K', 'Klinik', 'cat'),
    ('Laundry', 'laundry', 'L', 'Laundry', 'sign'),
    ('Pharmacy', 'pharmacy', 'A', 'Apotek', 'board');

CREATE TABLE PLACES (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    sub_district_id BIGINT NOT NULL,
    name VARCHAR(150) NOT NULL,
    address VARCHAR(255),
    location GEOMETRY(Point, 4326),
    operating_hours VARCHAR(255),
    estimated_price_range VARCHAR(100),
    short_description TEXT,
    FOREIGN KEY (sub_district_id) REFERENCES SUB_DISTRICTS(id)
);

CREATE TABLE PLACE_CATEGORIES (
    place_id BIGINT NOT NULL,
    category_id BIGINT NOT NULL,
    PRIMARY KEY (place_id, category_id),
    FOREIGN KEY (place_id) REFERENCES PLACES(id),
    FOREIGN KEY (category_id) REFERENCES CATEGORIES(id)
);

-- +goose Down
DROP TABLE PLACE_CATEGORIES;
DROP TABLE PLACES;
DROP TABLE CATEGORIES;
DROP TABLE SUB_DISTRICTS;
