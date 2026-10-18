-- =========================================================================
-- ROLLBACK DISKON PER PRODUK
-- =========================================================================

ALTER TABLE m_products
    DROP CONSTRAINT IF EXISTS m_products_discount_pct_check,
    DROP CONSTRAINT IF EXISTS m_products_discount_value_non_neg_check,
    DROP CONSTRAINT IF EXISTS m_products_discount_type_check;

ALTER TABLE m_products
    DROP COLUMN IF EXISTS discount_value,
    DROP COLUMN IF EXISTS discount_type;
