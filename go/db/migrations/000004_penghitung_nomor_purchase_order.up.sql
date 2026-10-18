-- =========================================================================
-- PENGHITUNG NOMOR DOKUMEN PURCHASE ORDER (PERBAIKAN NOMOR BENTROK)
-- =========================================================================
--
-- Latar belakang:
-- nomor PO dulu dibuat dari COUNT(...)+1 terhadap baris PO yang masih aktif.
-- Tiga masalah muncul dari sana:
--
--   1. Balapan. Dua permintaan bersamaan sama-sama menghitung nilai yang sama
--      lalu sama-sama memakai nomor itu. Hanya satu yang berhasil; sisanya gagal
--      dengan pelanggaran unique constraint.
--   2. Soft-delete membuat nomor dipakai ulang. Penghitung mengabaikan baris yang
--      dihapus, padahal barisnya tetap ada di tabel. Setelah PO hari ini dihapus,
--      PO berikutnya mencoba memakai nomor yang sama dan gagal.
--   3. Pencarian yang dipakai untuk menghitung juga mencocokkan NAMA SUPPLIER
--      yang kebetulan memuat prefiks hari itu, sehingga hitungannya bisa salah.
--
-- Perbaikan:
-- penghitung dipindah ke tabel khusus dan menaikkan nomor dengan satu UPSERT
-- atomik. Penambahan ini:
--   * balapan aman, karena baris prefiks yang sama dikunci PostgreSQL sehingga
--     setiap pemanggil mendapat angka berbeda;
--   * tidak bergantung pada baris PO, jadi soft-delete tidak menyebabkan nomor
--     dipakai ulang;
--   * tidak pernah mundur, sehingga nomor yang sudah terbit tidak akan pernah
--     diberikan lagi.
--
-- Kolom last_seq sengaja tidak dihitung ulang dari tabel PO. Kalau dihitung,
-- soft-delete atau penghapusan arsip akan membuat nomor kembali ke belakang.
--
-- Perintah: make migrateup

CREATE TABLE IF NOT EXISTS t_po_number_counters (
    prefix   character varying(8) PRIMARY KEY,
    last_seq bigint NOT NULL DEFAULT 0,
    CONSTRAINT t_po_number_counters_prefix_check CHECK (prefix ~ '^[0-9]{8}$'),
    CONSTRAINT t_po_number_counters_seq_check    CHECK (last_seq >= 0)
);

-- Backfill: seed per tanggal yang sudah punya nomor, memakai nomor TERTINGGI
-- per tanggal. Baris soft-deleted ikut dihitung dengan sengaja: nomornya sudah
-- pernah terbit dan tidak boleh dipakai ulang.
--
-- Slicing dipecah begini supaya pola dapat menangkap digit di tengah nama kolom.
INSERT INTO t_po_number_counters (prefix, last_seq)
SELECT substring(po.po_number from 4 for 8) AS prefix,
       MAX(substring(po.po_number from 13 for 4))::bigint AS last_seq
FROM t_purchase_orders po
WHERE po.po_number ~ '^PO/[0-9]{8}/[0-9]{4}$'
GROUP BY 1
ON CONFLICT (prefix) DO NOTHING;

-- Jaring pengaman: kalau ada nomor yang tidak cocok polanya, seed konservatif
-- memakai hitungan baris agar nomor baru tidak pernah menabrak nomor lama.
INSERT INTO t_po_number_counters (prefix, last_seq)
SELECT to_char(po.po_date, 'YYYYMMDD') AS prefix, COUNT(*)::bigint
FROM t_purchase_orders po
WHERE NOT EXISTS (
        SELECT 1 FROM t_po_number_counters c WHERE c.prefix = to_char(po.po_date, 'YYYYMMDD')
      )
  AND po.po_date IS NOT NULL
GROUP BY 1
ON CONFLICT (prefix) DO NOTHING;