INSERT INTO SUB_DISTRICTS (
    name,
    slug,
    flood_history,
    safety_crime_information,
    estimated_daily_spending,
    static_map_image_url
)
VALUES
    (
        'Setiabudi',
        'setiabudi',
        'Some streets near the Cideng and Karet drainage channels can pool after intense rain. Main roads usually clear quickly, but low-lying alleys may be difficult to cross during a storm.',
        'Busy commuter corridors are generally well lit. Use normal late-night precautions around quiet alleys and keep belongings secure in crowded areas.',
        150000.00,
        'https://placehold.co/800x500/png?text=Setiabudi+Map'
    ),
    (
        'Mampang Prapatan',
        'mampang-prapatan',
        'Short-lived flooding can occur around drainage bottlenecks near Tegal Parang and Mampang. Check local conditions during heavy rain before taking smaller roads.',
        'Commercial streets are active into the evening. Residential lanes are quieter, so use established routes and app-based transport when returning late.',
        135000.00,
        'https://placehold.co/800x500/png?text=Mampang+Prapatan+Map'
    )
ON CONFLICT (slug) DO NOTHING;

WITH SEEDED_PLACES (
    name,
    sub_district_slug,
    address,
    longitude,
    latitude,
    operating_hours,
    estimated_price_range,
    short_description,
    category_slug,
    attributes
) AS (
    VALUES
        (
            'Warung Nasi Sari Rasa',
            'setiabudi',
            'Jl. Setiabudi Tengah No. 12',
            106.8235,
            -6.2097,
            'Daily, 06:30-14:00',
            'Rp15.000-Rp35.000',
            'A simple rice counter with rotating home-style dishes and generous portions.',
            'affordable-food-warung',
            '{"halal": true, "serving_style": "Prasmanan", "seating_capacity": 20}'
        ),
        (
            'Kedai Pagi Karet',
            'setiabudi',
            'Jl. Karet Pasar Baru Timur No. 8',
            106.8271,
            -6.2140,
            'Mon-Sat, 06:00-20:00',
            'Rp12.000-Rp30.000',
            'Quick breakfast, coffee, and takeaway meals for commuters around Karet.',
            'affordable-food-warung',
            '{"halal": true, "serving_style": "Dimasak saat dipesan", "seating_capacity": 8}'
        ),
        (
            'Jarum Rapi Setiabudi',
            'setiabudi',
            'Jl. Setiabudi Selatan No. 21',
            106.8218,
            -6.2114,
            'Mon-Sat, 09:00-18:00',
            'Rp25.000-Rp150.000',
            'Neighborhood tailor for trouser hems, uniform repairs, and small alterations.',
            'tailor',
            '{"specialty": "Permak", "turnaround_days": {"min": 1, "max": 3}, "same_day_service": false}'
        ),
        (
            'Benang Sudirman',
            'setiabudi',
            'Jl. Prof. Dr. Satrio No. 17',
            106.8290,
            -6.2151,
            'Mon-Sat, 10:00-19:00',
            'Rp30.000-Rp250.000',
            'Small alteration shop that handles office wear and same-day repairs when available.',
            'tailor',
            '{"specialty": "Permak", "turnaround_days": {"min": 0, "max": 2}, "same_day_service": true}'
        ),
        (
            'Klinik Sehat Setiabudi',
            'setiabudi',
            'Jl. Setiabudi Raya No. 44',
            106.8247,
            -6.2170,
            'Daily, 08:00-21:00',
            'Rp50.000-Rp200.000',
            'General outpatient clinic for minor illnesses, basic checks, and first aid.',
            'local-clinic',
            '{"bpjs_accepted": true, "walk_in": true, "services": "Pemeriksaan umum, cek kesehatan dasar, pertolongan pertama", "opened_on": "2015-03-02"}'
        ),
        (
            'Klinik Menteng Dalam',
            'setiabudi',
            'Jl. Menteng Dalam VIII No. 5',
            106.8303,
            -6.2202,
            'Mon-Sat, 08:00-20:00',
            'Rp50.000-Rp175.000',
            'A compact clinic serving routine consultations and simple laboratory referrals.',
            'local-clinic',
            '{"bpjs_accepted": true, "walk_in": false, "services": "Konsultasi dokter umum, rujukan laboratorium", "opened_on": "2018-07-16"}'
        ),
        (
            'Busa Laundry 24 Jam',
            'setiabudi',
            'Jl. Karet Belakang No. 9',
            106.8262,
            -6.2090,
            'Daily, 07:00-23:00',
            'Rp8.000-Rp25.000/kg',
            'Self-service and drop-off laundry with late pickup hours for apartment residents.',
            'laundry',
            '{"service_type": "Swalayan & titip cuci", "turnaround_hours": {"min": 12, "max": 24}, "express_available": false, "minimum_weight_kg": 2}'
        ),
        (
            'CuciBersih Karet',
            'setiabudi',
            'Jl. Karet Sawah No. 16',
            106.8199,
            -6.2167,
            'Daily, 08:00-20:00',
            'Rp7.000-Rp22.000/kg',
            'Affordable wash, dry, and fold service with express options for work clothes.',
            'laundry',
            '{"service_type": "Titip cuci", "turnaround_hours": {"min": 6, "max": 48}, "express_available": true, "minimum_weight_kg": 3}'
        ),
        (
            'Apotek Sehat Sudirman',
            'setiabudi',
            'Jl. Jenderal Sudirman No. 55',
            106.8310,
            -6.2122,
            'Daily, 07:00-22:00',
            'Rp5.000-Rp250.000',
            'Convenient pharmacy for common medicines, vitamins, and basic health supplies.',
            'pharmacy',
            '{"accepts_prescriptions": true, "compounding": false, "delivery": "Ojek online"}'
        ),
        (
            'Apotek Taman Rasuna',
            'setiabudi',
            'Jl. Taman Rasuna Selatan No. 3',
            106.8341,
            -6.2181,
            'Daily, 08:00-21:00',
            'Rp5.000-Rp300.000',
            'Local pharmacy with over-the-counter medicine and personal care essentials.',
            'pharmacy',
            '{"accepts_prescriptions": false, "compounding": false, "delivery": "Tidak ada"}'
        ),
        (
            'Warung Bu Narti Mampang',
            'mampang-prapatan',
            'Jl. Mampang Prapatan Raya No. 31',
            106.8192,
            -6.2468,
            'Daily, 07:00-15:00',
            'Rp15.000-Rp35.000',
            'Casual warung serving rice plates, vegetables, and fried side dishes at lunch.',
            'affordable-food-warung',
            '{"halal": true, "serving_style": "Prasmanan", "seating_capacity": 16}'
        ),
        (
            'Dapur Gang Lima',
            'mampang-prapatan',
            'Jl. Bangka XI No. 5',
            106.8138,
            -6.2524,
            'Mon-Sat, 10:00-21:00',
            'Rp18.000-Rp45.000',
            'A tiny neighborhood kitchen with filling Indonesian meals and takeaway portions.',
            'affordable-food-warung',
            '{"halal": true, "serving_style": "Dimasak saat dipesan", "seating_capacity": 6}'
        ),
        (
            'Jahit Cepat Mampang',
            'mampang-prapatan',
            'Jl. Mampang Prapatan VIII No. 14',
            106.8162,
            -6.2492,
            'Mon-Sat, 09:00-18:00',
            'Rp20.000-Rp140.000',
            'Fast clothing repairs, zipper replacement, and basic resizing for daily wear.',
            'tailor',
            '{"specialty": "Reparasi", "turnaround_days": {"min": 0, "max": 2}, "same_day_service": true}'
        ),
        (
            'Rumah Kancing Bangka',
            'mampang-prapatan',
            'Jl. Bangka Raya No. 27',
            106.8099,
            -6.2543,
            'Mon-Sat, 10:00-19:00',
            'Rp25.000-Rp225.000',
            'Friendly tailor for hemming, patching, and simple custom work.',
            'tailor',
            '{"specialty": "Jahit baru", "turnaround_days": {"min": 3, "max": 7}, "same_day_service": false}'
        ),
        (
            'Klinik Pratama Kemang Selatan',
            'mampang-prapatan',
            'Jl. Kemang Selatan I No. 10',
            106.8250,
            -6.2528,
            'Daily, 08:00-22:00',
            'Rp60.000-Rp225.000',
            'General clinic offering consultations, wound care, and basic health screening.',
            'local-clinic',
            '{"bpjs_accepted": true, "walk_in": true, "services": "Konsultasi dokter umum, perawatan luka, skrining kesehatan", "opened_on": "2012-11-05"}'
        ),
        (
            'Klinik Keluarga Mampang',
            'mampang-prapatan',
            'Jl. Tegal Parang Utara No. 18',
            106.8174,
            -6.2570,
            'Mon-Sat, 08:00-20:00',
            'Rp50.000-Rp175.000',
            'Walk-in clinic for everyday health concerns and referrals to nearby hospitals.',
            'local-clinic',
            '{"bpjs_accepted": false, "walk_in": true, "services": "Konsultasi dokter umum, rujukan rumah sakit", "opened_on": "2020-01-20"}'
        ),
        (
            'Laundry Kilat Tegal Parang',
            'mampang-prapatan',
            'Jl. Tegal Parang Selatan No. 7',
            106.8115,
            -6.2486,
            'Daily, 07:00-22:00',
            'Rp8.000-Rp25.000/kg',
            'Drop-off laundry with an express service for residents and boarding-house tenants.',
            'laundry',
            '{"service_type": "Titip cuci", "turnaround_hours": {"min": 4, "max": 24}, "express_available": true, "minimum_weight_kg": 2}'
        ),
        (
            'FreshFold Laundry Bangka',
            'mampang-prapatan',
            'Jl. Bangka IX No. 22',
            106.8088,
            -6.2601,
            'Daily, 08:00-21:00',
            'Rp7.000-Rp22.000/kg',
            'Wash-and-fold service that also accepts blankets and small household items.',
            'laundry',
            '{"service_type": "Titip cuci", "turnaround_hours": {"min": 24, "max": 48}, "express_available": false, "minimum_weight_kg": 3}'
        ),
        (
            'Apotek Prapanca',
            'mampang-prapatan',
            'Jl. Prapanca Raya No. 41',
            106.8210,
            -6.2472,
            'Daily, 08:00-22:00',
            'Rp5.000-Rp275.000',
            'Pharmacy carrying common prescriptions, vitamins, and basic first-aid supplies.',
            'pharmacy',
            '{"accepts_prescriptions": true, "compounding": true, "delivery": "Kurir sendiri"}'
        ),
        (
            'Apotek Mampang Sejahtera',
            'mampang-prapatan',
            'Jl. Mampang Prapatan II No. 9',
            106.8146,
            -6.2561,
            'Daily, 07:00-21:00',
            'Rp5.000-Rp250.000',
            'Convenient neighborhood pharmacy for medicine refills and everyday health needs.',
            'pharmacy',
            '{"accepts_prescriptions": true, "compounding": false, "delivery": "Ojek online"}'
        )
)
INSERT INTO PLACES (
    sub_district_id,
    category_id,
    name,
    address,
    location,
    operating_hours,
    estimated_price_range,
    short_description,
    attributes
)
SELECT
    sd.id,
    c.id,
    sp.name,
    sp.address,
    ST_SetSRID(ST_MakePoint(sp.longitude, sp.latitude), 4326),
    sp.operating_hours,
    sp.estimated_price_range,
    sp.short_description,
    sp.attributes::jsonb
FROM SEEDED_PLACES sp
JOIN SUB_DISTRICTS sd ON sd.slug = sp.sub_district_slug
JOIN CATEGORIES c ON c.slug = sp.category_slug
WHERE NOT EXISTS (
    SELECT 1
    FROM PLACES existing
    WHERE existing.sub_district_id = sd.id
      AND existing.name = sp.name
);

-- Default admin account. Password: admin123 (change after first sign-in).
INSERT INTO USERS (name, email, password_hash, role)
VALUES (
    'Admin',
    'admin@rantau.go',
    '$argon2id$v=19$m=19456,t=2,p=1$8BH3b7P0mSW3iajNf9Rv4w$NDJvbJFopj4gJxJqltPEI+QcFoBc0dopSeoKBZyXrIQ',
    'admin'
)
ON CONFLICT (email) DO NOTHING;
