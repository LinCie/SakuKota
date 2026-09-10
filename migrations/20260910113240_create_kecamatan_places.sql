-- +goose Up
CREATE TABLE KECAMATAN (
    id INT PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    slug VARCHAR(100) NOT NULL UNIQUE
);

CREATE TABLE KECAMATAN_INFO (
    id INT PRIMARY KEY,
    kecamatan_id INT NOT NULL UNIQUE,
    flood_history TEXT,
    safety_crime_information TEXT,
    estimated_daily_spending DECIMAL(10, 2),
    static_map_image_url VARCHAR(255),
    FOREIGN KEY (kecamatan_id) REFERENCES KECAMATAN(id)
);

CREATE TABLE CATEGORY (
    id INT PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    slug VARCHAR(100) NOT NULL UNIQUE
);

CREATE TABLE PLACE (
    id INT PRIMARY KEY,
    kecamatan_id INT NOT NULL,
    name VARCHAR(150) NOT NULL,
    address VARCHAR(255),
    latitude DECIMAL(9, 6),
    longitude DECIMAL(9, 6),
    operating_hours VARCHAR(255),
    estimated_price_range VARCHAR(100),
    short_description TEXT,
    FOREIGN KEY (kecamatan_id) REFERENCES KECAMATAN(id)
);

CREATE TABLE PLACE_CATEGORY (
    place_id INT NOT NULL,
    category_id INT NOT NULL,
    PRIMARY KEY (place_id, category_id),
    FOREIGN KEY (place_id) REFERENCES PLACE(id),
    FOREIGN KEY (category_id) REFERENCES CATEGORY(id)
);

-- +goose Down
DROP TABLE PLACE_CATEGORY;
DROP TABLE PLACE;
DROP TABLE CATEGORY;
DROP TABLE KECAMATAN_INFO;
DROP TABLE KECAMATAN;
