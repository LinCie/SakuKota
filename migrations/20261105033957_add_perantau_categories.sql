-- +goose Up

-- More categories a perantau needs. `color` is grouped by theme, so a hue marks
-- a group and the code and icon tell its categories apart:
--   cat   (blue)   health and emergency: clinic, hospital, puskesmas, police
--   board (green)  care and community: pharmacy, worship
--   chili (red)    food and daily goods: warung, minimarket, water and gas
--   plum           crafts: tailor, print and copy
--   sign  (yellow) city services and paperwork: laundry, bank, transit, offices
INSERT INTO CATEGORIES (name, slug, code, label, color)
VALUES
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

WITH SEEDED_CATEGORY_FIELDS (
    sort_order,
    category_slug,
    key,
    label,
    data_type,
    required,
    options
) AS (
    VALUES
        (1, 'mosque', 'facility_type', 'Jenis tempat', 'select', TRUE, '["Masjid", "Musala"]'),
        (2, 'mosque', 'friday_prayer', 'Salat Jumat', 'boolean', TRUE, NULL),
        (3, 'mosque', 'women_prayer_area', 'Tempat salat perempuan', 'boolean', FALSE, NULL),
        (4, 'mosque', 'capacity', 'Kapasitas jemaah', 'number', FALSE, NULL),
        (1, 'worship-other', 'religion', 'Agama', 'select', TRUE, '["Kristen", "Katolik", "Hindu", "Buddha", "Konghucu"]'),
        (2, 'worship-other', 'service_schedule', 'Jadwal ibadah', 'text', FALSE, NULL),
        (1, 'police-post', 'facility_type', 'Jenis kantor', 'select', TRUE, '["Pos polisi", "Polsek", "Polres"]'),
        (2, 'police-post', 'open_24h', 'Buka 24 jam', 'boolean', TRUE, NULL),
        (3, 'police-post', 'skck_service', 'Layanan SKCK', 'boolean', FALSE, NULL),
        (4, 'police-post', 'phone', 'Telepon', 'text', FALSE, NULL),
        (1, 'hospital', 'igd_24h', 'IGD 24 jam', 'boolean', TRUE, NULL),
        (2, 'hospital', 'bpjs_accepted', 'Menerima BPJS', 'boolean', TRUE, NULL),
        (3, 'hospital', 'ownership', 'Kepemilikan', 'select', TRUE, '["Pemerintah", "Swasta", "TNI/Polri"]'),
        (4, 'hospital', 'hospital_class', 'Kelas rumah sakit', 'select', FALSE, '["A", "B", "C", "D"]'),
        (5, 'hospital', 'phone', 'Telepon', 'text', FALSE, NULL),
        (1, 'puskesmas', 'open_24h', 'Buka 24 jam', 'boolean', TRUE, NULL),
        (2, 'puskesmas', 'has_dental', 'Poli gigi', 'boolean', FALSE, NULL),
        (3, 'puskesmas', 'services', 'Layanan', 'text', FALSE, NULL),
        (4, 'puskesmas', 'phone', 'Telepon', 'text', FALSE, NULL),
        (1, 'bank-atm', 'banks', 'Bank', 'text', TRUE, NULL),
        (2, 'bank-atm', 'cash_deposit', 'Setor tunai', 'boolean', FALSE, NULL),
        (3, 'bank-atm', 'open_24h', 'Buka 24 jam', 'boolean', FALSE, NULL),
        (1, 'minimarket', 'store_type', 'Jenis toko', 'select', TRUE, '["Waralaba", "Kelontong"]'),
        (2, 'minimarket', 'open_24h', 'Buka 24 jam', 'boolean', FALSE, NULL),
        (3, 'minimarket', 'accepts_qris', 'Menerima QRIS', 'boolean', FALSE, NULL),
        (4, 'minimarket', 'has_atm', 'Ada ATM', 'boolean', FALSE, NULL),
        (1, 'transit-stop', 'mode', 'Moda', 'select', TRUE, '["TransJakarta", "KRL", "MRT", "LRT", "Angkot"]'),
        (2, 'transit-stop', 'lines', 'Rute/lintas', 'text', FALSE, NULL),
        (3, 'transit-stop', 'motorcycle_parking', 'Parkir motor', 'boolean', FALSE, NULL),
        (1, 'public-service', 'office_type', 'Jenis kantor', 'select', TRUE, '["Kelurahan", "Kecamatan", "Dukcapil", "Samsat", "Kantor Pos", "BPJS"]'),
        (2, 'public-service', 'services', 'Layanan', 'text', FALSE, NULL),
        (3, 'public-service', 'appointment_required', 'Perlu janji temu', 'boolean', FALSE, NULL),
        (1, 'water-gas', 'service', 'Layanan', 'select', TRUE, '["Air isi ulang", "Gas LPG", "Air & gas"]'),
        (2, 'water-gas', 'delivery', 'Antar ke rumah', 'boolean', FALSE, NULL),
        (1, 'print-copy', 'color_print', 'Cetak berwarna', 'boolean', FALSE, NULL),
        (2, 'print-copy', 'passport_photo', 'Pas foto', 'boolean', FALSE, NULL),
        (3, 'print-copy', 'open_24h', 'Buka 24 jam', 'boolean', FALSE, NULL)
)
INSERT INTO CATEGORY_FIELDS (category_id, key, label, data_type, required, options, sort_order)
SELECT c.id, scf.key, scf.label, scf.data_type, scf.required, scf.options::jsonb, scf.sort_order
FROM SEEDED_CATEGORY_FIELDS scf
JOIN CATEGORIES c ON c.slug = scf.category_slug;

-- +goose Down

-- Places and suggestions in these categories go with them. Images stored for
-- the deleted places are not queued for cleanup.
DELETE FROM PLACE_SUGGESTIONS
WHERE category_id IN (SELECT id FROM CATEGORIES WHERE slug IN (
    'mosque', 'worship-other', 'police-post', 'hospital', 'puskesmas', 'bank-atm',
    'minimarket', 'transit-stop', 'public-service', 'water-gas', 'print-copy'
));

DELETE FROM PLACES
WHERE category_id IN (SELECT id FROM CATEGORIES WHERE slug IN (
    'mosque', 'worship-other', 'police-post', 'hospital', 'puskesmas', 'bank-atm',
    'minimarket', 'transit-stop', 'public-service', 'water-gas', 'print-copy'
));

DELETE FROM CATEGORY_FIELDS
WHERE category_id IN (SELECT id FROM CATEGORIES WHERE slug IN (
    'mosque', 'worship-other', 'police-post', 'hospital', 'puskesmas', 'bank-atm',
    'minimarket', 'transit-stop', 'public-service', 'water-gas', 'print-copy'
));

DELETE FROM CATEGORIES
WHERE slug IN (
    'mosque', 'worship-other', 'police-post', 'hospital', 'puskesmas', 'bank-atm',
    'minimarket', 'transit-stop', 'public-service', 'water-gas', 'print-copy'
);
