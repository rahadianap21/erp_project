-- =========================================================================
-- MEMBER & POIN LOYALITAS
-- =========================================================================
--
-- Pelanggan penjualan berubah menjadi "member" yang menyimpan saldo poin.
--   * m_customers -> m_members (+ kolom points);
--   * t_sales.customer_id -> member_id, kini NULLABLE (POS boleh tanpa member);
--   * t_sales + points_earned / redeem_points / redeem_amount;
--   * tabel buku besar poin t_member_points (earn/redeem/reversal/adjust);
--   * izin customers:* -> members:*;
--   * pelanggan 'UMUM' di-soft-delete (POS default "Tanpa Member").
--
-- Perintah: make migrateup

-- =========================================================================
-- 1. MASTER MEMBER
-- =========================================================================
ALTER TABLE m_customers RENAME TO m_members;
ALTER INDEX idx_customers_code_active_unique RENAME TO idx_members_code_active_unique;
ALTER INDEX idx_m_customers_name RENAME TO idx_m_members_name;

-- Rapikan nama constraint yang masih berawalan m_customers -> m_members.
DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN
        SELECT conname, replace(conname, 'm_customers', 'm_members') AS newname
        FROM pg_constraint
        WHERE conrelid = 'm_members'::regclass
          AND conname LIKE 'm_customers%'
    LOOP
        EXECUTE format('ALTER TABLE m_members RENAME CONSTRAINT %I TO %I', r.conname, r.newname);
    END LOOP;
END $$;

ALTER TABLE m_members
    ADD COLUMN points BIGINT NOT NULL DEFAULT 0,
    ADD CONSTRAINT m_members_points_non_neg CHECK (points >= 0);

-- Member 'UMUM' tidak lagi dipakai; POS default tanpa member.
UPDATE m_members
SET deleted_at = NOW(), is_active = FALSE, updated_at = NOW()
WHERE code = 'UMUM' AND deleted_at IS NULL;

-- =========================================================================
-- 2. TRANSAKSI PENJUALAN: MEMBER OPSIONAL + POIN
-- =========================================================================
ALTER TABLE t_sales RENAME COLUMN customer_id TO member_id;
ALTER TABLE t_sales ALTER COLUMN member_id DROP NOT NULL;
ALTER TABLE t_sales RENAME CONSTRAINT t_sales_customer_id_fkey TO t_sales_member_id_fkey;
ALTER INDEX idx_t_sales_customer RENAME TO idx_t_sales_member;

ALTER TABLE t_sales
    ADD COLUMN points_earned BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN redeem_points BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN redeem_amount NUMERIC(15,2) NOT NULL DEFAULT 0,
    ADD CONSTRAINT t_sales_points_earned_non_neg CHECK (points_earned >= 0),
    ADD CONSTRAINT t_sales_redeem_points_non_neg CHECK (redeem_points >= 0),
    ADD CONSTRAINT t_sales_redeem_amount_non_neg CHECK (redeem_amount >= 0);

-- =========================================================================
-- 3. BUKU BESAR POIN
-- =========================================================================
CREATE TABLE IF NOT EXISTS t_member_points (
    id BIGSERIAL PRIMARY KEY,
    member_id BIGINT NOT NULL REFERENCES m_members(id) ON DELETE RESTRICT,
    sale_id BIGINT REFERENCES t_sales(id) ON DELETE SET NULL,
    entry_type VARCHAR(20) NOT NULL
        CHECK (entry_type IN ('EARN', 'REDEEM', 'EARN_REVERSAL', 'REDEEM_REVERSAL', 'ADJUST')),
    points BIGINT NOT NULL,
    balance_after BIGINT NOT NULL,
    note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    CONSTRAINT t_member_points_balance_after_non_neg CHECK (balance_after >= 0)
);

CREATE INDEX IF NOT EXISTS idx_t_member_points_member ON t_member_points (member_id);
CREATE INDEX IF NOT EXISTS idx_t_member_points_sale ON t_member_points (sale_id);

-- =========================================================================
-- 4. PERMISSION & GRANT: customers:* -> members:*
-- =========================================================================
INSERT INTO s_permissions (name, description)
SELECT
    replace(p.name, 'customers:', 'members:'),
    replace(p.description, 'pelanggan', 'member')
FROM s_permissions p
WHERE p.name LIKE 'customers:%'
ON CONFLICT (name) DO NOTHING;

-- Salin grant lama agar tidak ada role yang kehilangan akses.
INSERT INTO role_permissions (role_id, permission_id)
SELECT rp.role_id, pnew.id
FROM role_permissions rp
JOIN s_permissions pold ON pold.id = rp.permission_id AND pold.name LIKE 'customers:%'
JOIN s_permissions pnew ON pnew.name = replace(pold.name, 'customers:', 'members:')
ON CONFLICT DO NOTHING;

DELETE FROM role_permissions
WHERE permission_id IN (SELECT id FROM s_permissions WHERE name LIKE 'customers:%');

DELETE FROM s_permissions WHERE name LIKE 'customers:%';
