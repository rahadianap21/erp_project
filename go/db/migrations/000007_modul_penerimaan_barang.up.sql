-- =========================================================================
-- MODUL PENERIMAAN BARANG (GOODS RECEIPT NOTES / GRN)
-- =========================================================================
--
-- Modul ini mencatat penerimaan barang dari PO yang sudah disetujui (APPROVED).
-- Setiap kali gudang menerima barang, dibuat satu dokumen GRN (batch) dengan
-- tanggal dan siapa yang menerima. Setiap batch bersifat immutable karena:
--   * jumlah yang pernah ditambahkan ke stok (dalam satuan jual) harus tetap
--     bisa direproduksi untuk audit,
--   * pembatalan GRN akan membutuhkan pembalikan stok — yang merupakan modul
--     terpisah (tidak termasuk dalam scope v1 ini).
--
-- Struktur data mengikuti pola PO: header + detail, penomoran atomik berbasis
-- prefix tanggal, dan tidak ada soft-delete pada dokumen GRN.
--
-- current_stock dinyatakan dalam SATUAN JUAL (satuan terkecil). Saat menerima
-- barang yang dibeli dalam satuan lain, qty_received dalam satuan beli
-- dikalikan dengan conversion_rate produk untuk mendapatkan delta stok dalam
-- satuan jual. Nilai conversion_rate disimpan sebagai snapshot di detail GRN
-- agar perubahan master produk setelah PO tidak mengubah riwayat penerimaan.

-- 1. Tabel header GRN
CREATE TABLE IF NOT EXISTS t_goods_receipt_notes (
    id BIGSERIAL PRIMARY KEY,
    grn_number VARCHAR(50) NOT NULL UNIQUE,
    purchase_order_id BIGINT NOT NULL,
    received_date DATE NOT NULL DEFAULT CURRENT_DATE,
    total_qty INT NOT NULL DEFAULT 0,
    notes TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    created_by BIGINT DEFAULT NULL,
    updated_by BIGINT DEFAULT NULL,
    CONSTRAINT fk_grn_purchase_order FOREIGN KEY (purchase_order_id)
        REFERENCES t_purchase_orders(id) ON DELETE RESTRICT,
    CONSTRAINT t_goods_receipt_notes_grn_number_check CHECK (grn_number ~ '^(GRN|grn)/[0-9]{8}/[0-9]{4}$'),
    CONSTRAINT t_goods_receipt_notes_total_qty_check CHECK (total_qty >= 0)
);

CREATE INDEX IF NOT EXISTS idx_t_grn_purchase_order_id ON t_goods_receipt_notes (purchase_order_id);
CREATE INDEX IF NOT EXISTS idx_t_grn_received_date ON t_goods_receipt_notes (received_date);
CREATE INDEX IF NOT EXISTS idx_t_grn_grn_number ON t_goods_receipt_notes (grn_number);

-- 2. Tabel detail GRN
CREATE TABLE IF NOT EXISTS t_goods_receipt_note_details (
    id BIGSERIAL PRIMARY KEY,
    grn_id BIGINT NOT NULL,
    purchase_order_detail_id BIGINT NOT NULL,
    product_id BIGINT NOT NULL,
    qty_received INT NOT NULL,
    conversion_rate INT NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_grn_detail_header FOREIGN KEY (grn_id)
        REFERENCES t_goods_receipt_notes(id) ON DELETE CASCADE,
    CONSTRAINT fk_grn_detail_po_detail FOREIGN KEY (purchase_order_detail_id)
        REFERENCES t_purchase_order_details(id) ON DELETE RESTRICT,
    CONSTRAINT fk_grn_detail_product FOREIGN KEY (product_id)
        REFERENCES m_products(id) ON DELETE RESTRICT,
    CONSTRAINT t_grn_detail_qty_received_check CHECK (qty_received > 0),
    CONSTRAINT t_grn_detail_conversion_rate_check CHECK (conversion_rate >= 1)
);

CREATE INDEX IF NOT EXISTS idx_t_grn_detail_grn_id ON t_goods_receipt_note_details (grn_id);
CREATE INDEX IF NOT EXISTS idx_t_grn_detail_po_detail_id ON t_goods_receipt_note_details (purchase_order_detail_id);
CREATE INDEX IF NOT EXISTS idx_t_grn_detail_product_id ON t_goods_receipt_note_details (product_id);

-- 3. Penghitung nomor GRN
CREATE TABLE IF NOT EXISTS t_grn_number_counters (
    prefix VARCHAR(8) PRIMARY KEY,
    last_seq BIGINT NOT NULL DEFAULT 0,
    CONSTRAINT t_grn_number_counters_prefix_check CHECK (prefix ~ '^[0-9]{8}$'),
    CONSTRAINT t_grn_number_counters_seq_check CHECK (last_seq >= 0)
);

-- 4. Hak akses untuk penerimaan barang
INSERT INTO s_permissions (name, description)
VALUES
    ('grn:create', 'Mencatat penerimaan barang dari PO yang sudah disetujui'),
    ('grn:read',   'Melihat daftar dan detail Goods Receipt Notes')
ON CONFLICT (name) DO NOTHING;

-- 5. Berikan ke Super Admin
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM s_roles r, s_permissions p
WHERE r.name = 'Super Admin'
  AND p.name IN ('grn:create', 'grn:read')
ON CONFLICT DO NOTHING;

-- 6. Hak akses untuk menutup penerimaan PO (APPROVED → RECEIVED)
INSERT INTO s_permissions (name, description)
VALUES
    ('po:receive', 'Mengubah status PO menjadi RECEIVED setelah penerimaan barang')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM s_roles r, s_permissions p
WHERE r.name = 'Super Admin'
  AND p.name = 'po:receive'
ON CONFLICT DO NOTHING;
