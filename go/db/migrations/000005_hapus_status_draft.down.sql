-- Kebalikan dari 000005_hapus_status_draft.up.sql: kembalikan DRAFT ke daftar
-- nilai sah dan kembalikan editablePOStatuses di backend ke {DRAFT, PENDING}.
--
-- Peringatan: constraint versi lama lebih longgar daripada kode aplikasi yang
-- sekarang. Mengaktifkan ulang migrasi ini tanpa mengembalikan editablePOStatuses
-- akan membuat dokumen DRAFT tidak bisa dikoreksi maupun dihapus, padahal
-- database mengizinkannya. Keduanya harus dikembalikan bersama.

ALTER TABLE t_purchase_orders
    DROP CONSTRAINT t_purchase_orders_status_check;

ALTER TABLE t_purchase_orders
    ADD CONSTRAINT t_purchase_orders_status_check
    CHECK (status IN ('DRAFT', 'PENDING', 'APPROVED', 'RECEIVED', 'REJECTED'))
    NOT VALID;

ALTER TABLE t_purchase_orders
    VALIDATE CONSTRAINT t_purchase_orders_status_check;