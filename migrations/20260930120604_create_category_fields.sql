-- +goose Up
CREATE TABLE CATEGORY_FIELDS (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    category_id BIGINT NOT NULL,
    key VARCHAR(64) NOT NULL,
    label VARCHAR(100) NOT NULL,
    data_type VARCHAR(20) NOT NULL,
    required BOOLEAN NOT NULL DEFAULT FALSE,
    options JSONB,
    sort_order INTEGER NOT NULL,
    UNIQUE (category_id, key),
    FOREIGN KEY (category_id) REFERENCES CATEGORIES(id)
);

-- Custom fields per category, defined here like the categories themselves.
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
        (1, 'affordable-food-warung', 'halal', 'Halal', 'boolean', TRUE, NULL),
        (2, 'affordable-food-warung', 'serving_style', 'Cara penyajian', 'select', TRUE, '["Prasmanan", "Dimasak saat dipesan"]'),
        (3, 'affordable-food-warung', 'seating_capacity', 'Kapasitas tempat duduk', 'number', FALSE, NULL),
        (1, 'tailor', 'specialty', 'Keahlian', 'select', TRUE, '["Permak", "Reparasi", "Jahit baru"]'),
        (2, 'tailor', 'turnaround_days', 'Lama pengerjaan (hari)', 'range', TRUE, NULL),
        (3, 'tailor', 'same_day_service', 'Bisa selesai di hari yang sama', 'boolean', FALSE, NULL),
        (1, 'local-clinic', 'bpjs_accepted', 'Menerima BPJS', 'boolean', TRUE, NULL),
        (2, 'local-clinic', 'walk_in', 'Bisa datang tanpa janji', 'boolean', FALSE, NULL),
        (3, 'local-clinic', 'services', 'Layanan', 'text', FALSE, NULL),
        (4, 'local-clinic', 'opened_on', 'Mulai beroperasi', 'date', FALSE, NULL),
        (1, 'laundry', 'service_type', 'Jenis layanan', 'select', TRUE, '["Swalayan", "Titip cuci", "Swalayan & titip cuci"]'),
        (2, 'laundry', 'turnaround_hours', 'Lama pengerjaan (jam)', 'range', TRUE, NULL),
        (3, 'laundry', 'express_available', 'Layanan ekspres', 'boolean', FALSE, NULL),
        (4, 'laundry', 'minimum_weight_kg', 'Berat minimum (kg)', 'number', FALSE, NULL),
        (1, 'pharmacy', 'accepts_prescriptions', 'Melayani resep dokter', 'boolean', TRUE, NULL),
        (2, 'pharmacy', 'compounding', 'Melayani obat racikan', 'boolean', FALSE, NULL),
        (3, 'pharmacy', 'delivery', 'Pengantaran', 'select', FALSE, '["Tidak ada", "Kurir sendiri", "Ojek online"]')
)
INSERT INTO CATEGORY_FIELDS (category_id, key, label, data_type, required, options, sort_order)
SELECT c.id, scf.key, scf.label, scf.data_type, scf.required, scf.options::jsonb, scf.sort_order
FROM SEEDED_CATEGORY_FIELDS scf
JOIN CATEGORIES c ON c.slug = scf.category_slug;

ALTER TABLE PLACES
    ADD COLUMN attributes JSONB NOT NULL DEFAULT '{}'
        CHECK (jsonb_typeof(attributes) = 'object');

-- +goose Down
ALTER TABLE PLACES DROP COLUMN attributes;
DROP TABLE CATEGORY_FIELDS;
