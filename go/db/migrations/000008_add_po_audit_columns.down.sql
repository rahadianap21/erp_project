-- 000008_add_po_audit_columns.down.sql
-- Rollback kolom audit PO

ALTER TABLE t_purchase_orders
    DROP CONSTRAINT IF EXISTS fk_po_approved_by,
    DROP CONSTRAINT IF EXISTS fk_po_rejected_by;

ALTER TABLE t_purchase_orders
    DROP COLUMN IF EXISTS approved_by,
    DROP COLUMN IF EXISTS approved_at,
    DROP COLUMN IF EXISTS rejected_by,
    DROP COLUMN IF EXISTS rejected_at,
    DROP COLUMN IF EXISTS reject_reason;
