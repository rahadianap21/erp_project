-- =========================================================================
-- JURNAL PENUTUP (CLOSING ENTRY) PERIODE FISIKAL
-- =========================================================================
--
-- Latar belakang:
-- Penutupan periode sebelumnya hanya mengunci periode (status CLOSED) tanpa
-- memindahkan saldo akun pendapatan dan beban ke Laba Ditahan. Akibatnya saldo
-- pendapatan/beban menumpuk selamanya dan Neraca harus memakai baris ekuitas
-- sintetis "LABA (RUGI) BELUM DITUTUP" supaya tetap seimbang.
--
-- Mulai sekarang penutupan periode menerbitkan jurnal penutup otomatis:
--   * setiap akun PENDAPATAN dan BEBAN disanatkan ke nol,
--   * selisihnya (laba/rugi) dipindahkan ke akun Laba Ditahan (RETAINED_EARNINGS).
-- Jurnal itu diberi source_type 'CLOSING' dengan source_id = id periode,
-- sehingga unique index parsial (source_type, source_id) menjamin paling banyak
-- satu jurnal penutup POSTED untuk satu periode.
--
-- Dua perubahan skema diperlukan:
--   1. CHECK source_type harus mengizinkan nilai 'CLOSING'.
--   2. Unique index idempotensi harus MENGABAIKAN jurnal yang sudah dibalik
--      (reversed_by_id IS NOT NULL). Saat periode dibuka kembali, jurnal
--      penutupnya dibalik agar pembukuan kembali seperti sebelum tutup buku;
--      tanpa pengecualian ini, periode yang sama tidak akan pernah bisa
--      ditutup ulang karena slot (source_type, source_id) masih dianggap
--      terpakai oleh jurnal penutup lama yang statusnya tetap POSTED.

ALTER TABLE t_journal_entries
    DROP CONSTRAINT IF EXISTS t_journal_entries_source_type_check;

ALTER TABLE t_journal_entries
    ADD CONSTRAINT t_journal_entries_source_type_check
        CHECK (source_type IN ('MANUAL', 'GRN', 'SUPPLIER_PAYMENT', 'CASH_IN', 'CASH_OUT', 'CLOSING'));

DROP INDEX IF EXISTS idx_journal_source_unique;

CREATE UNIQUE INDEX IF NOT EXISTS idx_journal_source_unique
    ON t_journal_entries (source_type, source_id)
    WHERE source_type IS NOT NULL
      AND source_id IS NOT NULL
      AND status = 'POSTED'
      AND reversed_by_id IS NULL;
