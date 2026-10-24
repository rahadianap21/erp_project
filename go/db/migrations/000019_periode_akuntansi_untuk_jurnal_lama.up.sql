-- =========================================================================
-- BACKFILL PERIODE AKUNTANSI UNTUK JURNAL YANG SUDAH ADA
-- =========================================================================
--
-- Latar belakang:
-- Jurnal lama tidak memiliki foreign key ke m_accounting_periods. Akibatnya
-- bisa ada jurnal pada periode yang belum pernah dibuatkan baris periodenya.
-- Setelah migrasi ini, penulisan jurnal POSTED akan mensyaratkan periode OPEN,
-- sehingga jurnal lama tersebut akan kehilangan kemampuan menerima jurnal baru
-- atau pembalik bila baris periodenya tidak lebih dulu dibuat.
--
-- Perbaikan: buat satu periode OPEN untuk setiap pasangan (tahun, bulan) yang
-- sudah dipakai jurnal. Baris yang sudah ada tidak diubah statusnya.

INSERT INTO m_accounting_periods (period_year, period_month, status)
SELECT DISTINCT
    j.period_year,
    j.period_month,
    'OPEN'
FROM t_journal_entries j
ON CONFLICT (period_year, period_month) DO NOTHING;
