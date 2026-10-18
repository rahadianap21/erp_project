-- Kebalikan dari 000010_rata_rata_harga_pokok.up.sql: hapus harga pokok
-- rata-rata dari master produk.
--
-- Peringatan: kolom ini menyimpan hasil ringkasan seluruh riwayat penerimaan
-- yang pernah dilakukan terhadap suatu produk. Rolling back menghapus ringkasan
-- itu secara permanen dan tidak ada query yang bisa membangunnya kembali
-- tanpa mengulang seluruh dokumen GRN beserta harganya, sedangkan harga tidak
-- lagi tersimpan pada dokumen GRN.
--
-- Setelah kolom ini hilang, saldo buku besar Persediaan tetap ada dan tetap
-- benar, tapi harga pokok yang harus dipakai untuk pengurangan stok pada
-- penjualan jadi tidak diketahui lagi. Karena itu down migration ini hanya
-- aman bila modul penjualan/penyusutan Persediaan belum pernah dipakai.

ALTER TABLE m_products
    DROP CONSTRAINT IF EXISTS m_products_avg_cost_check;

ALTER TABLE m_products
    DROP COLUMN IF EXISTS avg_cost;
