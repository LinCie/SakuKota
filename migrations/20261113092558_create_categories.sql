-- +goose Up
-- Categories are defined here, not through the API. `code`, `label` and
-- `color` are how clients draw a category; `color` is a palette key that the
-- web app maps to its styles. It is grouped by theme, so a hue marks a group
-- and the code and icon tell its categories apart:
--   cat   (blue)   health and emergency: clinic, hospital, puskesmas, police
--   board (green)  care and community: pharmacy, worship
--   chili (red)    food and daily goods: warung, minimarket, water and gas
--   plum           crafts: tailor, print and copy
--   sign  (yellow) city services and paperwork: laundry, bank, transit, offices
-- `has_price` says whether places of the category have a price at all (a
-- mosque has none), and `price_unit` what the price is per, such as 'kg'.
CREATE TABLE CATEGORIES (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    slug VARCHAR(100) NOT NULL UNIQUE,
    code VARCHAR(1) NOT NULL,
    label VARCHAR(100) NOT NULL,
    color VARCHAR(20) NOT NULL CHECK (color IN ('chili', 'plum', 'cat', 'sign', 'board')),
    has_price BOOLEAN NOT NULL DEFAULT FALSE,
    price_unit VARCHAR(20) CHECK (has_price OR price_unit IS NULL)
);

CREATE UNIQUE INDEX categories_name_lower_key ON CATEGORIES (LOWER(name));

INSERT INTO CATEGORIES (name, slug, code, label, color, has_price, price_unit)
VALUES
    ('Affordable Food & Warung', 'affordable-food-warung', 'W', 'Warung', 'chili', TRUE, NULL),
    ('Tailor', 'tailor', 'J', 'Penjahit', 'plum', TRUE, NULL),
    ('Local Clinic', 'local-clinic', 'K', 'Klinik', 'cat', TRUE, NULL),
    ('Laundry', 'laundry', 'L', 'Laundry', 'sign', TRUE, 'kg'),
    ('Pharmacy', 'pharmacy', 'A', 'Apotek', 'board', TRUE, NULL),
    ('Mosque & Musala', 'mosque', 'M', 'Masjid', 'board', FALSE, NULL),
    ('Other Worship', 'worship-other', 'I', 'Ibadah Lain', 'board', FALSE, NULL),
    ('Police Post', 'police-post', 'P', 'Pos Polisi', 'cat', FALSE, NULL),
    ('Hospital', 'hospital', 'R', 'Rumah Sakit', 'cat', TRUE, NULL),
    ('Community Health Center', 'puskesmas', 'S', 'Puskesmas', 'cat', TRUE, NULL),
    ('Bank & ATM', 'bank-atm', 'B', 'Bank & ATM', 'sign', FALSE, NULL),
    ('Minimarket', 'minimarket', 'T', 'Minimarket', 'chili', TRUE, NULL),
    ('Transit Stop', 'transit-stop', 'H', 'Halte/Stasiun', 'sign', TRUE, NULL),
    ('Public Service Office', 'public-service', 'U', 'Layanan Publik', 'sign', FALSE, NULL),
    ('Water & Gas Refill', 'water-gas', 'G', 'Air & Gas', 'chili', TRUE, NULL),
    ('Print & Copy', 'print-copy', 'F', 'Fotokopi', 'plum', TRUE, 'lembar');

-- +goose Down
DROP TABLE CATEGORIES;
