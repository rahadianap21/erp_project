-- =========================================================================
-- BATAL: PAJAK PER-PRODUK & PENGATURAN PPN GLOBAL
-- =========================================================================

-- =========================================================================
-- 4. PERMISSION & GRANT
-- =========================================================================
DELETE FROM role_permissions
WHERE permission_id IN (SELECT id FROM s_permissions WHERE name IN ('settings:read', 'settings:update'));

DELETE FROM s_permissions WHERE name IN ('settings:read', 'settings:update');

-- =========================================================================
-- 3. PENGATURAN APLIKASI
-- =========================================================================
DROP TABLE IF EXISTS s_app_settings;

-- =========================================================================
-- 2. BARIS PENJUALAN
-- =========================================================================
ALTER TABLE t_sales_details
    DROP COLUMN IF EXISTS dpp_amount,
    DROP COLUMN IF EXISTS tax_amount;

-- =========================================================================
-- 1. MASTER PRODUK
-- =========================================================================
ALTER TABLE m_products DROP COLUMN IF EXISTS is_taxable;
