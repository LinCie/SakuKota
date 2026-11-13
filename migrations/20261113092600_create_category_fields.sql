-- +goose Up
-- The custom fields of a category, defined here like the categories
-- themselves. The place service validates a place's attributes by data type,
-- so a type it does not know must not be storable.
CREATE TABLE CATEGORY_FIELDS (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    category_id BIGINT NOT NULL,
    key VARCHAR(64) NOT NULL,
    label VARCHAR(100) NOT NULL,
    data_type VARCHAR(20) NOT NULL CHECK (data_type IN ('text', 'number', 'boolean', 'select', 'range', 'date')),
    required BOOLEAN NOT NULL DEFAULT FALSE,
    options JSONB,
    sort_order INTEGER NOT NULL,
    UNIQUE (category_id, key),
    FOREIGN KEY (category_id) REFERENCES CATEGORIES(id)
);

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
        (3, 'pharmacy', 'delivery', 'Pengantaran', 'select', FALSE, '["Tidak ada", "Kurir sendiri", "Ojek online"]'),
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
DROP TABLE CATEGORY_FIELDS;
