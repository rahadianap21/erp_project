-- =========================================================================
-- MASTER VOUCHER & DISKON POS
-- =========================================================================
--
-- Voucher adalah kupon sekali pakai yang di-scan kasir di POS. Kode voucher
-- menjadi isi barcode (Code128) dan unik selama baris belum dihapus.
--
--   * m_vouchers : master kupon (persen/nominal, minimal belanja, masa berlaku,
--                  penanda used_at untuk pemakaian tunggal);
--   * t_sales    : mencatat voucher yang dipakai beserta nominal potongannya;
--   * izin vouchers:* untuk admin, sedangkan kasir memakai pos:read.
--
-- Perintah: make migrateup

-- =========================================================================
-- 1. MASTER VOUCHER
-- =========================================================================
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

-- =========================================================================
-- 2. TRANSAKSI PENJUALAN: VOUCHER
-- =========================================================================
ALTER TABLE t_sales
    ADD COLUMN IF NOT EXISTS voucher_id BIGINT REFERENCES m_vouchers(id) ON DELETE RESTRICT,
    ADD COLUMN IF NOT EXISTS voucher_code VARCHAR(50) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS voucher_discount NUMERIC(15,2) NOT NULL DEFAULT 0
        CONSTRAINT t_sales_voucher_discount_non_neg CHECK (voucher_discount >= 0);

-- Satu voucher hanya boleh menempel pada satu nota POSTED. Nota VOID tidak
-- dihitung sehingga kupon dapat dipakai ulang setelah pembatalan.
CREATE UNIQUE INDEX IF NOT EXISTS idx_t_sales_voucher_posted_unique
    ON t_sales (voucher_id)
    WHERE voucher_id IS NOT NULL AND status = 'POSTED';

-- =========================================================================
-- 3. PERMISSION & GRANT
-- =========================================================================
INSERT INTO s_permissions (name, description) VALUES
    ('vouchers:read', 'Melihat master voucher'),
    ('vouchers:create', 'Menambah voucher'),
    ('vouchers:update', 'Mengubah voucher'),
    ('vouchers:delete', 'Menghapus voucher')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM s_roles r, s_permissions p
WHERE r.name IN ('Super Admin', 'Admin')
  AND p.name IN ('vouchers:read', 'vouchers:create', 'vouchers:update', 'vouchers:delete')
ON CONFLICT DO NOTHING;
