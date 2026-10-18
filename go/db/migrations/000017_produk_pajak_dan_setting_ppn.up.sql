-- =========================================================================
-- PAJAK PER-PRODUK & PENGATURAN PPN GLOBAL
-- =========================================================================
--
-- Pajak penjualan dipindahkan dari pemilih per-nota di POS menjadi:
--   * flag per produk m_products.is_taxable (produk lama default KENA PPN);
--   * satu tarif PPN global + mode harga, disimpan di s_app_settings:
--       - PPN_RATE        : fraksi, default 0.12 (12%);
--       - PPN_PRICE_MODE  : INCLUSIVE (harga termasuk PPN) / EXCLUSIVE;
--   * nilai pajak tiap baris disimpan di t_sales_details (dpp/tax);
--   * izin settings:read / settings:update.
--
-- Perintah: make migrateup

-- =========================================================================
-- 1. MASTER PRODUK: FLAG KENA PPN
-- =========================================================================
-- Produk lama dianggap kena PPN (TRUE) karena usaha sudah PKP. Admin dapat
-- mematikan flag ini untuk produk yang bebas PPN.
ALTER TABLE m_products
    ADD COLUMN is_taxable BOOLEAN NOT NULL DEFAULT TRUE;

-- =========================================================================
-- 2. BARIS PENJUALAN: DPP & PPN PER BARIS
-- =========================================================================
-- Kolom ini menyimpan pemisahan nilai pajak tiap baris. Baris lama (nota
-- sebelum migrasi) tetap nol karena memang tidak dihitung pajak per baris.
ALTER TABLE t_sales_details
    ADD COLUMN dpp_amount NUMERIC(15,2) NOT NULL DEFAULT 0,
    ADD COLUMN tax_amount NUMERIC(15,2) NOT NULL DEFAULT 0;

-- =========================================================================
-- 3. PENGATURAN APLIKASI (KEY-VALUE)
-- =========================================================================
CREATE TABLE IF NOT EXISTS s_app_settings (
    key VARCHAR(100) PRIMARY KEY,
    value TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL
);

INSERT INTO s_app_settings (key, value, description) VALUES
    ('PPN_RATE', '0.12', 'Tarif PPN dalam fraksi (0.12 = 12%)'),
    ('PPN_PRICE_MODE', 'INCLUSIVE', 'Mode harga jual: INCLUSIVE (harga sudah termasuk PPN) / EXCLUSIVE (PPN ditambahkan ke harga)')
ON CONFLICT (key) DO NOTHING;

-- =========================================================================
-- 4. PERMISSION & GRANT: settings:read / settings:update
-- =========================================================================
INSERT INTO s_permissions (name, description) VALUES
    ('settings:read', 'Melihat pengaturan aplikasi'),
    ('settings:update', 'Mengubah pengaturan aplikasi')
ON CONFLICT (name) DO NOTHING;

-- Beri akses ke role yang boleh mengubah master produk (admin/owner).
INSERT INTO role_permissions (role_id, permission_id)
SELECT DISTINCT rp.role_id, pnew.id
FROM role_permissions rp
JOIN s_permissions pold ON pold.id = rp.permission_id AND pold.name = 'products:update'
JOIN s_permissions pnew ON pnew.name IN ('settings:read', 'settings:update')
ON CONFLICT DO NOTHING;
