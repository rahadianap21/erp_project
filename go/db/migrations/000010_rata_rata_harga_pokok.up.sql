-- =========================================================================
-- RATA-RATA HARGA POKOK PENERIMAAN (MOVING AVERAGE) PADA MASTER PRODUK
-- =========================================================================
--
-- Latar belakang:
-- Saat ini m_products hanya punya purchase_price (harga beli awal, diisi
-- manual oleh pengguna) dan current_stock (dihitung modul penerimaan).
-- Tidak ada angka yang menyatakan berapa satu satuan jual benar-benar
-- berharga setelah barang masuk berkali-kali dengan harga berbeda.
--
-- Tanpa angka itu laporan keuangan tidak bisa menutup siklusnya. Buku besar
-- Persediaan sudah terisi oleh jurnal otomatis GRN (Fase 2A) dengan nilai
-- penerimaan sesungguhnya, tapi stok yang dicatat melalui penjualan harus
-- dihapus dengan HARGA YANG SAMA. Bila stok dihapus pada harga beli awal,
-- saldo buku besar dan nilai persediaan fisik akan terus menyebar.
--
-- Keputusan: Moving Average.
-- Setiap penerimaan menghitung ulang:
--     avg_cost = (stok_lama * avg_cost_lama + nilai_diterima) / stok_baru
--
-- Syarat pemakaian kolom ini:
--   * dinyatakan per SATUAN JUAL, sama seperti current_stock,
--   * nilai_diterima adalah DPP (pokok pengenaan pajak), bukan bruto,
--     karena PPN masukan adalah tagihan yang diterima kembali dari negara,
--     bukan biaya perolehan barang,
--   * kolom ini HANYA ditulis modul penerimaan barang, sebagaimana
--     current_stock. Update produk tidak boleh menyentuhnya supaya ada satu
--     penulis dan satu jejak bukti (GRN) untuk setiap perubahan.
--
-- Tipe NUMERIC(15,4) dipilih karena harga pokok SATUAN hasil bagi sering
-- mengandung lebih dari dua desimal. Membulatkannya ke dua desimal akan
-- menambah kesalahan setiap kali angka itu dikalikan jutaan baris stok.
-- Uang tetap dua desimal di kolom lain; hanya harga pokok per satuan yang
-- lebih halus.

ALTER TABLE m_products
    ADD COLUMN IF NOT EXISTS avg_cost NUMERIC(15,4) NOT NULL DEFAULT 0;

ALTER TABLE m_products
    DROP CONSTRAINT IF EXISTS m_products_avg_cost_check;

ALTER TABLE m_products
    ADD CONSTRAINT m_products_avg_cost_check CHECK (avg_cost >= 0);
