-- =========================================================================
-- 🗺️ SKEMA TABEL MASTER REGIONAL WILAYAH INDONESIA (PRODUKSI)
-- =========================================================================

-- 1. TABEL MASTER PROVINSI
CREATE TABLE m_provinces (
    id VARCHAR(2) PRIMARY KEY, -- Contoh: '31' untuk DKI Jakarta, '35' untuk Jawa Timur
    name VARCHAR(100) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    updated_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    deleted_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
);

-- 2. TABEL MASTER KOTA / KABUPATEN
CREATE TABLE m_regencies (
    id VARCHAR(4) PRIMARY KEY, -- Contoh: '3171' untuk Kota Jakarta Pusat
    province_id VARCHAR(2) NOT NULL,
    name VARCHAR(100) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    updated_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    deleted_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    
    -- Relasi Kunci Asing (Foreign Key Constraints)
    CONSTRAINT fk_regencies_province FOREIGN KEY (province_id) 
        REFERENCES m_provinces(id) ON DELETE CASCADE ON UPDATE CASCADE
);

-- 3. TABEL MASTER KECAMATAN
CREATE TABLE m_districts (
    id VARCHAR(6) PRIMARY KEY, -- Contoh: '317101' untuk Kecamatan Gambir
    regency_id VARCHAR(4) NOT NULL,
    name VARCHAR(100) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    updated_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    deleted_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    
    -- Relasi Kunci Asing (Foreign Key Constraints)
    CONSTRAINT fk_districts_regency FOREIGN KEY (regency_id) 
        REFERENCES m_regencies(id) ON DELETE CASCADE ON UPDATE CASCADE
);

-- 4. TABEL MASTER KELURAHAN / DESA
CREATE TABLE m_villages (
    id VARCHAR(10) PRIMARY KEY, -- Contoh: '3171011001' untuk Kelurahan Gambir
    district_id VARCHAR(6) NOT NULL,
    name VARCHAR(100) NOT NULL,
    postal_code VARCHAR(5) DEFAULT NULL, -- Kode Pos opsional disatukan di level desa/kelurahan
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    updated_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    deleted_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    
    -- Relasi Kunci Asing (Foreign Key Constraints)
    CONSTRAINT fk_villages_district FOREIGN KEY (district_id) 
        REFERENCES m_districts(id) ON DELETE CASCADE ON UPDATE CASCADE
);

-- =========================================================================
-- ⚡ INDEKS OPTIMASI PENCARIAN (DATABASE PERFORMANCE INDEXING)
-- =========================================================================
-- Menambahkan indeks khusus pada kolom pencarian nama wilayah (ILIKE) dan kolom relasi 
-- agar proses pencarian bertingkat (Cascading Dropdown) berjalan sangat cepat di level milidetik.

CREATE INDEX idx_m_provinces_name ON m_provinces (name);

CREATE INDEX idx_m_regencies_province_id ON m_regencies (province_id);
CREATE INDEX idx_m_regencies_name ON m_regencies (name);

CREATE INDEX idx_m_districts_regency_id ON m_districts (regency_id);
CREATE INDEX idx_m_districts_name ON m_districts (name);

CREATE INDEX idx_m_villages_district_id ON m_villages (district_id);
CREATE INDEX idx_m_villages_name ON m_villages (name);