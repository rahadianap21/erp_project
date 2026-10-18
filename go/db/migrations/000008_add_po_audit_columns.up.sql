-- 000008_add_po_audit_columns.up.sql
-- Tambahkan kolom audit untuk approval/rejection Purchase Order

ALTER TABLE t_purchase_orders
    ADD COLUMN IF NOT EXISTS approved_by BIGINT DEFAULT NULL,
    ADD COLUMN IF NOT EXISTS approved_at TIMESTAMP DEFAULT NULL,
    ADD COLUMN IF NOT EXISTS rejected_by BIGINT DEFAULT NULL,
    ADD COLUMN IF NOT EXISTS rejected_at TIMESTAMP DEFAULT NULL,
    ADD COLUMN IF NOT EXISTS reject_reason TEXT DEFAULT NULL;

ALTER TABLE t_purchase_orders
    ADD CONSTRAINT fk_po_approved_by FOREIGN KEY (approved_by) REFERENCES s_users(id) ON DELETE SET NULL,
    ADD CONSTRAINT fk_po_rejected_by FOREIGN KEY (rejected_by) REFERENCES s_users(id) ON DELETE SET NULL;
