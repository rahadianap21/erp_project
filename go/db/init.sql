-- schema.sql

CREATE TABLE IF NOT EXISTS s_users (
    id BIGSERIAL PRIMARY KEY,
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    password TEXT NOT NULL,
    role_id INT NOT NULL REFERENCES s_roles(id) ON DELETE RESTRICT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    updated_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    deleted_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL
);

CREATE UNIQUE INDEX idx_users_email_active_unique 
ON s_users (email) 
WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS s_roles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    app VARCHAR(20) NOT NULL DEFAULT 'erp',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    updated_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    deleted_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS s_permissions (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    updated_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    deleted_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_id INT REFERENCES s_roles(id) ON DELETE CASCADE,
    permission_id INT REFERENCES s_permissions(id) ON DELETE CASCADE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    updated_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    deleted_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    PRIMARY KEY (role_id, permission_id)
);

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
    deleted_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL
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

-- 1. Hapus constraint atau indeks unik lama pada email jika ada (sesuaikan dengan nama constraint Anda)
ALTER TABLE s_users DROP CONSTRAINT IF EXISTS s_users_email_key;
DROP INDEX IF EXISTS idx_users_email_unique;

-- 2. 🌟 KUNCI UTAMA: Buat Partial Unique Index baru
-- Indeks ini hanya mendeteksi duplikat JIKA data tersebut belum dihapus (deleted_at IS NULL)
-- Jika data lamanya sudah di-soft delete (deleted_at IS NOT NULL), database mengizinkan baris baru masuk!
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_active_unique 
ON s_users (email) 
WHERE deleted_at IS NULL;

-- Ditaruh di baris paling bawah berkas init.sql Anda

CREATE TABLE IF NOT EXISTS m_suppliers (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    pic_name VARCHAR(150) NOT NULL,
    phone VARCHAR(20) NOT NULL,
    email VARCHAR(150) NOT NULL,
    address TEXT NOT NULL,
    village_id VARCHAR(10) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    updated_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    deleted_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,

    CONSTRAINT fk_suppliers_village FOREIGN KEY (village_id) 
        REFERENCES m_villages(id) ON DELETE RESTRICT ON UPDATE CASCADE
);

CREATE INDEX idx_m_suppliers_name ON m_suppliers (name);
CREATE INDEX idx_m_suppliers_village_id ON m_suppliers (village_id);
CREATE UNIQUE INDEX idx_suppliers_email_active_unique ON m_suppliers (email) WHERE deleted_at IS NULL;

-- =========================================================================
-- 🏦 SKEMA TABEL MASTER BANK NASIONAL & INTERNASIONAL (PRODUKSI)
-- =========================================================================

CREATE TABLE IF NOT EXISTS m_banks (
    id SERIAL PRIMARY KEY,                             -- 🌟 KUNCI UTAMA: Angka berurutan otomatis
    kode VARCHAR(5) NOT NULL,                          -- 🌟 KODE BANK BI: Contoh: '014', '008', '009'
    name VARCHAR(150) NOT NULL,                        -- Nama Resmi Bank
    short_name VARCHAR(50) NOT NULL,                  -- Nama Singkat (e.g., BCA, MANDIRI)
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    updated_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    deleted_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL
);

-- =========================================================================
-- ⚡ INDEKS OPTIMASI & KEKANG UNIK (PERFORMANCE INDEXING)
-- =========================================================================
-- Menjamin tidak ada kode bank atau nama bank yang ganda di dalam sistem
CREATE UNIQUE INDEX idx_banks_kode_active_unique ON m_banks (kode) WHERE deleted_at IS NULL;
CREATE INDEX idx_m_banks_name ON m_banks (name);
CREATE INDEX idx_m_banks_short_name ON m_banks (short_name);

-- 1. Tambahkan kolom finansial baru ke dalam tabel m_suppliers
ALTER TABLE m_suppliers ADD COLUMN IF NOT EXISTS bank_id INT DEFAULT NULL;
ALTER TABLE m_suppliers ADD COLUMN IF NOT EXISTS account_number VARCHAR(50) DEFAULT NULL;
ALTER TABLE m_suppliers ADD COLUMN IF NOT EXISTS account_name VARCHAR(100) DEFAULT NULL;

-- 2. 🌟 KUNCI INTEGRITAS: Pasang constraint Foreign Key yang mengikat ke tabel m_banks
ALTER TABLE m_suppliers 
ADD CONSTRAINT fk_suppliers_bank FOREIGN KEY (bank_id) 
REFERENCES m_banks(id) ON DELETE RESTRICT ON UPDATE CASCADE;

-- 3. Tambahkan indeks optimasi untuk mempercepat kueri laporan keuangan masa depan
CREATE INDEX IF NOT EXISTS idx_m_suppliers_bank_id ON m_suppliers (bank_id);

-- =========================================================================
-- 🗂️ SKEMA TABEL MASTER KATEGORI PRODUK (HIERARKI BERTINGKAT)
-- =========================================================================

CREATE TABLE IF NOT EXISTS m_product_categories (
    id SERIAL PRIMARY KEY,
    name VARCHAR(150) NOT NULL,
    slug VARCHAR(150) NOT NULL,                        -- Untuk kebutuhan URL ramah atau kode unik kategori
    description TEXT,
    
    -- 🌲 SELF-REFERENCING FOR SUB-CATEGORIES
    -- Jika parent_id diisi ID kategori lain, maka baris ini otomatis menjadi Sub-Kategori
    parent_id INT DEFAULT NULL,
    
    -- INDIKATOR STATUS & JEJAK AUDIT (AUDIT TRAIL)
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    updated_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    deleted_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,

    -- RELASI KUNCI ASING KE TABEL SENDIRI
    CONSTRAINT fk_categories_parent FOREIGN KEY (parent_id) 
        REFERENCES m_product_categories(id) ON DELETE RESTRICT ON UPDATE CASCADE
);

-- =========================================================================
-- ⚡ INDEKS OPTIMASI PENCARIAN (PERFORMANCE INDEXING)
-- =========================================================================
CREATE UNIQUE INDEX idx_categories_slug_active_unique ON m_product_categories (slug) WHERE deleted_at IS NULL;
CREATE INDEX idx_m_product_categories_parent_id ON m_product_categories (parent_id);

-- =========================================================================
-- 📦 SKEMA TABEL MASTER PRODUK / BARANG (ENTERPRISE PRODUCTION GRADE)
-- =========================================================================

CREATE TABLE IF NOT EXISTS m_products (
    id BIGSERIAL PRIMARY KEY,
    sku VARCHAR(50) NOT NULL,                          -- Stock Keeping Unit (Kode Unik Internal Barang)
    barcode VARCHAR(100),                              -- Kode Barcode Internasional (EAN-13 / UPC jika ada)
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- 📊 RELASI MASTER DATA (FOREIGN KEYS)
    category_id INT NOT NULL,                          -- Mengikat ke m_product_categories
    supplier_id BIGINT NOT NULL,                       -- Mengikat ke m_suppliers (Vendor Utama)
    
    -- 💵 METRIKA AKUNTANSI & INVENTARIS
    unit_of_measure VARCHAR(20) NOT NULL DEFAULT 'PCS',-- Satuan Barang. Contoh: 'PCS', 'BOX', 'KILOGRAM', 'LITER'
    purchase_price NUMERIC(15, 2) NOT NULL DEFAULT 0,  -- Harga beli modal dari supplier (HPP dasar)
    selling_price NUMERIC(15, 2) NOT NULL DEFAULT 0,   -- Harga jual standar ke pelanggan/distributor
    min_stock_level INT NOT NULL DEFAULT 5,            -- Batas stok minimum untuk memicu alert re-order otomatis
    discount_type VARCHAR(10) NOT NULL DEFAULT 'NONE',
    discount_value NUMERIC(15,2) NOT NULL DEFAULT 0,
    is_taxable BOOLEAN NOT NULL DEFAULT TRUE,          -- Apakah produk kena PPN saat dijual

    -- INDIKATOR STATUS & JEJAK AUDIT (AUDIT TRAIL)
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    updated_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    deleted_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,

    -- RELASI KUNCI ASING (INTEGRITY CONSTRAINTS)
    CONSTRAINT fk_products_category FOREIGN KEY (category_id) 
        REFERENCES m_product_categories(id) ON DELETE RESTRICT ON UPDATE CASCADE,
        
    CONSTRAINT fk_products_supplier FOREIGN KEY (supplier_id) 
        REFERENCES m_suppliers(id) ON DELETE RESTRICT ON UPDATE CASCADE,
    CONSTRAINT m_products_discount_type_check CHECK (discount_type IN ('NONE', 'PERCENT', 'AMOUNT')),
    CONSTRAINT m_products_discount_value_non_neg_check CHECK (discount_value >= 0),
    CONSTRAINT m_products_discount_pct_check CHECK (discount_type != 'PERCENT' OR (discount_value >= 0 AND discount_value <= 100))
);

-- =========================================================================
-- ⚡ INDEKS OPTIMASI PENCARIAN & VALISASI UNIK (PERFORMANCE INDEXING)
-- =========================================================================
-- Menjamin tidak ada kode SKU yang kembar di dalam sistem untuk produk aktif
CREATE UNIQUE INDEX idx_products_sku_active_unique ON m_products (sku) WHERE deleted_at IS NULL;

-- Mempercepat kueri filtering pencarian barang di gudang berskala besar
CREATE INDEX idx_m_products_name ON m_products (name);
CREATE INDEX idx_m_products_category_id ON m_products (category_id);
CREATE INDEX idx_m_products_supplier_id ON m_products (supplier_id);

CREATE TABLE IF NOT EXISTS m_units (
    id SERIAL PRIMARY KEY,                             -- Kunci utama angka otomatis
    kode VARCHAR(20) NOT NULL,                         -- Kode Singkat Satuan. Contoh: 'PCS', 'BOX', 'KG', 'LITER'
    name VARCHAR(100) NOT NULL,                        -- Nama Panjang Satuan. Contoh: 'PIECES', 'KARTON', 'KILOGRAM'
    description TEXT,                                  -- Catatan penjelasan tambahan
    
    -- INDIKATOR STATUS & JEJAK AUDIT (AUDIT TRAIL)
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    updated_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    deleted_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL
);

-- =========================================================================
-- ⚡ INDEKS OPTIMASI & KEKANG UNIK (PERFORMANCE INDEXING)
-- =========================================================================
-- Menjamin tidak ada kode satuan yang ganda di dalam sistem untuk data aktif
CREATE UNIQUE INDEX idx_units_kode_active_unique ON m_units (kode) WHERE deleted_at IS NULL;
CREATE INDEX idx_m_units_name ON m_units (name);

-- 1. Tambahkan kolom relasi baru unit_id ke tabel m_products
ALTER TABLE m_products ADD COLUMN IF NOT EXISTS unit_id INT DEFAULT NULL;

-- 2. Migrasikan data teks lama ke ID tabel master baru secara otomatis (Data Seeder Safety)
UPDATE m_products p 
SET unit_id = u.id 
FROM m_units u 
WHERE p.unit_of_measure = u.kode;

-- 3. Set kolom unit_id menjadi NOT NULL setelah data termigrasi (Aturan Kaku)
ALTER TABLE m_products ALTER COLUMN unit_id SET NOT NULL;

-- 4. Hapus kolom teks usang unit_of_measure yang sudah tidak digunakan
ALTER TABLE m_products DROP COLUMN IF EXISTS unit_of_measure;

-- 5. 🌟 KUNCI INTEGRITAS: Pasang constraint Foreign Key yang mengikat ke tabel m_units
ALTER TABLE m_products 
ADD CONSTRAINT fk_products_unit FOREIGN KEY (unit_id) 
REFERENCES m_units(id) ON DELETE RESTRICT ON UPDATE CASCADE;

-- 6. Pasang indeks optimasi performa kueri
CREATE INDEX IF NOT EXISTS idx_m_products_unit_id ON m_products (unit_id);

-- Tambahkan kolom current_stock dengan nilai default awal 0
ALTER TABLE m_products ADD COLUMN IF NOT EXISTS current_stock INT NOT NULL DEFAULT 0;

-- Tambahkan kekang check otomatis di level basis data agar stok tidak pernah bisa bernilai minus (< 0)
ALTER TABLE m_products ADD CONSTRAINT chk_products_current_stock_non_negative CHECK (current_stock >= 0);

-- Buat indeks optimasi untuk mempermudah kueri monitoring stok kritis/habis masa depan
CREATE INDEX IF NOT EXISTS idx_m_products_current_stock ON m_products (current_stock);

-- 1. Tambahkan dua kolom relasi baru untuk UoM Jual dan UoM Beli
ALTER TABLE m_products ADD COLUMN IF NOT EXISTS selling_unit_id INT DEFAULT NULL;
ALTER TABLE m_products ADD COLUMN IF NOT EXISTS purchase_unit_id INT DEFAULT NULL;

-- 2. Migrasikan data unit_id lama ke kedua kolom baru agar aman (Data Seeder Safety)
UPDATE m_products SET selling_unit_id = unit_id, purchase_unit_id = unit_id;

-- 3. Set kolom baru menjadi NOT NULL setelah data berhasil disalin murni
ALTER TABLE m_products ALTER COLUMN selling_unit_id SET NOT NULL;
ALTER TABLE m_products ALTER COLUMN purchase_unit_id SET NOT NULL;

-- 4. Hapus kolom tunggal unit_id lama yang sudah usang
ALTER TABLE m_products DROP COLUMN IF EXISTS unit_id;

-- 5. 🔒 KUNCI INTEGRITAS RELASIONAL: Pasang Foreign Key ganda yang mengikat ke tabel m_units
ALTER TABLE m_products 
ADD CONSTRAINT fk_products_selling_unit FOREIGN KEY (selling_unit_id) 
REFERENCES m_units(id) ON DELETE RESTRICT ON UPDATE CASCADE;

ALTER TABLE m_products 
ADD CONSTRAINT fk_products_purchase_unit FOREIGN KEY (purchase_unit_id) 
REFERENCES m_units(id) ON DELETE RESTRICT ON UPDATE CASCADE;

-- 6. Pasang indeks optimasi performa kueri
CREATE INDEX IF NOT EXISTS idx_m_products_selling_unit_id ON m_products (selling_unit_id);
CREATE INDEX IF NOT EXISTS idx_m_products_purchase_unit_id ON m_products (purchase_unit_id);

-- 7. 🌟 FAKTOR KONVERSI SATUAN
-- current_stock dinyatakan dalam SATUAN JUAL (satuan terkecil). Satu satuan beli
-- berisi conversion_rate satuan jual, jadi penerimaan barang bisa menaikkan
-- stok dalam satuan yang sama dengan yang dipakai min_stock_level.
-- Nilai 1 berarti tidak ada konversi, yaitu satuan beli sama dengan satuan jual.
-- Default 1 dipilih karena faktor yang sebenarnya tidak bisa dikarang sistem:
-- master produk yang satuan belinya masih salah harus diperbaiki lewat form.
ALTER TABLE m_products ADD COLUMN IF NOT EXISTS conversion_rate INT NOT NULL DEFAULT 1;

ALTER TABLE m_products DROP CONSTRAINT IF EXISTS m_products_conversion_rate_check;

ALTER TABLE m_products
ADD CONSTRAINT m_products_conversion_rate_check CHECK (conversion_rate >= 1);

-- 8. 💵 HARGA POKOK RATA-RATA (MOVING AVERAGE)
-- Harga pokok SATUAN JUAL hasil penimbangan seluruh penerimaan barang.
-- Dihitung ulang setiap penerimaan ditutup, memakai nilai DPP barang yang masuk
-- supaya selaras dengan saldo buku besar Persediaan.
-- NUMERIC(15,4) karena hasil bagi per satuan umumnya lebih dari dua desimal;
-- membulatkannya ke dua desimal akan menumpuk kesalahan pada kuantitas besar.
-- Ditulis hanya oleh modul penerimaan barang, sama seperti current_stock.
ALTER TABLE m_products ADD COLUMN IF NOT EXISTS avg_cost NUMERIC(15,4) NOT NULL DEFAULT 0;

ALTER TABLE m_products DROP CONSTRAINT IF EXISTS m_products_avg_cost_check;

ALTER TABLE m_products
ADD CONSTRAINT m_products_avg_cost_check CHECK (avg_cost >= 0);

-- 🌟 1. TABEL HEADER PURCHASE ORDER
CREATE TABLE IF NOT EXISTS t_purchase_orders (
    id BIGSERIAL PRIMARY KEY,
    po_number VARCHAR(50) NOT NULL UNIQUE,          -- Format Dokumen: PO/YYYYMMDD/0001
    supplier_id BIGINT NOT NULL,                    -- Relasi ke master supplier
    po_date DATE NOT NULL DEFAULT CURRENT_DATE,     -- Tanggal terbit dokumen
    expected_delivery_date DATE DEFAULT NULL,       -- Estimasi kedatangan barang
    total_amount NUMERIC(15,2) NOT NULL DEFAULT 0,  -- Akumulasi finansial PO (HPP * Qty)
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',  -- Status: PENDING, APPROVED, RECEIVED, REJECTED
    notes TEXT,                                     -- Catatan instruksi pengiriman
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_by BIGINT DEFAULT NULL,
    updated_by BIGINT DEFAULT NULL,
    deleted_at TIMESTAMP DEFAULT NULL,
    approved_by BIGINT DEFAULT NULL,
    approved_at TIMESTAMP DEFAULT NULL,
    rejected_by BIGINT DEFAULT NULL,
    rejected_at TIMESTAMP DEFAULT NULL,
    reject_reason TEXT DEFAULT NULL,

    
    CONSTRAINT fk_po_supplier FOREIGN KEY (supplier_id) REFERENCES m_suppliers(id) ON DELETE RESTRICT,
    CONSTRAINT fk_po_approved_by FOREIGN KEY (approved_by) REFERENCES s_users(id) ON DELETE SET NULL,
    CONSTRAINT fk_po_rejected_by FOREIGN KEY (rejected_by) REFERENCES s_users(id) ON DELETE SET NULL

);

-- 🌟 2. TABEL BARIS DETAIL ITEM DOKUMEN PO (ONE-TO-MANY)
CREATE TABLE IF NOT EXISTS t_purchase_order_details (
    id BIGSERIAL PRIMARY KEY,
    purchase_order_id BIGINT NOT NULL,              -- Link foreign key ke header PO
    product_id BIGINT NOT NULL,                     -- Relasi ke master produk/barang
    qty_ordered INT NOT NULL CHECK (qty_ordered > 0),-- Jumlah kuantitas yang dipesan ke vendor
    qty_received INT NOT NULL DEFAULT 0,            -- Jumlah riil yang sudah diterima di gudang (GRN)
    
    -- 🌟 INTEGRASI DUAL SATUAN: Pembelian barang dari supplier wajib memakai unit grosir / Satuan Beli
    purchase_unit_id INT NOT NULL, 
    
    unit_price NUMERIC(15,2) NOT NULL DEFAULT 0,    -- Harga beli modal satuan (HPP) saat PO dikunci
    sub_total NUMERIC(15,2) NOT NULL DEFAULT 0,     -- Perhitungan otomatis: qty_ordered * unit_price
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    CONSTRAINT fk_po_detail_header FOREIGN KEY (purchase_order_id) REFERENCES t_purchase_orders(id) ON DELETE CASCADE,
    CONSTRAINT fk_po_detail_product FOREIGN KEY (product_id) REFERENCES m_products(id) ON DELETE RESTRICT,
    CONSTRAINT fk_po_detail_unit FOREIGN KEY (purchase_unit_id) REFERENCES m_units(id) ON DELETE RESTRICT
);

-- =========================================================================
-- PENGHITUNG NOMOR DOKUMEN PO
-- =========================================================================
-- Satu baris per prefiks tanggal (format YYYYMMDD). Penomoran PO memakai
-- UPSERT atomik terhadap tabel ini, bukan COUNT(*)+1 terhadap tabel PO, karena
-- penghitung berbasis baris PO bermasalah pada tiga hal: balapan saat dua
-- permintaan bersamaan, nomor dipakai ulang setelah soft-delete, dan pencocokan
-- yang ikut menghitung nama supplier yang kebetulan memuat prefiks hari itu.
--
-- last_seq sengaja tidak dihitung ulang dari tabel PO. Kalau dihitung,
-- soft-delete atau arsip akan membuat nomor mundur dan nomor yang sudah pernah
-- terbit bisa dipakai lagi.
CREATE TABLE IF NOT EXISTS t_po_number_counters (
    prefix   VARCHAR(8) PRIMARY KEY,
    last_seq BIGINT NOT NULL DEFAULT 0,
    CONSTRAINT t_po_number_counters_prefix_check CHECK (prefix ~ '^[0-9]{8}$'),
    CONSTRAINT t_po_number_counters_seq_check    CHECK (last_seq >= 0)
);

-- 🔒 PASANG INDEKS AKSELERASI MONITORING TRANSAKSI VITAL
CREATE INDEX IF NOT EXISTS idx_t_purchase_orders_number ON t_purchase_orders (po_number);
CREATE INDEX IF NOT EXISTS idx_t_purchase_orders_status ON t_purchase_orders (status);
CREATE INDEX IF NOT EXISTS idx_t_purchase_order_details_po ON t_purchase_order_details (purchase_order_id);

-- 1. Tambahkan kolom kebijakan pajak default pada Master Supplier
ALTER TABLE m_suppliers ADD COLUMN IF NOT EXISTS default_tax_type VARCHAR(20) NOT NULL DEFAULT 'NON_PPN';
-- Aturan Nilai Konvensi: 'NON_PPN', 'TAX_EXCLUSIVE', 'TAX_INCLUSIVE'

-- 2. Tambahkan kolom kalkulasi finansial pajak pada Transaksi Header PO
ALTER TABLE t_purchase_orders ADD COLUMN IF NOT EXISTS tax_type VARCHAR(20) NOT NULL DEFAULT 'NON_PPN';
ALTER TABLE t_purchase_orders ADD COLUMN IF NOT EXISTS dpp_amount NUMERIC(15,2) NOT NULL DEFAULT 0;
ALTER TABLE t_purchase_orders ADD COLUMN IF NOT EXISTS tax_amount NUMERIC(15,2) NOT NULL DEFAULT 0;

-- 3. Pasang indeks optimasi kueri laporan keuangan akuntansi
CREATE INDEX IF NOT EXISTS idx_t_purchase_orders_tax_type ON t_purchase_orders (tax_type);

-- =========================================================================
-- MODUL PENERIMAAN BARANG (GOODS RECEIPT NOTES / GRN)
-- =========================================================================
--
-- Mencatat penerimaan barang dari PO berstatus APPROVED. Satu PO boleh
-- menerima beberapa batch pada tanggal/penerima berbeda, sehingga header dan
-- detail dipisah seperti PO. Batch bersifat immutable pada v1: pembatalan
-- GRN akan memerlukan pembalikan stok yang belum ada modulnya.
--
-- current_stock dinyatakan dalam SATUAN JUAL (satuan terkecil). Qty yang
-- diterima dicatat dalam satuan beli, lalu dikalikan conversion_rate produk
-- untuk memperoleh delta stok dalam satuan jual. conversion_rate disimpan
-- sebagai snapshot di detail agar perubahan master produk setelah PO tidak
-- mengubah riwayat penerimaan.

-- 1. Header GRN
CREATE TABLE IF NOT EXISTS t_goods_receipt_notes (
    id BIGSERIAL PRIMARY KEY,
    grn_number VARCHAR(50) NOT NULL UNIQUE,
    purchase_order_id BIGINT NOT NULL,
    received_date DATE NOT NULL DEFAULT CURRENT_DATE,
    total_qty INT NOT NULL DEFAULT 0,
    notes TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_by BIGINT DEFAULT NULL,
    updated_by BIGINT DEFAULT NULL,
    CONSTRAINT fk_grn_purchase_order FOREIGN KEY (purchase_order_id)
        REFERENCES t_purchase_orders(id) ON DELETE RESTRICT,
    CONSTRAINT t_goods_receipt_notes_grn_number_check CHECK (grn_number ~ '^(GRN|grn)/[0-9]{8}/[0-9]{4}$'),
    CONSTRAINT t_goods_receipt_notes_total_qty_check CHECK (total_qty >= 0)
);

CREATE INDEX IF NOT EXISTS idx_t_grn_purchase_order_id ON t_goods_receipt_notes (purchase_order_id);
CREATE INDEX IF NOT EXISTS idx_t_grn_received_date ON t_goods_receipt_notes (received_date);
CREATE INDEX IF NOT EXISTS idx_t_grn_grn_number ON t_goods_receipt_notes (grn_number);

-- 2. Detail GRN
CREATE TABLE IF NOT EXISTS t_goods_receipt_note_details (
    id BIGSERIAL PRIMARY KEY,
    grn_id BIGINT NOT NULL,
    purchase_order_detail_id BIGINT NOT NULL,
    product_id BIGINT NOT NULL,
    qty_received INT NOT NULL,
    conversion_rate INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_grn_detail_header FOREIGN KEY (grn_id)
        REFERENCES t_goods_receipt_notes(id) ON DELETE CASCADE,
    CONSTRAINT fk_grn_detail_po_detail FOREIGN KEY (purchase_order_detail_id)
        REFERENCES t_purchase_order_details(id) ON DELETE RESTRICT,
    CONSTRAINT fk_grn_detail_product FOREIGN KEY (product_id)
        REFERENCES m_products(id) ON DELETE RESTRICT,
    CONSTRAINT t_grn_detail_qty_received_check CHECK (qty_received > 0),
    CONSTRAINT t_grn_detail_conversion_rate_check CHECK (conversion_rate >= 1)
);

CREATE INDEX IF NOT EXISTS idx_t_grn_detail_grn_id ON t_goods_receipt_note_details (grn_id);
CREATE INDEX IF NOT EXISTS idx_t_grn_detail_po_detail_id ON t_goods_receipt_note_details (purchase_order_detail_id);
CREATE INDEX IF NOT EXISTS idx_t_grn_detail_product_id ON t_goods_receipt_note_details (product_id);

-- 3. Penghitung nomor GRN (satu baris per prefiks tanggal YYYYMMDD)
CREATE TABLE IF NOT EXISTS t_grn_number_counters (
    prefix VARCHAR(8) PRIMARY KEY,
    last_seq BIGINT NOT NULL DEFAULT 0,
    CONSTRAINT t_grn_number_counters_prefix_check CHECK (prefix ~ '^[0-9]{8}$'),
    CONSTRAINT t_grn_number_counters_seq_check CHECK (last_seq >= 0)
);

-- =========================================================================
-- MODUL AKUNTANSI (FASE 1): BAGAN AKUN, PERIODE FISKAL, JURNAL UMUM
-- =========================================================================
--
-- Konsisten dengan 000009_modul_akuntansi.up.sql. DDL di sini hanya sumber
-- skema untuk sqlc; seed CoA/periode/permission hidup di migrasi.

-- 1. Bagan Akun (Chart of Accounts)
CREATE TABLE IF NOT EXISTS m_chart_of_accounts (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(20) NOT NULL,
    name VARCHAR(150) NOT NULL,
    account_type VARCHAR(20) NOT NULL DEFAULT 'ASSET'
        CHECK (account_type IN ('ASSET', 'LIABILITY', 'EQUITY', 'REVENUE', 'EXPENSE')),
    normal_balance VARCHAR(6) NOT NULL DEFAULT 'DEBIT'
        CHECK (normal_balance IN ('DEBIT', 'CREDIT')),
    parent_id BIGINT DEFAULT NULL,
    is_header BOOLEAN NOT NULL DEFAULT FALSE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    updated_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    deleted_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    CONSTRAINT fk_coa_parent FOREIGN KEY (parent_id)
        REFERENCES m_chart_of_accounts(id) ON DELETE RESTRICT,
    CONSTRAINT m_chart_of_accounts_balance_check CHECK (
        (account_type IN ('ASSET', 'EXPENSE') AND normal_balance = 'DEBIT')
        OR (account_type IN ('LIABILITY', 'EQUITY', 'REVENUE') AND normal_balance = 'CREDIT')
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_coa_code_active_unique
    ON m_chart_of_accounts (code) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_m_coa_parent_id ON m_chart_of_accounts (parent_id);
CREATE INDEX IF NOT EXISTS idx_m_coa_type ON m_chart_of_accounts (account_type);

-- 2. Periode fiskal
CREATE TABLE IF NOT EXISTS m_accounting_periods (
    id BIGSERIAL PRIMARY KEY,
    period_year INT NOT NULL,
    period_month INT NOT NULL,
    status VARCHAR(10) NOT NULL DEFAULT 'OPEN'
        CHECK (status IN ('OPEN', 'CLOSED')),
    closed_at TIMESTAMP WITH TIME ZONE,
    closed_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    updated_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    CONSTRAINT m_accounting_periods_period_unique UNIQUE (period_year, period_month),
    CONSTRAINT m_accounting_periods_month_check CHECK (period_month BETWEEN 1 AND 12),
    CONSTRAINT m_accounting_periods_year_check CHECK (period_year BETWEEN 2000 AND 2100)
);

-- 3. Pemetaan akun untuk auto-posting
CREATE TABLE IF NOT EXISTS s_accounting_settings (
    mapping_key VARCHAR(30) PRIMARY KEY,
    account_id BIGINT NOT NULL,
    description VARCHAR(200) DEFAULT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    CONSTRAINT fk_acc_settings_account FOREIGN KEY (account_id)
        REFERENCES m_chart_of_accounts(id) ON DELETE RESTRICT,
    CONSTRAINT s_accounting_settings_key_check CHECK (mapping_key IN (
        'INVENTORY', 'AP', 'AR', 'COGS', 'SALES_REVENUE',
        'VAT_INPUT', 'VAT_OUTPUT', 'CASH', 'RETAINED_EARNINGS'
    ))
);

-- 3b. Pengaturan aplikasi umum (key-value sederhana).
CREATE TABLE IF NOT EXISTS s_app_settings (
    key VARCHAR(100) PRIMARY KEY,
    value TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL
);

-- 4. Jurnal umum (header)
CREATE TABLE IF NOT EXISTS t_journal_entries (
    id BIGSERIAL PRIMARY KEY,
    entry_number VARCHAR(50) NOT NULL UNIQUE,
    entry_date DATE NOT NULL DEFAULT CURRENT_DATE,
    description TEXT,
    source_type VARCHAR(30) DEFAULT NULL
        CHECK (source_type IN ('MANUAL', 'GRN', 'SUPPLIER_PAYMENT', 'CASH_IN', 'CASH_OUT', 'CLOSING', 'SALE', 'SALE_PAYMENT')),
    source_id BIGINT DEFAULT NULL,
    source_reference VARCHAR(50) DEFAULT NULL,
    status VARCHAR(10) NOT NULL DEFAULT 'DRAFT'
        CHECK (status IN ('DRAFT', 'POSTED', 'VOID')),
    total_debit NUMERIC(15,2) NOT NULL DEFAULT 0,
    total_credit NUMERIC(15,2) NOT NULL DEFAULT 0,
    period_year INT NOT NULL,
    period_month INT NOT NULL DEFAULT 1,
    posted_at TIMESTAMP DEFAULT NULL,
    posted_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    voided_at TIMESTAMP DEFAULT NULL,
    voided_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    void_reason TEXT,
    reversal_of_id BIGINT REFERENCES t_journal_entries(id) ON DELETE RESTRICT,
    reversed_by_id BIGINT REFERENCES t_journal_entries(id) ON DELETE SET NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_by BIGINT DEFAULT NULL,
    updated_by BIGINT DEFAULT NULL,
    CONSTRAINT t_journal_entries_totals_check CHECK (total_debit = total_credit),
    CONSTRAINT t_journal_entries_period_month_check CHECK (period_month BETWEEN 1 AND 12),
    CONSTRAINT t_journal_entries_period_year_check CHECK (period_year BETWEEN 2000 AND 2100)
);

CREATE INDEX IF NOT EXISTS idx_t_journal_entries_date ON t_journal_entries (entry_date);
CREATE INDEX IF NOT EXISTS idx_t_journal_entries_period ON t_journal_entries (period_year, period_month);
CREATE INDEX IF NOT EXISTS idx_t_journal_entries_source ON t_journal_entries (source_type, source_id);
CREATE INDEX IF NOT EXISTS idx_t_journal_entries_status ON t_journal_entries (status);

CREATE UNIQUE INDEX IF NOT EXISTS idx_journal_source_unique
    ON t_journal_entries (source_type, source_id)
    WHERE source_type IS NOT NULL
      AND source_id IS NOT NULL
      AND status = 'POSTED'
      AND reversed_by_id IS NULL;

-- 5. Jurnal umum (lines)
CREATE TABLE IF NOT EXISTS t_journal_entry_lines (
    id BIGSERIAL PRIMARY KEY,
    journal_entry_id BIGINT NOT NULL,
    line_no INT NOT NULL,
    account_id BIGINT NOT NULL,
    description TEXT,
    debit NUMERIC(15,2) NOT NULL DEFAULT 0,
    credit NUMERIC(15,2) NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_jel_journal FOREIGN KEY (journal_entry_id)
        REFERENCES t_journal_entries(id) ON DELETE CASCADE,
    CONSTRAINT fk_jel_account FOREIGN KEY (account_id)
        REFERENCES m_chart_of_accounts(id) ON DELETE RESTRICT,
    CONSTRAINT t_journal_entry_lines_single_side_check
        CHECK (debit >= 0 AND credit >= 0 AND NOT (debit > 0 AND credit > 0)),
    CONSTRAINT t_journal_entry_lines_has_amount_check
        CHECK (debit > 0 OR credit > 0),
    CONSTRAINT t_journal_entry_lines_line_no_unique UNIQUE (journal_entry_id, line_no)
);

CREATE INDEX IF NOT EXISTS idx_t_journal_entry_lines_entry ON t_journal_entry_lines (journal_entry_id);
CREATE INDEX IF NOT EXISTS idx_t_journal_entry_lines_account ON t_journal_entry_lines (account_id);

-- 6. Penghitung nomor jurnal
CREATE TABLE IF NOT EXISTS t_journal_number_counters (
    prefix VARCHAR(8) PRIMARY KEY,
    last_seq BIGINT NOT NULL DEFAULT 0,
    CONSTRAINT t_journal_number_counters_prefix_check CHECK (prefix ~ '^[0-9]{8}$'),
    CONSTRAINT t_journal_number_counters_seq_check CHECK (last_seq >= 0)
);

-- =========================================================================
-- MODUL PENJUALAN / KASIR (POS)
-- =========================================================================

CREATE TABLE IF NOT EXISTS m_members (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(30) NOT NULL,
    name VARCHAR(150) NOT NULL,
    phone VARCHAR(30) NOT NULL DEFAULT '',
    email VARCHAR(150) NOT NULL DEFAULT '',
    address TEXT NOT NULL DEFAULT '',
    default_term_days INT NOT NULL DEFAULT 0,
    points BIGINT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    updated_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    deleted_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    CONSTRAINT m_members_default_term_days_check CHECK (default_term_days >= 0),
    CONSTRAINT m_members_points_non_neg CHECK (points >= 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_members_code_active_unique
    ON m_members (code) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_m_members_name ON m_members (name);

CREATE TABLE IF NOT EXISTS m_payment_methods (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(30) NOT NULL,
    name VARCHAR(100) NOT NULL,
    channel VARCHAR(20) NOT NULL DEFAULT 'CASH'
        CHECK (channel IN ('CASH', 'BANK', 'EWALLET', 'CREDIT')),
    account_id BIGINT REFERENCES m_chart_of_accounts(id) ON DELETE RESTRICT,
    is_cash BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    updated_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    deleted_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_payment_methods_code_active_unique
    ON m_payment_methods (code) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS m_vouchers (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(50) NOT NULL,
    name VARCHAR(150) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    discount_type VARCHAR(10) NOT NULL DEFAULT 'AMOUNT'
        CHECK (discount_type IN ('PERCENT', 'AMOUNT')),
    discount_value NUMERIC(15,2) NOT NULL DEFAULT 0
        CHECK (discount_value >= 0),
    min_purchase NUMERIC(15,2) NOT NULL DEFAULT 0
        CHECK (min_purchase >= 0),
    max_discount NUMERIC(15,2) NOT NULL DEFAULT 0
        CHECK (max_discount >= 0),
    valid_from DATE,
    valid_until DATE,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    used_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    updated_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    deleted_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    CONSTRAINT m_vouchers_percent_check
        CHECK (discount_type != 'PERCENT' OR (discount_value >= 0 AND discount_value <= 100)),
    CONSTRAINT m_vouchers_valid_period_check
        CHECK (valid_from IS NULL OR valid_until IS NULL OR valid_until >= valid_from)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_vouchers_code_active_unique
    ON m_vouchers (code) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_m_vouchers_name ON m_vouchers (name);
CREATE INDEX IF NOT EXISTS idx_m_vouchers_used_at ON m_vouchers (used_at);

CREATE TABLE IF NOT EXISTS t_sales (
    id BIGSERIAL PRIMARY KEY,
    sale_number VARCHAR(50) NOT NULL,
    sale_date DATE NOT NULL DEFAULT CURRENT_DATE,
    member_id BIGINT REFERENCES m_members(id) ON DELETE RESTRICT,
    tax_type VARCHAR(20) NOT NULL DEFAULT 'NON_PPN'
        CHECK (tax_type IN ('NON_PPN', 'TAX_EXCLUSIVE', 'TAX_INCLUSIVE')),
    subtotal NUMERIC(15,2) NOT NULL DEFAULT 0,
    item_discount NUMERIC(15,2) NOT NULL DEFAULT 0,
    discount_type VARCHAR(10) NOT NULL DEFAULT 'NONE'
        CHECK (discount_type IN ('NONE', 'PERCENT', 'AMOUNT')),
    discount_value NUMERIC(15,2) NOT NULL DEFAULT 0,
    discount_amount NUMERIC(15,2) NOT NULL DEFAULT 0,
    dpp_amount NUMERIC(15,2) NOT NULL DEFAULT 0,
    tax_amount NUMERIC(15,2) NOT NULL DEFAULT 0,
    total_amount NUMERIC(15,2) NOT NULL DEFAULT 0,
    paid_amount NUMERIC(15,2) NOT NULL DEFAULT 0,
    change_amount NUMERIC(15,2) NOT NULL DEFAULT 0
        CONSTRAINT t_sales_change_amount_non_neg CHECK (change_amount >= 0),
    points_earned BIGINT NOT NULL DEFAULT 0
        CONSTRAINT t_sales_points_earned_non_neg CHECK (points_earned >= 0),
    redeem_points BIGINT NOT NULL DEFAULT 0
        CONSTRAINT t_sales_redeem_points_non_neg CHECK (redeem_points >= 0),
    redeem_amount NUMERIC(15,2) NOT NULL DEFAULT 0
        CONSTRAINT t_sales_redeem_amount_non_neg CHECK (redeem_amount >= 0),
    voucher_id BIGINT REFERENCES m_vouchers(id) ON DELETE RESTRICT,
    voucher_code VARCHAR(50) NOT NULL DEFAULT '',
    voucher_discount NUMERIC(15,2) NOT NULL DEFAULT 0
        CONSTRAINT t_sales_voucher_discount_non_neg CHECK (voucher_discount >= 0),
    cost_total NUMERIC(15,2) NOT NULL DEFAULT 0,
    payment_status VARCHAR(10) NOT NULL DEFAULT 'UNPAID'
        CHECK (payment_status IN ('UNPAID', 'PARTIAL', 'PAID')),
    status VARCHAR(10) NOT NULL DEFAULT 'POSTED'
        CHECK (status IN ('POSTED', 'VOID')),
    due_date DATE,
    notes TEXT NOT NULL DEFAULT '',
    journal_entry_id BIGINT REFERENCES t_journal_entries(id) ON DELETE SET NULL,
    voided_at TIMESTAMP WITH TIME ZONE,
    voided_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    void_reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    updated_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    CONSTRAINT t_sales_sale_number_unique UNIQUE (sale_number)
);

CREATE INDEX IF NOT EXISTS idx_t_sales_date ON t_sales (sale_date);
CREATE INDEX IF NOT EXISTS idx_t_sales_member ON t_sales (member_id);
CREATE INDEX IF NOT EXISTS idx_t_sales_status ON t_sales (status);
CREATE UNIQUE INDEX IF NOT EXISTS idx_t_sales_voucher_posted_unique
    ON t_sales (voucher_id)
    WHERE voucher_id IS NOT NULL AND status = 'POSTED';

CREATE TABLE IF NOT EXISTS t_sales_details (
    id BIGSERIAL PRIMARY KEY,
    sale_id BIGINT NOT NULL REFERENCES t_sales(id) ON DELETE CASCADE,
    product_id BIGINT NOT NULL REFERENCES m_products(id) ON DELETE RESTRICT,
    sku VARCHAR(50) NOT NULL DEFAULT '',
    product_name VARCHAR(255) NOT NULL DEFAULT '',
    qty INT NOT NULL CHECK (qty > 0),
    unit_price NUMERIC(15,2) NOT NULL DEFAULT 0,
    discount_type VARCHAR(10) NOT NULL DEFAULT 'NONE'
        CHECK (discount_type IN ('NONE', 'PERCENT', 'AMOUNT')),
    discount_value NUMERIC(15,2) NOT NULL DEFAULT 0,
    discount_amount NUMERIC(15,2) NOT NULL DEFAULT 0,
    gross_amount NUMERIC(15,2) NOT NULL DEFAULT 0,
    line_total NUMERIC(15,2) NOT NULL DEFAULT 0,
    dpp_amount NUMERIC(15,2) NOT NULL DEFAULT 0,
    tax_amount NUMERIC(15,2) NOT NULL DEFAULT 0,
    cost_unit NUMERIC(15,4) NOT NULL DEFAULT 0,
    cost_total NUMERIC(15,2) NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_t_sales_details_sale ON t_sales_details (sale_id);
CREATE INDEX IF NOT EXISTS idx_t_sales_details_product ON t_sales_details (product_id);

CREATE TABLE IF NOT EXISTS t_sales_payments (
    id BIGSERIAL PRIMARY KEY,
    sale_id BIGINT NOT NULL REFERENCES t_sales(id) ON DELETE CASCADE,
    payment_method_id BIGINT NOT NULL REFERENCES m_payment_methods(id) ON DELETE RESTRICT,
    amount NUMERIC(15,2) NOT NULL DEFAULT 0,
    reference_number VARCHAR(100) NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT t_sales_payments_amount_check CHECK (amount > 0)
);

CREATE INDEX IF NOT EXISTS idx_t_sales_payments_sale ON t_sales_payments (sale_id);

CREATE TABLE IF NOT EXISTS t_member_points (
    id BIGSERIAL PRIMARY KEY,
    member_id BIGINT NOT NULL REFERENCES m_members(id) ON DELETE RESTRICT,
    sale_id BIGINT REFERENCES t_sales(id) ON DELETE SET NULL,
    entry_type VARCHAR(20) NOT NULL
        CHECK (entry_type IN ('EARN', 'REDEEM', 'EARN_REVERSAL', 'REDEEM_REVERSAL', 'ADJUST')),
    points BIGINT NOT NULL,
    balance_after BIGINT NOT NULL,
    note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    CONSTRAINT t_member_points_balance_after_non_neg CHECK (balance_after >= 0)
);

CREATE INDEX IF NOT EXISTS idx_t_member_points_member ON t_member_points (member_id);
CREATE INDEX IF NOT EXISTS idx_t_member_points_sale ON t_member_points (sale_id);

CREATE TABLE IF NOT EXISTS t_sales_number_counters (
    prefix VARCHAR(8) PRIMARY KEY,
    last_seq BIGINT NOT NULL DEFAULT 0,
    CONSTRAINT t_sales_number_counters_prefix_check CHECK (prefix ~ '^[0-9]{8}$'),
    CONSTRAINT t_sales_number_counters_seq_check CHECK (last_seq >= 0)
);





-- =========================================================================
-- IDEMPOTENSI HTTP
-- =========================================================================
--
-- Satu baris = satu (pengguna, endpoint, Idempotency-Key). Selama status
-- IN_PROGRESS permintaan lain dengan key sama ditolak; setelah COMPLETED
-- respons disimpan dan diputar ulang. request_hash membedakan retry payload
-- yang sama dari penyalahgunaan key untuk request berbeda.

CREATE TABLE IF NOT EXISTS t_idempotency_keys (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES s_users(id) ON DELETE CASCADE,
    scope VARCHAR(255) NOT NULL,
    idempotency_key VARCHAR(255) NOT NULL,
    request_hash CHAR(64) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'IN_PROGRESS'
        CHECK (status IN ('IN_PROGRESS', 'COMPLETED')),
    response_status INT,
    response_body BYTEA,
    response_content_type VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP + INTERVAL '24 hours',
    CONSTRAINT t_idempotency_keys_unique UNIQUE (user_id, scope, idempotency_key),
    CONSTRAINT t_idempotency_keys_completed_has_status
        CHECK (status = 'IN_PROGRESS' OR response_status IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS idx_t_idempotency_keys_expires
    ON t_idempotency_keys (expires_at);
