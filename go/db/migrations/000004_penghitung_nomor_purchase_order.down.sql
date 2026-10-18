-- =========================================================================
-- MUNDURKAN PENGHITUNG NOMOR PURCHASE ORDER
-- =========================================================================
--
-- Menghapus tabel penghitung. Nomor PO yang sudah terbit tidak tersentuh, jadi
-- setelah migrasi ini aplikasi kembali memakai COUNT+1 dan nomor bentrok bisa
-- terjadi lagi. Jalankan ulang migrasi naik agar penghitung aktif kembali.

DROP TABLE IF EXISTS t_po_number_counters;