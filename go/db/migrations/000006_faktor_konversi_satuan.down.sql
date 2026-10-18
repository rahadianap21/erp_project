-- Kebalikan dari 000006_faktor_konversi_satuan.up.sql: hapus faktor konversi
-- dan kembali ke kenyataan bahwa current_stock tidak punya satuan yang jelas.
--
-- Peringatan: menghapus kolom ini menghilangkan satu-satunya angka yang
-- menyatakan berapa satuan jual yang didapat dari satu satuan beli. Modul
-- penerimaan barang (GRN) menghitung current_stock dengan mengalikan qty yang
-- diterima dengan faktor ini, jadi menjalankan migrasi ini pada database yang
-- sudah dipakai menerima barang akan membuat angka stok yang sudah tercatat
-- kehilangan dasar perhitungannya.
--
-- Karena itu down migration ini hanya aman pada database yang belum pernah
-- dipakai menerima barang. Bila memang harus kembali ke skema lama, hapus
-- seluruh data penerimaan barang secara eksplisit lebih dahulu.

ALTER TABLE m_products
    DROP CONSTRAINT IF EXISTS m_products_conversion_rate_check;

ALTER TABLE m_products
    DROP COLUMN IF EXISTS conversion_rate;