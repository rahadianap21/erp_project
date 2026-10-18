-- =========================================================================
-- FAKTOR KONVERSI SATUAN PADA MASTER PRODUK
-- =========================================================================
--
-- Latar belakang:
-- m_products menyimpan DUA satuan sekaligus: purchase_unit_id (satuan beli)
-- dan selling_unit_id (satuan jual). Sampai sekarang tidak ada satu pun kolom
-- yang menyatakan berapa banyak satuan jual yang didapat dari satu satuan beli.
-- Padahal m_units hanya punya kode, nama, deskripsi, dan is_active — tidak ada
-- tempat menyimpan kelipatan.
--
-- Akibatnya kolom current_stock tidak punya satuan yang jelas. Modul penerimaan
-- barang (GRN) adalah kode pertama yang akan MENULIS current_stock, dan angka
-- yang ditulis harus langsung benar karena tidak ada modul lain yang bisa
-- membandingkannya.
--
-- Keputusan: mulai sekarang current_stock dinyatakan dalam SATUAN JUAL, yaitu
-- satuan terkecil. Satuan ini paling granular sehingga penyusun laporan dan
-- peringatan min_stock_level tidak perlu tahu satuan beli apa pun.
--
-- Faktor konversi disederhanakan jadi satu kolom pada m_products, bukan tabel
-- konversi terpisah. Alasannya: skema produk hanya mengizinkan satu pasangan
-- purchase_unit_id + selling_unit_id per produk, jadi satu produk paling banyak
-- butuh satu faktor. Tabel terpisah baru perlu kalau produk boleh dibeli dalam
-- beberapa satuan sekaligus, dan itu belum jadi kebutuhan.
--
-- Tipe INT dengan CHECK >= 1 dipilih supaya:
--   * aritmetika stok tetap bilangan bulat; current_stock tidak perlu diubah
--     menjadi NUMERIC,
--   * selling_unit_id yang bermakna "satuan terkecil" selalu punya faktor
--     >= 1. Faktor pecahan (mis. 1 KG = 0.5 KG) akan meruntuhkan definisi
--     satuan terkecil itu sendiri dan perlu kolom NUMERIC beserta penyesuaian
--     lain.
--
-- Faktor yang tidak bisa berlaku untuk suatu produk dibiarkan bernilai 1
-- sebagai default, BUKAN dikarang. Perlu dicatat: lima dari enam produk aktif
-- sekarang punya purchase_unit_id yang berbeda dari selling_unit_id, dan tiga
-- di antaranya tidak masuk akal (kursi kerja yang satuan belinya SHEET,
-- ball bearing yang satuan belinya SHEET, pelat besi yang dibalik). Pemilik data
-- harus memperbaikinya sendiri lewat form produk. Yang bisa dilakukan
-- sistem hanyalah memasang kolom dan validasinya, lalu menolak penerimaan barang
-- yang akan memakai faktor yang belum benar.
--
-- Backfill: kolom sengaja TIDAK diisi ulang lewat migrasi. Nilai current_stock
-- yang sudah ada berasal dari data contoh dan tidak diketahui lagi satuan
-- apa pun. Reset manual ke 0 lebih aman daripada mempertahankan angka yang
-- salah, tapi keputusan itu milik pemilik data, bukan migrasi.
--
-- Perintah: make migrateup

ALTER TABLE m_products
    ADD COLUMN IF NOT EXISTS conversion_rate INT NOT NULL DEFAULT 1;

-- Pagar integritas: faktor konversi harus bilangan bulat positif.
ALTER TABLE m_products
    DROP CONSTRAINT IF EXISTS m_products_conversion_rate_check;

ALTER TABLE m_products
    ADD CONSTRAINT m_products_conversion_rate_check
    CHECK (conversion_rate >= 1);

-- Faktor >= 1 membuat current_stock yang bertambah selalu tidak negatif, jadi
-- CHECK current_stock >= 0 yang sudah ada tetap berlaku dan tidak perlu diubah.