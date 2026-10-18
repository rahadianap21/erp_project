-- Pembalikan 000015_pos_wajib_lunas.
-- 1. Kembalikan metode KREDIT ke daftar aktif.
UPDATE m_payment_methods
SET is_active = TRUE,
    updated_at = NOW()
WHERE code = 'KREDIT'
  AND deleted_at IS NULL
  AND is_active = FALSE;

-- 2. Hapus kolom kembalian.
ALTER TABLE t_sales DROP CONSTRAINT IF EXISTS t_sales_change_amount_non_neg;
ALTER TABLE t_sales DROP COLUMN IF EXISTS change_amount;
