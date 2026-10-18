-- Kebalikan dari 000007_modul_penerimaan_barang.up.sql
--
-- Peringatan: menghapus tabel GRN juga menghapus seluruh detail penerimaan
-- barang (ON DELETE CASCADE). Hal ini hanya aman jika belum ada laporan yang
-- bergantung padanya. Selain itu, angka current_stock yang sudah ditambah lewat
-- GRN tidak akan dikoreksi oleh migrasi ini.

DROP TABLE IF EXISTS t_goods_receipt_note_details;
DROP TABLE IF EXISTS t_goods_receipt_notes;
DROP TABLE IF EXISTS t_grn_number_counters;

DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM s_permissions WHERE name IN ('grn:create', 'grn:read', 'po:receive')
);

DELETE FROM s_permissions WHERE name IN ('grn:create', 'grn:read', 'po:receive');
