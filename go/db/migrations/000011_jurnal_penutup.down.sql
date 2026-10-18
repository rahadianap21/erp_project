-- Kembalikan skema source_type dan unique index idempotensi ke bentuk semula.
--
-- Peringatan: migrasi turun ini gagal bila masih ada jurnal ber-source_type
-- 'CLOSING'. Batalkan dulu seluruh jurnal penutup sebelum menurunkan migrasi.

DROP INDEX IF EXISTS idx_journal_source_unique;

CREATE UNIQUE INDEX IF NOT EXISTS idx_journal_source_unique
    ON t_journal_entries (source_type, source_id)
    WHERE source_type IS NOT NULL AND source_id IS NOT NULL AND status = 'POSTED';

ALTER TABLE t_journal_entries
    DROP CONSTRAINT IF EXISTS t_journal_entries_source_type_check;

ALTER TABLE t_journal_entries
    ADD CONSTRAINT t_journal_entries_source_type_check
        CHECK (source_type IN ('MANUAL', 'GRN', 'SUPPLIER_PAYMENT', 'CASH_IN', 'CASH_OUT'));
