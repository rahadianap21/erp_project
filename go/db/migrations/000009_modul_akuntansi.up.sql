-- 000009_modul_akuntansi.up.sql
-- Modul Akuntansi: Bagan Akun (CoA), Periode Fiskal, Jurnal Umum double-entry,
-- pemetaan akun untuk auto-posting, dan penghitung nomor jurnal.
--
-- Prinsip yang dipegang seluruh modul akuntansi:
--   1. Setiap jurnal WAJIB balance (total_debit = total_credit), dipaksa oleh
--      CHECK constraint di tabel header dan bukan hanya di lapisan aplikasi.
--   2. Akun boleh diposting hanya jika akun posting (bukan is_header).
--   3. Jurnal hanya boleh diposting ke periode fiskal berstatus OPEN.
--   4. Auto-posting (GRN, kas, penjualan) dijamin idempoten lewat partial
--      unique index (source_type, source_id) WHERE status = 'POSTED'.
--   5. Nomor jurnal memakai penghitung UPSERT per prefiks tanggal (pola yang
--      sama dengan t_po_number_counters / t_grn_number_counters).

-- =========================================================================
-- 1. BAGAN AKUN (CHART OF ACCOUNTS)
-- =========================================================================
CREATE TABLE IF NOT EXISTS m_chart_of_accounts (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(20) NOT NULL,                       -- Contoh: '1-1101'
    name VARCHAR(150) NOT NULL,
    account_type VARCHAR(20) NOT NULL DEFAULT 'ASSET'
        CHECK (account_type IN ('ASSET', 'LIABILITY', 'EQUITY', 'REVENUE', 'EXPENSE')),
    normal_balance VARCHAR(6) NOT NULL DEFAULT 'DEBIT'
        CHECK (normal_balance IN ('DEBIT', 'CREDIT')),
    parent_id BIGINT DEFAULT NULL,                   -- Hierarki akun header/child
    is_header BOOLEAN NOT NULL DEFAULT FALSE,        -- TRUE = akun grup, tak boleh diposting
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    created_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    updated_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    deleted_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    CONSTRAINT fk_coa_parent FOREIGN KEY (parent_id)
        REFERENCES m_chart_of_accounts(id) ON DELETE RESTRICT,
    -- Jaga konsistensi: tipe akun menentukan saldo normal. Aset & Beban
    -- bersaldo normal Debit; Kewajiban, Ekuitas, Pendapatan bersaldo Kredit.
    CONSTRAINT m_chart_of_accounts_balance_check CHECK (
        (account_type IN ('ASSET', 'EXPENSE') AND normal_balance = 'DEBIT')
        OR (account_type IN ('LIABILITY', 'EQUITY', 'REVENUE') AND normal_balance = 'CREDIT')
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_coa_code_active_unique
    ON m_chart_of_accounts (code) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_m_coa_parent_id ON m_chart_of_accounts (parent_id);
CREATE INDEX IF NOT EXISTS idx_m_coa_type ON m_chart_of_accounts (account_type);

INSERT INTO m_chart_of_accounts (code, name, account_type, normal_balance, is_header)
VALUES
    ('1-1000', 'ASET LANCAR',               'ASSET',     'DEBIT',  TRUE),
    ('1-2000', 'ASET TETAP',                'ASSET',     'DEBIT',  TRUE),
    ('2-1000', 'HUTANG LANCAR',             'LIABILITY', 'CREDIT', TRUE),
    ('3-1000', 'MODAL',                     'EQUITY',    'CREDIT', TRUE),
    ('4-1000', 'PENDAPATAN',                'REVENUE',   'CREDIT', TRUE),
    ('5-1000', 'HARGA POKOK PENJUALAN',     'EXPENSE',   'DEBIT',  TRUE),
    ('5-2000', 'BEBAN OPERASIONAL',         'EXPENSE',   'DEBIT',  TRUE)
ON CONFLICT DO NOTHING;

INSERT INTO m_chart_of_accounts (code, name, account_type, normal_balance, parent_id, is_header)
SELECT v.code, v.name, v.type, v.bal, p.id, v.header
FROM (VALUES
    ('1-1100', 'KAS & BANK',             'ASSET',     'DEBIT',  '1-1000', TRUE),
    ('1-1300', 'PIUTANG USAHA',          'ASSET',     'DEBIT',  '1-1000', FALSE),
    ('1-1400', 'PERSEDIAAN BARANG DAGANG','ASSET',    'DEBIT',  '1-1000', FALSE),
    ('1-1500', 'PPN MASUKAN',            'ASSET',     'DEBIT',  '1-1000', FALSE),
    ('1-2100', 'PERALATAN TOKO',         'ASSET',     'DEBIT',  '1-2000', FALSE),
    ('1-2200', 'KENDARAAN',              'ASSET',     'DEBIT',  '1-2000', FALSE),
    ('2-1100', 'HUTANG USAHA',           'LIABILITY', 'CREDIT', '2-1000', FALSE),
    ('2-1200', 'HUTANG PAJAK',           'LIABILITY', 'CREDIT', '2-1000', TRUE),
    ('2-1300', 'HUTANG GAJI',            'LIABILITY', 'CREDIT', '2-1000', FALSE),
    ('3-1100', 'MODAL PEMILIK',          'EQUITY',    'CREDIT', '3-1000', FALSE),
    ('3-1200', 'PRIVE',                  'EQUITY',    'CREDIT', '3-1000', FALSE),
    ('3-1300', 'LABA DITAHAN',           'EQUITY',    'CREDIT', '3-1000', FALSE),
    ('4-1100', 'PENDAPATAN PENJUALAN',   'REVENUE',   'CREDIT', '4-1000', FALSE),
    ('4-1200', 'PENDAPATAN LAIN-LAIN',   'REVENUE',   'CREDIT', '4-1000', FALSE),
    ('5-1100', 'HPP BARANG DAGANGAN',    'EXPENSE',   'DEBIT',  '5-1000', FALSE),
    ('5-2100', 'BEBAN GAJI',             'EXPENSE',   'DEBIT',  '5-2000', FALSE),
    ('5-2200', 'BEBAN SEWA',             'EXPENSE',   'DEBIT',  '5-2000', FALSE),
    ('5-2300', 'BEBAN LISTRIK & AIR',    'EXPENSE',   'DEBIT',  '5-2000', FALSE),
    ('5-2400', 'BEBAN TRANSPORTASI',     'EXPENSE',   'DEBIT',  '5-2000', FALSE),
    ('5-2500', 'BEBAN PERLENGKAPAN',     'EXPENSE',   'DEBIT',  '5-2000', FALSE),
    ('5-2600', 'BEBAN LAIN-LAIN',        'EXPENSE',   'DEBIT',  '5-2000', FALSE)
) AS v(code, name, type, bal, parent_code, header)
JOIN m_chart_of_accounts p ON p.code = v.parent_code
ON CONFLICT DO NOTHING;

INSERT INTO m_chart_of_accounts (code, name, account_type, normal_balance, parent_id, is_header)
SELECT v.code, v.name, v.type, v.bal, p.id, v.header
FROM (VALUES
    ('1-1101', 'KAS TUNAI',             'ASSET',     'DEBIT',  '1-1100', FALSE),
    ('1-1102', 'KAS KECIL',             'ASSET',     'DEBIT',  '1-1100', FALSE),
    ('1-1200', 'REKENING BANK',         'ASSET',     'DEBIT',  '1-1100', TRUE),
    ('2-1201', 'PPN KELUARAN',          'LIABILITY', 'CREDIT', '2-1200', FALSE),
    ('2-1202', 'HUTANG PPH 23',         'LIABILITY', 'CREDIT', '2-1200', FALSE)
) AS v(code, name, type, bal, parent_code, header)
JOIN m_chart_of_accounts p ON p.code = v.parent_code
ON CONFLICT DO NOTHING;

INSERT INTO m_chart_of_accounts (code, name, account_type, normal_balance, parent_id, is_header)
SELECT v.code, v.name, v.type, v.bal, p.id, v.header
FROM (VALUES
    ('1-1201', 'BANK BCA',              'ASSET',     'DEBIT',  '1-1200', FALSE),
    ('1-1202', 'BANK MANDIRI',          'ASSET',     'DEBIT',  '1-1200', FALSE)
) AS v(code, name, type, bal, parent_code, header)
JOIN m_chart_of_accounts p ON p.code = v.parent_code
ON CONFLICT DO NOTHING;

-- =========================================================================
-- 2. PERIODE FISIKAL
-- =========================================================================
CREATE TABLE IF NOT EXISTS m_accounting_periods (
    id BIGSERIAL PRIMARY KEY,
    period_year INT NOT NULL,
    period_month INT NOT NULL,
    status VARCHAR(10) NOT NULL DEFAULT 'OPEN'
        CHECK (status IN ('OPEN', 'CLOSED')),
    closed_at TIMESTAMP WITH TIME ZONE,
    closed_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    updated_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    CONSTRAINT m_accounting_periods_period_unique UNIQUE (period_year, period_month),
    CONSTRAINT m_accounting_periods_month_check CHECK (period_month BETWEEN 1 AND 12),
    CONSTRAINT m_accounting_periods_year_check CHECK (period_year BETWEEN 2000 AND 2100)
);

-- Buka periode bulan berjalan secara otomatis.
INSERT INTO m_accounting_periods (period_year, period_month, status)
SELECT EXTRACT(YEAR FROM CURRENT_DATE)::int,
       EXTRACT(MONTH FROM CURRENT_DATE)::int,
       'OPEN'
ON CONFLICT (period_year, period_month) DO NOTHING;

-- =========================================================================
-- 3. PEMETAAN AKUN UNTUK AUTO-POSTING
-- =========================================================================
-- Menyimpan akun CoA yang dipakai modul transaksi (GRN, kas, penjualan) saat
-- membuat jurnal otomatis. Dipisah dari kode CoA di usecase agar tetap bisa
-- diubah dari database tanpa merombak logika program.
CREATE TABLE IF NOT EXISTS s_accounting_settings (
    mapping_key VARCHAR(30) PRIMARY KEY,
    account_id BIGINT NOT NULL,
    description VARCHAR(200) DEFAULT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    CONSTRAINT fk_acc_settings_account FOREIGN KEY (account_id)
        REFERENCES m_chart_of_accounts(id) ON DELETE RESTRICT,
    CONSTRAINT s_accounting_settings_key_check CHECK (mapping_key IN (
        'INVENTORY', 'AP', 'AR', 'COGS', 'SALES_REVENUE',
        'VAT_INPUT', 'VAT_OUTPUT', 'CASH', 'RETAINED_EARNINGS'
    ))
);

INSERT INTO s_accounting_settings (mapping_key, account_id, description)
SELECT v.key, a.id, v.detail
FROM (VALUES
    ('INVENTORY',         '1-1400', 'Persediaan saat penerimaan barang (GRN)'),
    ('AP',                '2-1100', 'Hutang usaha ke supplier'),
    ('AR',                '1-1300', 'Piutang usaha dari pelanggan'),
    ('COGS',              '5-1100', 'Harga pokok penjualan'),
    ('SALES_REVENUE',     '4-1100', 'Pendapatan penjualan'),
    ('VAT_INPUT',         '1-1500', 'PPN masukan (pembelian)'),
    ('VAT_OUTPUT',        '2-1201', 'PPN keluaran (penjualan)'),
    ('CASH',              '1-1101', 'Akun kas default'),
    ('RETAINED_EARNINGS', '3-1300', 'Laba ditahan saat penutupan tahun')
) AS v(key, coa_code, detail)
JOIN m_chart_of_accounts a ON a.code = v.coa_code
ON CONFLICT (mapping_key) DO NOTHING;

-- =========================================================================
-- 4. JURNAL UMUM (HEADER + LINES)
-- =========================================================================
CREATE TABLE IF NOT EXISTS t_journal_entries (
    id BIGSERIAL PRIMARY KEY,
    entry_number VARCHAR(50) NOT NULL UNIQUE,        -- Format: JV/YYYYMMDD/0001
    entry_date DATE NOT NULL DEFAULT CURRENT_DATE,
    description TEXT,
    source_type VARCHAR(30) DEFAULT NULL
        CHECK (source_type IN ('MANUAL', 'GRN', 'SUPPLIER_PAYMENT', 'CASH_IN', 'CASH_OUT')),
    source_id BIGINT DEFAULT NULL,
    source_reference VARCHAR(50) DEFAULT NULL,       -- Nomor dokumen sumber
    status VARCHAR(10) NOT NULL DEFAULT 'DRAFT'
        CHECK (status IN ('DRAFT', 'POSTED', 'VOID')),
    total_debit NUMERIC(15,2) NOT NULL DEFAULT 0,
    total_credit NUMERIC(15,2) NOT NULL DEFAULT 0,
    period_year INT NOT NULL,
    period_month INT NOT NULL DEFAULT 1,
    posted_at TIMESTAMP DEFAULT NULL,
    posted_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    voided_at TIMESTAMP DEFAULT NULL,
    voided_by BIGINT REFERENCES s_users(id) ON DELETE SET NULL,
    void_reason TEXT,
    -- reversal_of_id dipasang pada jurnal PEMBALIK yang merujuk jurnal asli.
    -- reversed_by_id dipasang pada jurnal ASLI setelah berhasil dibalik.
    -- Jurnal yang sudah POSTED tidak boleh dicoret begitu saja: pembatalannya
    -- wajib lewat jurnal pembalik (double-entry), bukan menimpa status.
    reversal_of_id BIGINT REFERENCES t_journal_entries(id) ON DELETE RESTRICT,
    reversed_by_id BIGINT REFERENCES t_journal_entries(id) ON DELETE SET NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_by BIGINT DEFAULT NULL,
    updated_by BIGINT DEFAULT NULL,
    CONSTRAINT t_journal_entries_totals_check CHECK (total_debit = total_credit),
    CONSTRAINT t_journal_entries_period_month_check CHECK (period_month BETWEEN 1 AND 12),
    CONSTRAINT t_journal_entries_period_year_check CHECK (period_year BETWEEN 2000 AND 2100)
);

CREATE INDEX IF NOT EXISTS idx_t_journal_entries_date ON t_journal_entries (entry_date);
CREATE INDEX IF NOT EXISTS idx_t_journal_entries_period ON t_journal_entries (period_year, period_month);
CREATE INDEX IF NOT EXISTS idx_t_journal_entries_source ON t_journal_entries (source_type, source_id);
CREATE INDEX IF NOT EXISTS idx_t_journal_entries_status ON t_journal_entries (status);

-- Idempotensi auto-posting: kombinasi (source_type, source_id) hanya boleh
-- menghasilkan satu jurnal POSTED. Kalau nanti reposting batch, baris kedua
-- gagal di sini sehingga dokumen tidak pernah dicatat dua kali.
CREATE UNIQUE INDEX IF NOT EXISTS idx_journal_source_unique
    ON t_journal_entries (source_type, source_id)
    WHERE source_type IS NOT NULL AND source_id IS NOT NULL AND status = 'POSTED';

CREATE TABLE IF NOT EXISTS t_journal_entry_lines (
    id BIGSERIAL PRIMARY KEY,
    journal_entry_id BIGINT NOT NULL,
    line_no INT NOT NULL,
    account_id BIGINT NOT NULL,
    description TEXT,
    debit NUMERIC(15,2) NOT NULL DEFAULT 0,
    credit NUMERIC(15,2) NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_jel_journal FOREIGN KEY (journal_entry_id)
        REFERENCES t_journal_entries(id) ON DELETE CASCADE,
    CONSTRAINT fk_jel_account FOREIGN KEY (account_id)
        REFERENCES m_chart_of_accounts(id) ON DELETE RESTRICT,
    CONSTRAINT t_journal_entry_lines_single_side_check
        CHECK (debit >= 0 AND credit >= 0 AND NOT (debit > 0 AND credit > 0)),
    CONSTRAINT t_journal_entry_lines_has_amount_check
        CHECK (debit > 0 OR credit > 0),
    CONSTRAINT t_journal_entry_lines_line_no_unique UNIQUE (journal_entry_id, line_no)
);

CREATE INDEX IF NOT EXISTS idx_t_journal_entry_lines_entry ON t_journal_entry_lines (journal_entry_id);
CREATE INDEX IF NOT EXISTS idx_t_journal_entry_lines_account ON t_journal_entry_lines (account_id);

-- =========================================================================
-- 5. PENGHITUNG NOMOR JURNAL
-- =========================================================================
CREATE TABLE IF NOT EXISTS t_journal_number_counters (
    prefix VARCHAR(8) PRIMARY KEY,
    last_seq BIGINT NOT NULL DEFAULT 0,
    CONSTRAINT t_journal_number_counters_prefix_check CHECK (prefix ~ '^[0-9]{8}$'),
    CONSTRAINT t_journal_number_counters_seq_check CHECK (last_seq >= 0)
);

-- =========================================================================
-- 6. HAK AKSES MODUL AKUNTANSI
-- =========================================================================
INSERT INTO s_permissions (name, description)
VALUES
    ('accounting:read',         'Melihat bagan akun, jurnal, dan laporan akuntansi'),
    ('accounting:create',       'Membuat jurnal umum dan periode fiskal'),
    ('accounting:update',       'Mengubah bagan akun'),
    ('accounting:delete',       'Menghapus bagan akun'),
    ('accounting:post',         'Memposting dan membatalkan jurnal (void/reversal)'),
    ('accounting:close-period', 'Menutup dan membuka periode fiskal')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM s_roles r, s_permissions p
WHERE r.name = 'Super Admin'
  AND p.name IN ('accounting:read', 'accounting:create', 'accounting:update',
                 'accounting:delete', 'accounting:post', 'accounting:close-period')
ON CONFLICT DO NOTHING;