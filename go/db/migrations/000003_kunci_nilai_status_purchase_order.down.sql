-- =========================================================================
-- MUNDURKAN PERBATASAN NILAI STATUS PURCHASE ORDER
-- =========================================================================
--
-- Hapus CHECK constraint agar status bebas kembali. Data yang sudah ada tidak
-- diubah oleh migrasi ini.
--
-- Peringatan: tanpa constraint ini, compare-and-swap tetap aman dari sisi balapan
-- karena syarat status dievaluasi di dalam UPDATE. Yang hilang adalah jaminan
-- bahwa status selalu salah satu dari lima nilai yang dikenal.

ALTER TABLE t_purchase_orders
    DROP CONSTRAINT IF EXISTS t_purchase_orders_status_check;