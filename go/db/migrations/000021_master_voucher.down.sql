-- =========================================================================
-- BATAL: MASTER VOUCHER & DISKON POS
-- =========================================================================

-- =========================================================================
-- 3. PERMISSION & GRANT
-- =========================================================================
DELETE FROM role_permissions
WHERE permission_id IN (SELECT id FROM s_permissions WHERE name LIKE 'vouchers:%');

DELETE FROM s_permissions WHERE name LIKE 'vouchers:%';

-- =========================================================================
-- 2. TRANSAKSI PENJUALAN: VOUCHER
-- =========================================================================
DROP INDEX IF EXISTS idx_t_sales_voucher_posted_unique;

ALTER TABLE t_sales
    DROP CONSTRAINT IF EXISTS t_sales_voucher_discount_non_neg,
    DROP COLUMN IF EXISTS voucher_discount,
    DROP COLUMN IF EXISTS voucher_code,
    DROP COLUMN IF EXISTS voucher_id;

-- =========================================================================
-- 1. MASTER VOUCHER
-- =========================================================================
DROP TABLE IF EXISTS m_vouchers;
