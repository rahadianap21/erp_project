-- =========================================================================
-- HAPUS STATUS DRAFT DARI DAFTAR NILAI SAH (PEMBERSIHAN)
-- =========================================================================
--
-- Latar belakang:
-- migrasi 000003 memasang CHECK constraint dengan lima nilai: DRAFT, PENDING,
-- APPROVED, RECEIVED, REJECTED. Sekarang DRAFT dihapus dari daftar itu karena
-- tidak pernah dipakai sama sekali.
--
-- Bukti bahwa DRAFT tidak terjangkau:
--   * CreatePO selalu menetapkan status PENDING sebelum menyimpan dokumen
--     (purchase_order_usecase.go), jadi tidak ada jalur create yang melempar
--     dokumen berstatus DRAFT.
--   * Tidak ada kueri, service, atau handler lain yang menulis status DRAFT.
--     Satu-satunya kemunculannya di Go adalah konstanta yang tidak pernah
--     dipanggil.
--   * Tidak ada satu pun baris DRAFT di database.
--
-- Yang terjadi setelah DRAFT dihapus:
-- editablePOStatuses di backend ikut menyempit dari {DRAFT, PENDING} menjadi
-- {PENDING}. Dua test yang semula membuktikan "draft boleh dikoreksi / boleh
-- dicabut" justru tidak pernah menyangkut alur nyata, karena tidak ada dokumen
-- yang pernah sampai ke status itu; keduanya kini membuktikan perilaku PENDING.
--
-- Memperketat constraint seperti ini penting karena constraint adalah pagar
-- terakhir. Semua pemeriksaan di aplikasi bisa salah, atau ada kode baru yang
-- lupamengecek. Nilai yang tidak pernah dipakai sebaiknya tidak punya izin,
-- supaya tidak muncul sebagai opsi palsu di filter maupun timeline.
--
-- Perintah: make migrateup

-- NOT VALID dulu supaya baris yang sudah ada tidak terkunci penuh selama
-- pemindaian, lalu VALIDATE di bawah tetap memverifikasi semuanya.
ALTER TABLE t_purchase_orders
    DROP CONSTRAINT t_purchase_orders_status_check;

ALTER TABLE t_purchase_orders
    ADD CONSTRAINT t_purchase_orders_status_check
    CHECK (status IN ('PENDING', 'APPROVED', 'RECEIVED', 'REJECTED'))
    NOT VALID;

ALTER TABLE t_purchase_orders
    VALIDATE CONSTRAINT t_purchase_orders_status_check;