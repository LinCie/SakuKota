-- Load-test data: made-up places, 50,000 of them, over 17 kecamatan. Use
-- cmd/seed for the real data. Run one or the other on an empty database.

-- Jakarta kecamatan. A temporary table so the bulk places below can use each
-- one's approximate center.
CREATE TEMPORARY TABLE SEEDED_SUB_DISTRICTS (
    name,
    slug,
    flood_history,
    safety_crime_information,
    estimated_daily_spending,
    longitude,
    latitude
) ON COMMIT DROP AS
    VALUES
        (
            'Setiabudi',
            'setiabudi',
            'Some streets near the Cideng and Karet drainage channels can pool after intense rain. Main roads usually clear quickly, but low-lying alleys may be difficult to cross during a storm.',
            'Busy commuter corridors are generally well lit. Use normal late-night precautions around quiet alleys and keep belongings secure in crowded areas.',
            150000.00,
            106.8300,
            -6.2150
        ),
        (
            'Mampang Prapatan',
            'mampang-prapatan',
            'Short-lived flooding can occur around drainage bottlenecks near Tegal Parang and Mampang. Check local conditions during heavy rain before taking smaller roads.',
            'Commercial streets are active into the evening. Residential lanes are quieter, so use established routes and app-based transport when returning late.',
            135000.00,
            106.8230,
            -6.2450
        ),
        (
            'Tebet',
            'tebet',
            'Lanes near the Ciliwung river can flood after long, heavy rain. Main roads around the station usually stay passable.',
            'Food streets and the station area stay lively into the night. Keep belongings close in crowds and walk lit main roads when it is late.',
            140000.00,
            106.8560,
            -6.2260
        ),
        (
            'Menteng',
            'menteng',
            'Main avenues drain well, though some side streets can pool briefly during intense rain.',
            'Quiet, well-lit residential streets with offices and embassies. Usual precautions apply around busy evening spots.',
            175000.00,
            106.8340,
            -6.1960
        ),
        (
            'Tanah Abang',
            'tanah-abang',
            'Streets near the market and canals can pool after heavy rain, slowing traffic around the station.',
            'The market gets very crowded, so keep bags in front and watch for pickpockets. Daytime traffic is heavy.',
            130000.00,
            106.8130,
            -6.2030
        ),
        (
            'Kebayoran Baru',
            'kebayoran-baru',
            'Mostly well drained, with brief pooling at a few low points during storms.',
            'Commercial areas stay busy into the night. Use app-based transport when returning late.',
            180000.00,
            106.7990,
            -6.2440
        ),
        (
            'Pancoran',
            'pancoran',
            'Low-lying lanes near drainage channels may pool after heavy rain. Main roads usually clear quickly.',
            'Arterial roads carry heavy traffic, so cross with care and use pedestrian bridges.',
            135000.00,
            106.8450,
            -6.2550
        ),
        (
            'Gambir',
            'gambir',
            'Central avenues drain well, but some side streets see short-lived pooling in storms.',
            'A central area of offices and parks that is busy by day. Keep belongings close around the station.',
            160000.00,
            106.8200,
            -6.1760
        ),
        (
            'Senen',
            'senen',
            'Streets around the market and station can pool after heavy rain.',
            'A crowded market and transport hub. Watch for pickpockets and take care on quiet streets at night.',
            120000.00,
            106.8450,
            -6.1830
        ),
        (
            'Kemayoran',
            'kemayoran',
            'Some low-lying neighborhoods flood in the rainy season. Check conditions before travelling in heavy rain.',
            'A mix of homes and event venues. Quieter streets at night call for normal precautions.',
            125000.00,
            106.8530,
            -6.1620
        ),
        (
            'Grogol Petamburan',
            'grogol-petamburan',
            'Low areas near rivers and canals can flood after heavy rain.',
            'Busy around campuses and malls. Keep belongings secure on crowded public transport.',
            130000.00,
            106.7900,
            -6.1650
        ),
        (
            'Kelapa Gading',
            'kelapa-gading',
            'Prone to flooding in heavy rainy seasons, and some streets can be impassable for hours.',
            'Residential and mall area with security in the commercial zones. Normal precautions on side streets at night.',
            165000.00,
            106.9050,
            -6.1600
        ),
        (
            'Cengkareng',
            'cengkareng',
            'Low-lying parts can flood in the rainy season. Follow local updates during storms.',
            'A large residential district. Use main roads at night and keep vehicles locked.',
            115000.00,
            106.7350,
            -6.1500
        ),
        (
            'Jatinegara',
            'jatinegara',
            'Neighborhoods along the Ciliwung can flood when the river rises. Follow local warnings.',
            'A busy market and station area. Watch belongings in crowds.',
            120000.00,
            106.8700,
            -6.2250
        ),
        (
            'Pasar Minggu',
            'pasar-minggu',
            'Areas near the Ciliwung can flood when the river rises after upstream rain.',
            'A busy market and station. Keep valuables secure; residential lanes are quieter at night.',
            125000.00,
            106.8400,
            -6.2850
        ),
        (
            'Cilandak',
            'cilandak',
            'Generally drains well, with occasional pooling on some roads after heavy storms.',
            'Mostly residential with commercial strips. Normal precautions at night.',
            155000.00,
            106.8000,
            -6.2900
        ),
        (
            'Kebayoran Lama',
            'kebayoran-lama',
            'Some low areas near rivers may flood during heavy rain.',
            'A busy market and arterial roads. Keep belongings secure in crowds.',
            135000.00,
            106.7780,
            -6.2450
        );

INSERT INTO SUB_DISTRICTS (
    name,
    slug,
    flood_history,
    safety_crime_information,
    estimated_daily_spending
)
SELECT
    name,
    slug,
    flood_history,
    safety_crime_information,
    estimated_daily_spending
FROM SEEDED_SUB_DISTRICTS
ON CONFLICT (slug) DO NOTHING;

-- Hand-written places. A temporary table so the bulk places below can copy
-- them.
CREATE TEMPORARY TABLE SEEDED_PLACES (
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
) ON COMMIT DROP AS
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
        ),
        (
            'Masjid Hidayatullah Karet',
            'setiabudi',
            'Jl. Masjid Hidayatullah, Karet Semanggi',
            106.8185,
            -6.2150,
            'Daily, 04:00-22:00',
            NULL,
            'Neighborhood mosque with Friday prayers, open to commuters between shifts.',
            'mosque',
            '{"facility_type": "Masjid", "friday_prayer": true, "women_prayer_area": true}'
        ),
        (
            'Masjid Baitul Mughni',
            'setiabudi',
            'Jl. Jend. Gatot Subroto Kav. 26, Kuningan Timur',
            106.8330,
            -6.2310,
            'Daily, 04:00-22:00',
            NULL,
            'Large mosque along Gatot Subroto, convenient for office workers in Kuningan.',
            'mosque',
            '{"facility_type": "Masjid", "friday_prayer": true, "women_prayer_area": true}'
        ),
        (
            'Masjid Al-Istikmal',
            'mampang-prapatan',
            'Jl. Mampang Prapatan XVI No. 7, Tegal Parang',
            106.8280,
            -6.2478,
            'Daily, 04:00-22:00',
            NULL,
            'Local mosque for the Tegal Parang area with daily prayers and Friday sermons.',
            'mosque',
            '{"facility_type": "Masjid", "friday_prayer": true, "women_prayer_area": true}'
        ),
        (
            'Gereja Kristen Setiabudi Tengah',
            'setiabudi',
            'Jl. Setiabudi Tengah No. 3',
            106.8240,
            -6.2105,
            'Sun, 07:00-19:00',
            NULL,
            'Protestant congregation with Sunday services in the morning and evening.',
            'worship-other',
            '{"religion": "Kristen", "service_schedule": "Minggu 08:00 dan 17:00"}'
        ),
        (
            'Vihara Dharma Mampang',
            'mampang-prapatan',
            'Jl. Mampang Prapatan IV No. 5',
            106.8190,
            -6.2400,
            'Daily, 06:00-18:00',
            NULL,
            'Quiet vihara open daily for prayer and community events.',
            'worship-other',
            '{"religion": "Buddha", "service_schedule": "Setiap hari 06:00-18:00"}'
        ),
        (
            'Polsek Metro Setiabudi',
            'setiabudi',
            'Jl. Taman Setiabudi I No. 1, Karet',
            106.8259,
            -6.2112,
            'Daily, 24 hours',
            NULL,
            'Sector police station for reports, lost documents, and SKCK applications.',
            'police-post',
            '{"facility_type": "Polsek", "open_24h": true, "skck_service": true}'
        ),
        (
            'Polsek Metro Mampang Prapatan',
            'mampang-prapatan',
            'Jl. Kapten Pierre Tendean, Mampang Prapatan',
            106.8186,
            -6.2385,
            'Daily, 24 hours',
            NULL,
            'Sector police station for the Mampang Prapatan area, open around the clock.',
            'police-post',
            '{"facility_type": "Polsek", "open_24h": true, "skck_service": true}'
        ),
        (
            'Pos Polisi Perempatan Mampang',
            'mampang-prapatan',
            'Jl. Mampang Prapatan Raya',
            106.8208,
            -6.2455,
            'Daily, 07:00-21:00',
            NULL,
            'Small police post at the Mampang intersection for traffic and street safety.',
            'police-post',
            '{"facility_type": "Pos polisi", "open_24h": false, "skck_service": false}'
        ),
        (
            'RSUD Mampang Prapatan',
            'mampang-prapatan',
            'Jl. Kapten Pierre Tendean, Mampang Prapatan',
            106.8190,
            -6.2400,
            'Daily, 24 hours',
            NULL,
            'Public district hospital with a 24-hour emergency room that accepts BPJS.',
            'hospital',
            '{"igd_24h": true, "bpjs_accepted": true, "ownership": "Pemerintah", "hospital_class": "D"}'
        ),
        (
            'RS Karet Medika',
            'setiabudi',
            'Jl. Karet Pasar Baru Barat No. 30',
            106.8225,
            -6.2125,
            'Daily, 24 hours',
            'Rp150.000-Rp5.000.000',
            'Private hospital with a 24-hour emergency room and inpatient wards.',
            'hospital',
            '{"igd_24h": true, "bpjs_accepted": true, "ownership": "Swasta", "hospital_class": "C"}'
        ),
        (
            'RS Kuningan Sehat',
            'setiabudi',
            'Jl. H.R. Rasuna Said Kav. 12',
            106.8315,
            -6.2230,
            'Daily, 24 hours',
            'Rp250.000-Rp10.000.000',
            'Private hospital near the Kuningan offices with specialist clinics and a 24-hour emergency room.',
            'hospital',
            '{"igd_24h": true, "bpjs_accepted": false, "ownership": "Swasta", "hospital_class": "B"}'
        ),
        (
            'Puskesmas Kecamatan Setiabudi',
            'setiabudi',
            'Jl. Halimun Raya No. 13, Guntur',
            106.8335,
            -6.2145,
            'Mon-Sat, 07:30-15:00',
            NULL,
            'Government community health center for check-ups, immunization, and basic dental care.',
            'puskesmas',
            '{"open_24h": false, "has_dental": true, "services": "Pemeriksaan umum, imunisasi, KIA, poli gigi"}'
        ),
        (
            'Puskesmas Kecamatan Mampang Prapatan',
            'mampang-prapatan',
            'Jl. Liliana Blok C24-25, Pela Mampang',
            106.8150,
            -6.2520,
            'Mon-Sat, 07:30-15:00',
            NULL,
            'Government community health center for check-ups, immunization, and basic dental care.',
            'puskesmas',
            '{"open_24h": false, "has_dental": true, "services": "Pemeriksaan umum, imunisasi, KIA, poli gigi"}'
        ),
        (
            'ATM Center Karet',
            'setiabudi',
            'Jl. Karet Pasar Baru Timur No. 2',
            106.8268,
            -6.2138,
            'Daily, 24 hours',
            NULL,
            'Cluster of ATMs from several banks, including a cash deposit machine.',
            'bank-atm',
            '{"banks": "BCA, Mandiri, BNI, BRI", "cash_deposit": true, "open_24h": true}'
        ),
        (
            'Gerai ATM Mampang',
            'mampang-prapatan',
            'Jl. Mampang Prapatan Raya No. 55',
            106.8225,
            -6.2490,
            'Daily, 24 hours',
            NULL,
            'Roadside ATM booth for cash withdrawals.',
            'bank-atm',
            '{"banks": "BRI, Mandiri", "cash_deposit": false, "open_24h": true}'
        ),
        (
            'Minimarket Karet 24 Jam',
            'setiabudi',
            'Jl. Karet Pasar Baru Barat No. 4',
            106.8240,
            -6.2118,
            'Daily, 24 hours',
            'Rp3.000-Rp100.000',
            'Chain minimarket open all night for snacks, instant meals, and household basics.',
            'minimarket',
            '{"store_type": "Waralaba", "open_24h": true, "accepts_qris": true, "has_atm": true}'
        ),
        (
            'Toko Kelontong Bu Yati',
            'setiabudi',
            'Jl. Setiabudi Selatan No. 7',
            106.8222,
            -6.2120,
            'Daily, 06:00-22:00',
            'Rp1.000-Rp50.000',
            'Family-run kelontong shop with eggs, rice, cooking oil, and cheap daily goods.',
            'minimarket',
            '{"store_type": "Kelontong", "open_24h": false, "accepts_qris": true, "has_atm": false}'
        ),
        (
            'Minimarket Prapanca Raya',
            'mampang-prapatan',
            'Jl. Prapanca Raya No. 20',
            106.8205,
            -6.2478,
            'Daily, 06:00-23:00',
            'Rp3.000-Rp100.000',
            'Chain minimarket on Prapanca for groceries and everyday supplies.',
            'minimarket',
            '{"store_type": "Waralaba", "open_24h": false, "accepts_qris": true, "has_atm": false}'
        ),
        (
            'Halte TransJakarta Setiabudi Integritas',
            'setiabudi',
            'Jl. H.R. Rasuna Said, Setiabudi',
            106.8298,
            -6.2080,
            'Daily, 05:00-22:00',
            NULL,
            'TransJakarta stop on Rasuna Said for commuters heading to Kuningan and Dukuh Atas.',
            'transit-stop',
            '{"mode": "TransJakarta", "lines": "Koridor 6", "motorcycle_parking": false}'
        ),
        (
            'Stasiun MRT Setiabudi Astra',
            'setiabudi',
            'Jl. Jend. Sudirman, Karet Semanggi',
            106.8200,
            -6.2105,
            'Daily, 05:00-24:00',
            NULL,
            'MRT station on Sudirman connecting Lebak Bulus and Bundaran HI.',
            'transit-stop',
            '{"mode": "MRT", "lines": "Lebak Bulus - Bundaran HI", "motorcycle_parking": false}'
        ),
        (
            'Halte TransJakarta Mampang Prapatan',
            'mampang-prapatan',
            'Jl. Mampang Prapatan Raya',
            106.8257,
            -6.2505,
            'Daily, 05:00-22:00',
            NULL,
            'TransJakarta stop on Mampang Prapatan, served by Koridor 6.',
            'transit-stop',
            '{"mode": "TransJakarta", "lines": "Koridor 6", "motorcycle_parking": false}'
        ),
        (
            'Kantor Kecamatan Setiabudi',
            'setiabudi',
            'Jl. Setiabudi Barat No. 8K',
            106.8236,
            -6.2085,
            'Mon-Fri, 08:00-16:00',
            NULL,
            'District office for residency administration and local government services.',
            'public-service',
            '{"office_type": "Kecamatan", "services": "Administrasi kependudukan, layanan terpadu kecamatan"}'
        ),
        (
            'Kelurahan Karet Kuningan',
            'setiabudi',
            'Karet Kuningan, Setiabudi',
            106.8272,
            -6.2194,
            'Mon-Fri, 08:00-16:00',
            NULL,
            'Village office where newcomers get domicile letters and help with KTP and KK.',
            'public-service',
            '{"office_type": "Kelurahan", "services": "Surat keterangan domisili, pengantar KTP dan KK"}'
        ),
        (
            'Kelurahan Tegal Parang',
            'mampang-prapatan',
            'Jl. Tegal Parang Selatan V, Tegal Parang',
            106.8298,
            -6.2494,
            'Mon-Fri, 08:00-16:00',
            NULL,
            'Village office where newcomers get domicile letters and help with KTP and KK.',
            'public-service',
            '{"office_type": "Kelurahan", "services": "Surat keterangan domisili, pengantar KTP dan KK"}'
        ),
        (
            'Depot Air Isi Ulang Bening',
            'setiabudi',
            'Jl. Karet Sawah No. 3',
            106.8205,
            -6.2160,
            'Daily, 06:00-21:00',
            'Rp5.000-Rp7.000/galon',
            'Refill depot for drinking water gallons, with delivery to nearby buildings.',
            'water-gas',
            '{"service": "Air isi ulang", "delivery": true}'
        ),
        (
            'Galon & Gas Berkah',
            'mampang-prapatan',
            'Jl. Bangka II No. 11',
            106.8095,
            -6.2575,
            'Daily, 07:00-20:00',
            'Rp5.000-Rp25.000',
            'Water gallon refills and 3 kg LPG cylinders, delivered to your door.',
            'water-gas',
            '{"service": "Air & gas", "delivery": true}'
        ),
        (
            'Agen Gas Elpiji Pak Dodi',
            'mampang-prapatan',
            'Jl. Tegal Parang Utara No. 6',
            106.8285,
            -6.2465,
            'Daily, 07:00-19:00',
            'Rp22.000-Rp25.000/tabung',
            'Local agent selling 3 kg LPG cylinders and refills.',
            'water-gas',
            '{"service": "Gas LPG", "delivery": false}'
        ),
        (
            'Fotokopi Karet Jaya',
            'setiabudi',
            'Jl. Karet Belakang No. 14',
            106.8258,
            -6.2095,
            'Mon-Sat, 08:00-20:00',
            'Rp300-Rp3.000/lembar',
            'Photocopy and print shop for CVs, document copies, and passport photos.',
            'print-copy',
            '{"color_print": true, "passport_photo": true, "open_24h": false}'
        ),
        (
            'Print & Copy 24 Jam Mampang',
            'mampang-prapatan',
            'Jl. Mampang Prapatan VIII No. 2',
            106.8180,
            -6.2455,
            'Daily, 24 hours',
            'Rp300-Rp3.000/lembar',
            'Round-the-clock copy and print counter for last-minute documents.',
            'print-copy',
            '{"color_print": true, "passport_photo": false, "open_24h": true}'
        );

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

-- Bulk places copying the hand-written ones under numbered names, spread over the
-- sub-districts (1 in 20 has no location); the fixed seed keeps every database alike.
SELECT setseed(0.42);

WITH BULK_PLACES AS (
    SELECT
        t.*,
        t.name || ' ' || g.n AS bulk_name,
        s.slug AS bulk_sub_district_slug,
        s.longitude AS center_longitude,
        s.latitude AS center_latitude,
        g.n,
        random() AS r1,
        random() AS r2
    FROM generate_series(1, 50000) AS g(n)
    JOIN (
        SELECT sp.*, ROW_NUMBER() OVER (ORDER BY sp.name) - 1 AS idx
        FROM SEEDED_PLACES sp
    ) t ON t.idx = g.n % (SELECT COUNT(*) FROM SEEDED_PLACES)
    -- Steps to the next sub-district after each round of templates, so every
    -- sub-district gets every template.
    JOIN (
        SELECT ssd.*, ROW_NUMBER() OVER (ORDER BY ssd.slug) - 1 AS idx
        FROM SEEDED_SUB_DISTRICTS ssd
    ) s ON s.idx = (g.n / (SELECT COUNT(*) FROM SEEDED_PLACES))
        % (SELECT COUNT(*) FROM SEEDED_SUB_DISTRICTS)
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
    bp.bulk_name,
    bp.address,
    CASE
        WHEN bp.n % 20 = 0 THEN NULL
        ELSE ST_SetSRID(ST_MakePoint(
            bp.center_longitude + (bp.r1 - 0.5) * 0.035,
            bp.center_latitude + (bp.r2 - 0.5) * 0.035
        ), 4326)
    END,
    bp.operating_hours,
    bp.estimated_price_range,
    bp.short_description,
    bp.attributes::jsonb
FROM BULK_PLACES bp
JOIN SUB_DISTRICTS sd ON sd.slug = bp.bulk_sub_district_slug
JOIN CATEGORIES c ON c.slug = bp.category_slug
WHERE NOT EXISTS (
    SELECT 1
    FROM PLACES existing
    WHERE existing.sub_district_id = sd.id
      AND existing.name = bp.bulk_name
)
ORDER BY bp.n;

-- Default admin account. Password: admin123 (change after first sign-in).
INSERT INTO USERS (name, email, password_hash, role)
VALUES (
    'Admin',
    'admin@sakukota.go',
    '$argon2id$v=19$m=19456,t=2,p=1$8BH3b7P0mSW3iajNf9Rv4w$NDJvbJFopj4gJxJqltPEI+QcFoBc0dopSeoKBZyXrIQ',
    'admin'
)
ON CONFLICT (email) DO NOTHING;

-- Demo reviewers. Password for each: sakukota123.
INSERT INTO USERS (name, email, password_hash)
VALUES
    ('Dimas Pratama', 'dimas@sakukota.go', '$argon2id$v=19$m=19456,t=2,p=1$CU8t66QxfAc7yy5IrrFljg$bk0MQ0DdfX4fqeHeErH5aAen0/T43SlpdotFcAldYDE'),
    ('Sari Wulandari', 'sari@sakukota.go', '$argon2id$v=19$m=19456,t=2,p=1$CU8t66QxfAc7yy5IrrFljg$bk0MQ0DdfX4fqeHeErH5aAen0/T43SlpdotFcAldYDE'),
    ('Budi Santoso', 'budi@sakukota.go', '$argon2id$v=19$m=19456,t=2,p=1$CU8t66QxfAc7yy5IrrFljg$bk0MQ0DdfX4fqeHeErH5aAen0/T43SlpdotFcAldYDE')
ON CONFLICT (email) DO NOTHING;

WITH SEEDED_REVIEWS (
    place_name,
    sub_district_slug,
    user_email,
    rating,
    comment
) AS (
    VALUES
        ('Warung Nasi Sari Rasa', 'setiabudi', 'dimas@sakukota.go', 5, 'Generous portions and the prices have stayed the same for months.'),
        ('Warung Nasi Sari Rasa', 'setiabudi', 'sari@sakukota.go', 4, 'Good home-style food. Go before noon, the best dishes sell out.'),
        ('Warung Nasi Sari Rasa', 'setiabudi', 'budi@sakukota.go', 4, NULL),
        ('Kedai Pagi Karet', 'setiabudi', 'dimas@sakukota.go', 3, 'Fine for a quick breakfast, but the queue gets long after 7.'),
        ('Jarum Rapi Setiabudi', 'setiabudi', 'sari@sakukota.go', 5, 'Hemmed two pairs of trousers the same day for a fair price.'),
        ('Busa Laundry 24 Jam', 'setiabudi', 'budi@sakukota.go', 4, 'Clothes come back folded and on time.'),
        ('Busa Laundry 24 Jam', 'setiabudi', 'dimas@sakukota.go', 2, 'Lost a sock twice. Cheap, though.'),
        ('Warung Bu Narti Mampang', 'mampang-prapatan', 'sari@sakukota.go', 5, 'Cheapest proper lunch around here.'),
        ('Apotek Prapanca', 'mampang-prapatan', 'budi@sakukota.go', 4, NULL)
)
INSERT INTO PLACE_REVIEWS (place_id, user_id, rating, comment)
SELECT p.id, u.id, sr.rating, sr.comment
FROM SEEDED_REVIEWS sr
JOIN SUB_DISTRICTS sd ON sd.slug = sr.sub_district_slug
JOIN PLACES p ON p.sub_district_id = sd.id AND p.name = sr.place_name
JOIN USERS u ON u.email = sr.user_email
ON CONFLICT (place_id, user_id) DO NOTHING;

-- Seeded reviews skip the API, which keeps each place's rating totals, so
-- recount them from the stored reviews.
UPDATE PLACES p
SET rating_count = (SELECT COUNT(*) FROM PLACE_REVIEWS r WHERE r.place_id = p.id),
    rating_sum = (SELECT COALESCE(SUM(r.rating), 0) FROM PLACE_REVIEWS r WHERE r.place_id = p.id);

-- Refresh the planner's statistics so the bulk places are planned for at once.
ANALYZE PLACES;
