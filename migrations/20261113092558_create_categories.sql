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
CREATE TABLE CATEGORIES (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    slug VARCHAR(100) NOT NULL UNIQUE,
    code VARCHAR(1) NOT NULL,
    label VARCHAR(100) NOT NULL,
    color VARCHAR(20) NOT NULL CHECK (color IN ('chili', 'plum', 'cat', 'sign', 'board'))
);

CREATE UNIQUE INDEX categories_name_lower_key ON CATEGORIES (LOWER(name));

INSERT INTO CATEGORIES (name, slug, code, label, color)
VALUES
    ('Affordable Food & Warung', 'affordable-food-warung', 'W', 'Warung', 'chili'),
    ('Tailor', 'tailor', 'J', 'Penjahit', 'plum'),
    ('Local Clinic', 'local-clinic', 'K', 'Klinik', 'cat'),
    ('Laundry', 'laundry', 'L', 'Laundry', 'sign'),
    ('Pharmacy', 'pharmacy', 'A', 'Apotek', 'board'),
    ('Mosque & Musala', 'mosque', 'M', 'Masjid', 'board'),
    ('Other Worship', 'worship-other', 'I', 'Ibadah Lain', 'board'),
    ('Police Post', 'police-post', 'P', 'Pos Polisi', 'cat'),
    ('Hospital', 'hospital', 'R', 'Rumah Sakit', 'cat'),
    ('Community Health Center', 'puskesmas', 'S', 'Puskesmas', 'cat'),
    ('Bank & ATM', 'bank-atm', 'B', 'Bank & ATM', 'sign'),
    ('Minimarket', 'minimarket', 'T', 'Minimarket', 'chili'),
    ('Transit Stop', 'transit-stop', 'H', 'Halte/Stasiun', 'sign'),
    ('Public Service Office', 'public-service', 'U', 'Layanan Publik', 'sign'),
    ('Water & Gas Refill', 'water-gas', 'G', 'Air & Gas', 'chili'),
    ('Print & Copy', 'print-copy', 'F', 'Fotokopi', 'plum');

-- +goose Down
DROP TABLE CATEGORIES;
