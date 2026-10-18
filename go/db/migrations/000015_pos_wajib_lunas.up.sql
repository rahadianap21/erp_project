-- =========================================================================
-- POS WAJIB LUNAS + KEMBALIAN + NONAKTIFKAN METODE KREDIT
-- =========================================================================
--
-- Aturan baru POS:
--   * Setiap nota wajib lunas: total pembayaran >= total nota. Kelebihan
--     bayar dicatat sebagai kembalian di t_sales.change_amount (tidak
--     dijurnal; kas naik sesuai total nota).
--   * Satu metode pembayaran per nota (divalidasi di domain).
--   * Metode pembayaran KREDIT / Tempo dinonaktifkan dari daftar aktif
--     (master data tetap ada, hanya tak muncul di POS). Modul penjualan
--     B2B tempo akan memakai menu terpisah pada tahap berikutnya.
--
-- Perintah: make migrateup

-- 1. Kolom kembalian pada nota penjualan.
ALTER TABLE t_sales
    ADD COLUMN change_amount NUMERIC(15,2) NOT NULL DEFAULT 0
    CONSTRAINT t_sales_change_amount_non_neg CHECK (change_amount >= 0);

-- 2. Sembunyikan metode KREDIT dari daftar metode aktif (termasuk bootstrap POS).
UPDATE m_payment_methods
SET is_active = FALSE,
    updated_at = NOW()
WHERE code = 'KREDIT'
  AND deleted_at IS NULL
  AND is_active = TRUE;
