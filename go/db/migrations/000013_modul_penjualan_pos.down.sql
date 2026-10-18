-- =========================================================================
-- PEMBATALAN MODUL PENJUALAN / KASIR (POS)
-- =========================================================================
-- Perintah: make migratedown

-- 1. Cabut permission modul penjualan.
DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM s_permissions
    WHERE name IN (
        'sales:read', 'sales:create', 'sales:void',
        'customers:read', 'customers:create', 'customers:update', 'customers:delete',
        'payment-methods:read', 'payment-methods:create', 'payment-methods:update',
        'payment-methods:delete'
    )
);

DELETE FROM s_permissions
WHERE name IN (
    'sales:read', 'sales:create', 'sales:void',
    'customers:read', 'customers:create', 'customers:update', 'customers:delete',
    'payment-methods:read', 'payment-methods:create', 'payment-methods:update',
    'payment-methods:delete'
);

-- 2. Tabel penjualan (urutan terbalik dari dependensi).
DROP TABLE IF EXISTS t_sales_payments;
DROP TABLE IF EXISTS t_sales_details;
DROP TABLE IF EXISTS t_sales_number_counters;
DROP TABLE IF EXISTS t_sales;
DROP TABLE IF EXISTS m_payment_methods;
DROP TABLE IF EXISTS m_customers;

-- 3. Kembalikan jenis sumber jurnal ke daftar semula.
ALTER TABLE t_journal_entries
    DROP CONSTRAINT IF EXISTS t_journal_entries_source_type_check;

ALTER TABLE t_journal_entries
    ADD CONSTRAINT t_journal_entries_source_type_check CHECK (
        source_type IN ('MANUAL', 'GRN', 'SUPPLIER_PAYMENT', 'CASH_IN', 'CASH_OUT', 'CLOSING')
    );
