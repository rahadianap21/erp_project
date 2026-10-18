-- =========================================================================
-- PERBATASAN NILAI STATUS PURCHASE ORDER (PERBAIKAN RACE APPROVE/CANCEL)
-- =========================================================================
--
-- Latar belakang:
-- kolom t_purchase_orders.status tidak punya batasan nilai apa pun, sehingga
-- string apa saja bisa masuk. Race approve/cancel membuat masalah ini lebih
-- serius: versi lama membaca status lewat SELECT lalu menulis status baru lewat
-- UPDATE tanpa syarat status, jadi dua permintaan yang datang bersamaan bisa
-- sama-sama berhasil dan dokumen diam-diam berakhir pada status penulisan
-- terakhir.
--
-- Perbaikan status dilakukan sebagai compare-and-swap (lihat
-- TransitionPurchaseOrderStatus), dan daftar status yang sah dikunci di sini
-- supaya daftar itu tidak bisa aus seiring waktu.
--
-- Nilai yang diizinkan mengikuti alur dokumen yang sudah dipakai aplikasi:
--   DRAFT     -> dokumen baru, masih bisa dikoreksi
--   PENDING   -> menunggu persetujuan
--   APPROVED  -> sudah disetujui, menjadi rujukan penerimaan barang
--   RECEIVED  -> barang sudah diterima di gudang
--   REJECTED -> dibatalkan
--
-- Perintah: make migrateup

-- NOT VALID dipakai supaya migrasi ini tidak mengunci penuh tabel saat memindai seluruh baris; VALIDATE di bawah tetap memverifikasi semua baris
-- lama, jadi hanya pemindaian yang ditunda, bukan pemeriksaannya.
ALTER TABLE t_purchase_orders
    ADD CONSTRAINT t_purchase_orders_status_check
    CHECK (status IN ('DRAFT', 'PENDING', 'APPROVED', 'RECEIVED', 'REJECTED'))
    NOT VALID;

ALTER TABLE t_purchase_orders
    VALIDATE CONSTRAINT t_purchase_orders_status_check;