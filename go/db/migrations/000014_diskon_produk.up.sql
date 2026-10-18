-- =========================================================================
-- MODUL PENJUALAN / KASIR (POS) - DISKON PER PRODUK
-- =========================================================================
-- Admin dapat mengatur diskon bawaan per produk (jenis + nilai). Diskon ini
-- bersifat opsional dan dapat diubah saat transaksi POS.

ALTER TABLE m_products
    ADD COLUMN IF NOT EXISTS discount_type VARCHAR(10) NOT NULL DEFAULT 'NONE',
    ADD COLUMN IF NOT EXISTS discount_value NUMERIC(15,2) NOT NULL DEFAULT 0;

-- Postgres tidak mendukung ADD CONSTRAINT IF NOT EXISTS; dijaga lewat DO block.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'm_products'::regclass
          AND conname = 'm_products_discount_type_check'
    ) THEN
        ALTER TABLE m_products ADD CONSTRAINT m_products_discount_type_check
            CHECK (discount_type IN ('NONE', 'PERCENT', 'AMOUNT'));
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'm_products'::regclass
          AND conname = 'm_products_discount_value_non_neg_check'
    ) THEN
        ALTER TABLE m_products ADD CONSTRAINT m_products_discount_value_non_neg_check
            CHECK (discount_value >= 0);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conrelid = 'm_products'::regclass
          AND conname = 'm_products_discount_pct_check'
    ) THEN
        ALTER TABLE m_products ADD CONSTRAINT m_products_discount_pct_check
            CHECK (discount_type != 'PERCENT' OR (discount_value >= 0 AND discount_value <= 100));
    END IF;
END $$;
