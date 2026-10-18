-- =========================================================================
-- BATAL: MEMBER & POIN LOYALITAS
-- =========================================================================

-- =========================================================================
-- 4. PERMISSION & GRANT: members:* -> customers:*
-- =========================================================================
INSERT INTO s_permissions (name, description)
SELECT
    replace(p.name, 'members:', 'customers:'),
    replace(p.description, 'member', 'pelanggan')
FROM s_permissions p
WHERE p.name LIKE 'members:%'
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT rp.role_id, pnew.id
FROM role_permissions rp
JOIN s_permissions pold ON pold.id = rp.permission_id AND pold.name LIKE 'members:%'
JOIN s_permissions pnew ON pnew.name = replace(pold.name, 'members:', 'customers:')
ON CONFLICT DO NOTHING;

DELETE FROM role_permissions
WHERE permission_id IN (SELECT id FROM s_permissions WHERE name LIKE 'members:%');

DELETE FROM s_permissions WHERE name LIKE 'members:%';

-- =========================================================================
-- 3. BUKU BESAR POIN
-- =========================================================================
DROP TABLE IF EXISTS t_member_points;

-- =========================================================================
-- 2. TRANSAKSI PENJUALAN
-- =========================================================================
ALTER TABLE t_sales
    DROP CONSTRAINT IF EXISTS t_sales_points_earned_non_neg,
    DROP CONSTRAINT IF EXISTS t_sales_redeem_points_non_neg,
    DROP CONSTRAINT IF EXISTS t_sales_redeem_amount_non_neg;

ALTER TABLE t_sales
    DROP COLUMN IF EXISTS points_earned,
    DROP COLUMN IF EXISTS redeem_points,
    DROP COLUMN IF EXISTS redeem_amount;

-- Pulihkan NOT NULL hanya jika semua baris punya member.
UPDATE t_sales SET member_id = (SELECT id FROM m_members WHERE code = 'UMUM' ORDER BY id LIMIT 1)
WHERE member_id IS NULL;

ALTER TABLE t_sales ALTER COLUMN member_id SET NOT NULL;
ALTER TABLE t_sales RENAME CONSTRAINT t_sales_member_id_fkey TO t_sales_customer_id_fkey;
ALTER INDEX idx_t_sales_member RENAME TO idx_t_sales_customer;
ALTER TABLE t_sales RENAME COLUMN member_id TO customer_id;

-- =========================================================================
-- 1. MASTER MEMBER
-- =========================================================================
UPDATE m_members
SET deleted_at = NULL, is_active = TRUE, updated_at = NOW()
WHERE code = 'UMUM';

ALTER TABLE m_members DROP CONSTRAINT IF EXISTS m_members_points_non_neg;
ALTER TABLE m_members DROP COLUMN IF EXISTS points;

DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN
        SELECT conname, replace(conname, 'm_members', 'm_customers') AS newname
        FROM pg_constraint
        WHERE conrelid = 'm_members'::regclass
          AND conname LIKE 'm_members%'
    LOOP
        EXECUTE format('ALTER TABLE m_members RENAME CONSTRAINT %I TO %I', r.conname, r.newname);
    END LOOP;
END $$;

ALTER INDEX idx_members_code_active_unique RENAME TO idx_customers_code_active_unique;
ALTER INDEX idx_m_members_name RENAME TO idx_m_customers_name;
ALTER TABLE m_members RENAME TO m_customers;
