-- =========================================================================
-- MODUL PENJUALAN / KASIR (POS)
-- =========================================================================
--
-- Modul ini adalah inti penjualan yang dipakai aplikasi kasir (POS). Cakupan
-- tahap pertama:
--   * master pelanggan (m_customers) dengan satu pelanggan umum bawaan;
--   * metode pembayaran (m_payment_methods) yang dipetakan langsung ke akun CoA;
--   * transaksi penjualan (t_sales) + rincian (t_sales_details) + pembayaran
--     (t_sales_payments);
--   * penghitung nomor nota (t_sales_number_counters).
--
-- Penjualan mengurangi stok dan menerbitkan jurnal otomatis (source_type
-- 'SALE'). Pajak per nota (NON_PPN / TAX_EXCLUSIVE / TAX_INCLUSIVE) dan diskon
-- per baris maupun per nota dihitung di usecase.
--
-- Perintah: make migrateup

-- =========================================================================
-- 1. PERLUAS JENIS SUMBER JURNAL
-- =========================================================================
-- 'SALE'  : jurnal penjualan (debit kas/piutang/HPP, kredit pendapatan/PPN/persediaan).
-- 'SALE_PAYMENT' : jurnal pelunasan piutang penjualan (tahap berikutnya).
ALTER TABLE t_journal_entries
    DROP CONSTRAINT IF EXISTS t_journal_entries_source_type_check;

ALTER TABLE t_journal_entries
    ADD CONSTRAINT t_journal_entries_source_type_check CHECK (
        source_type IN (
            'MANUAL', 'GRN', 'SUPPLIER_PAYMENT', 'CASH_IN', 'CASH_OUT',
            'CLOSING', 'SALE', 'SALE_PAYMENT'
        )
    );

-- =========================================================================
-- 2. MASTER PELANGGAN
-- =========================================================================
CREATE TABLE IF NOT EXISTS m_customers (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(30) NOT NULL,
    name VARCHAR(150) NOT NULL,
    phone VARCHAR(30) NOT NULL DEFAULT '',
    email VARCHAR(150) NOT NULL DEFAULT '',
    address TEXT NOT NULL DEFAULT '',
    default_term_days INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    updated_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    deleted_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    CONSTRAINT m_customers_default_term_days_check CHECK (default_term_days >= 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_customers_code_active_unique
    ON m_customers (code) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_m_customers_name ON m_customers (name);

-- Pelanggan umum bawaan untuk penjualan tunai tanpa identitas.
INSERT INTO m_customers (code, name, default_term_days)
SELECT 'UMUM', 'Pelanggan Umum', 0
WHERE NOT EXISTS (
    SELECT 1 FROM m_customers WHERE code = 'UMUM' AND deleted_at IS NULL
);

-- =========================================================================
-- 3. METODE PEMBAYARAN
-- =========================================================================
-- account_id menunjuk langsung ke akun CoA tujuan kas/bank/e-wallet. Metode
-- ber-channel CREDIT tidak dipakai untuk pembayaran tunai (sisa menjadi piutang
-- lewat mapping AR), sehingga account_id boleh NULL.
CREATE TABLE IF NOT EXISTS m_payment_methods (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(30) NOT NULL,
    name VARCHAR(100) NOT NULL,
    channel VARCHAR(20) NOT NULL DEFAULT 'CASH'
        CHECK (channel IN ('CASH', 'BANK', 'EWALLET', 'CREDIT')),
    account_id BIGINT REFERENCES m_chart_of_accounts(id) ON DELETE RESTRICT,
    is_cash BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order INT NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    updated_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    deleted_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_payment_methods_code_active_unique
    ON m_payment_methods (code) WHERE deleted_at IS NULL;

-- Akun e-wallet baru (kode 1-1103 masih kosong; 1-1101/1-1102 sudah dipakai).
INSERT INTO m_chart_of_accounts (code, name, account_type, normal_balance, parent_id, is_header)
SELECT '1-1103', 'E-WALLET', 'ASSET', 'DEBIT', p.id, FALSE
FROM m_chart_of_accounts p
WHERE p.code = '1-1100'
  AND NOT EXISTS (SELECT 1 FROM m_chart_of_accounts WHERE code = '1-1103')
ON CONFLICT DO NOTHING;

-- Metode bawaan. Transfer memakai akun bank yang sudah ada.
INSERT INTO m_payment_methods (code, name, channel, account_id, is_cash, sort_order)
SELECT v.code, v.name, v.channel, a.id, v.is_cash, v.sort_order
FROM (VALUES
    ('TUNAI',    'Tunai',          'CASH',    '1-1101', TRUE,  1),
    ('TF_BCA',   'Transfer BCA',   'BANK',    '1-1201', FALSE, 2),
    ('TF_MDR',   'Transfer Mandiri','BANK',   '1-1202', FALSE, 3),
    ('EWALLET',  'E-Wallet',       'EWALLET', '1-1103', FALSE, 4),
    ('KREDIT',   'Kredit / Tempo', 'CREDIT',  NULL,     FALSE, 5)
) AS v(code, name, channel, coa_code, is_cash, sort_order)
LEFT JOIN m_chart_of_accounts a ON a.code = v.coa_code
WHERE NOT EXISTS (
    SELECT 1 FROM m_payment_methods m WHERE m.code = v.code AND m.deleted_at IS NULL
)
ON CONFLICT DO NOTHING;

-- =========================================================================
-- 4. TRANSAKSI PENJUALAN
-- =========================================================================
CREATE TABLE IF NOT EXISTS t_sales (
    id BIGSERIAL PRIMARY KEY,
    sale_number VARCHAR(50) NOT NULL,
    sale_date DATE NOT NULL DEFAULT CURRENT_DATE,
    customer_id BIGINT NOT NULL REFERENCES m_customers(id) ON DELETE RESTRICT,
    tax_type VARCHAR(20) NOT NULL DEFAULT 'NON_PPN'
        CHECK (tax_type IN ('NON_PPN', 'TAX_EXCLUSIVE', 'TAX_INCLUSIVE')),
    subtotal NUMERIC(15,2) NOT NULL DEFAULT 0,
    item_discount NUMERIC(15,2) NOT NULL DEFAULT 0,
    discount_type VARCHAR(10) NOT NULL DEFAULT 'NONE'
        CHECK (discount_type IN ('NONE', 'PERCENT', 'AMOUNT')),
    discount_value NUMERIC(15,2) NOT NULL DEFAULT 0,
    discount_amount NUMERIC(15,2) NOT NULL DEFAULT 0,
    dpp_amount NUMERIC(15,2) NOT NULL DEFAULT 0,
    tax_amount NUMERIC(15,2) NOT NULL DEFAULT 0,
    total_amount NUMERIC(15,2) NOT NULL DEFAULT 0,
    paid_amount NUMERIC(15,2) NOT NULL DEFAULT 0,
    cost_total NUMERIC(15,2) NOT NULL DEFAULT 0,
    payment_status VARCHAR(10) NOT NULL DEFAULT 'UNPAID'
        CHECK (payment_status IN ('UNPAID', 'PARTIAL', 'PAID')),
    status VARCHAR(10) NOT NULL DEFAULT 'POSTED'
        CHECK (status IN ('POSTED', 'VOID')),
    due_date DATE,
    notes TEXT NOT NULL DEFAULT '',
    journal_entry_id BIGINT REFERENCES t_journal_entries(id) ON DELETE SET NULL,
    voided_at TIMESTAMP WITH TIME ZONE,
    voided_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    void_reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    updated_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    CONSTRAINT t_sales_sale_number_unique UNIQUE (sale_number)
);

CREATE INDEX IF NOT EXISTS idx_t_sales_date ON t_sales (sale_date);
CREATE INDEX IF NOT EXISTS idx_t_sales_customer ON t_sales (customer_id);
CREATE INDEX IF NOT EXISTS idx_t_sales_status ON t_sales (status);

CREATE TABLE IF NOT EXISTS t_sales_details (
    id BIGSERIAL PRIMARY KEY,
    sale_id BIGINT NOT NULL REFERENCES t_sales(id) ON DELETE CASCADE,
    product_id BIGINT NOT NULL REFERENCES m_products(id) ON DELETE RESTRICT,
    sku VARCHAR(50) NOT NULL DEFAULT '',
    product_name VARCHAR(255) NOT NULL DEFAULT '',
    qty INT NOT NULL CHECK (qty > 0),
    unit_price NUMERIC(15,2) NOT NULL DEFAULT 0,
    discount_type VARCHAR(10) NOT NULL DEFAULT 'NONE'
        CHECK (discount_type IN ('NONE', 'PERCENT', 'AMOUNT')),
    discount_value NUMERIC(15,2) NOT NULL DEFAULT 0,
    discount_amount NUMERIC(15,2) NOT NULL DEFAULT 0,
    gross_amount NUMERIC(15,2) NOT NULL DEFAULT 0,
    line_total NUMERIC(15,2) NOT NULL DEFAULT 0,
    cost_unit NUMERIC(15,4) NOT NULL DEFAULT 0,
    cost_total NUMERIC(15,2) NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_t_sales_details_sale ON t_sales_details (sale_id);
CREATE INDEX IF NOT EXISTS idx_t_sales_details_product ON t_sales_details (product_id);

CREATE TABLE IF NOT EXISTS t_sales_payments (
    id BIGSERIAL PRIMARY KEY,
    sale_id BIGINT NOT NULL REFERENCES t_sales(id) ON DELETE CASCADE,
    payment_method_id BIGINT NOT NULL REFERENCES m_payment_methods(id) ON DELETE RESTRICT,
    amount NUMERIC(15,2) NOT NULL DEFAULT 0,
    reference_number VARCHAR(100) NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT t_sales_payments_amount_check CHECK (amount > 0)
);

CREATE INDEX IF NOT EXISTS idx_t_sales_payments_sale ON t_sales_payments (sale_id);

-- =========================================================================
-- 5. PENGHITUNG NOMOR NOTA
-- =========================================================================
CREATE TABLE IF NOT EXISTS t_sales_number_counters (
    prefix VARCHAR(8) PRIMARY KEY,
    last_seq BIGINT NOT NULL DEFAULT 0,
    CONSTRAINT t_sales_number_counters_prefix_check CHECK (prefix ~ '^[0-9]{8}$'),
    CONSTRAINT t_sales_number_counters_seq_check CHECK (last_seq >= 0)
);

-- =========================================================================
-- 6. PERMISSION & GRANT
-- =========================================================================
INSERT INTO s_permissions (name, description)
VALUES
    ('sales:read', 'Melihat data penjualan'),
    ('sales:create', 'Membuat transaksi penjualan'),
    ('sales:void', 'Membatalkan transaksi penjualan'),
    ('customers:read', 'Melihat data pelanggan'),
    ('customers:create', 'Menambah pelanggan'),
    ('customers:update', 'Mengubah pelanggan'),
    ('customers:delete', 'Menghapus pelanggan'),
    ('payment-methods:read', 'Melihat metode pembayaran'),
    ('payment-methods:create', 'Menambah metode pembayaran'),
    ('payment-methods:update', 'Mengubah metode pembayaran'),
    ('payment-methods:delete', 'Menghapus metode pembayaran')
ON CONFLICT (name) DO NOTHING;

-- Admin ERP mendapat seluruh permission baru.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM s_roles r, s_permissions p
WHERE r.name IN ('Super Admin', 'Admin')
  AND p.name IN (
    'sales:read', 'sales:create', 'sales:void',
    'customers:read', 'customers:create', 'customers:update', 'customers:delete',
    'payment-methods:read', 'payment-methods:create', 'payment-methods:update',
    'payment-methods:delete'
  )
ON CONFLICT DO NOTHING;

-- Kasir boleh melihat & membuat penjualan, plus membaca master yang dibutuhkan
-- POS (pelanggan, metode pembayaran, produk lewat pos:read).
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM s_roles r, s_permissions p
WHERE r.name = 'Kasir'
  AND p.name IN ('sales:read', 'sales:create', 'customers:read', 'payment-methods:read')
ON CONFLICT DO NOTHING;
