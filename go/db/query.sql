-- query.sql

-- =========================================================================
-- 🗂️ MODUL MANAJEMEN PENGGUNA (USERS & AUTENTIKASI)
-- =========================================================================

-- name: CreateUser :one
INSERT INTO s_users (name, email, password, role_id)
VALUES (
    sqlc.arg(name)::text, 
    sqlc.arg(email)::text, 
    sqlc.arg(password)::text, 
    sqlc.arg(role_id)::int
)
RETURNING id, name, email, role_id, is_active, created_at;

-- name: GetUserByEmailWithRole :one
SELECT 
    u.id, u.name, u.email, u.password, u.is_active, u.role_id, r.name AS role_name,
    r.app AS role_app
FROM s_users u
INNER JOIN s_roles r ON u.role_id = r.id
WHERE u.email = sqlc.arg(email)::text AND u.deleted_at IS NULL AND u.is_active = TRUE;

-- name: GetUserByIDWithRole :one
-- Dipakai OtorisasiAktif untuk membaca role dan status aktif user pada setiap
-- request. Sengaja TIDAK memfilter is_active: repository ini juga dipakai admin
-- untuk mengelola akun yang sedang dinonaktifkan, jadi pemeriksaan
-- status aktif dilakukan di lapisan pemanggil, bukan disembunyikan di sini.
-- deleted_at tetap difilter supaya user yang sudah dihapus tidak pernah lolos.
-- COALESCE pada u.name itu wajib, bukan kosmetik. Kolom name nullable di
-- database, sedangkan field Go-nya string. Tanpa COALESCE, satu user yang namanya
-- NULL membuat pemindaian gagal dan request-nya jadi 500. Karena query ini
-- dipanggil di setiap request terproteksi, satu baris NULL bisa mengunci seluruh
-- akses user tersebut.
SELECT 
    u.id, COALESCE(u.name, '') AS name, u.email, u.password, u.is_active, u.role_id, r.name AS role_name,
    r.app AS role_app
FROM s_users u
INNER JOIN s_roles r ON u.role_id = r.id
WHERE u.id = sqlc.arg(id)::bigint AND u.deleted_at IS NULL;

-- name: GetAllUsersPaginated :many
-- 🌟 ONE-TRIP QUERY: List pengguna dan total count didapat simultan lewat WINDOW FUNCTION
SELECT 
    u.id, u.name, u.email, u.password, u.role_id, r.name as role_name,
    COUNT(*) OVER() as total_count
FROM s_users u
JOIN s_roles r ON u.role_id = r.id
WHERE u.deleted_at IS NULL
  AND (sqlc.arg(search_query)::text = '' OR u.name ILIKE '%' || sqlc.arg(search_query)::text || '%' OR u.email ILIKE '%' || sqlc.arg(search_query)::text || '%')
  AND (sqlc.arg(role_filter)::text = 'ALL' OR r.name = sqlc.arg(role_filter)::text)
ORDER BY u.id DESC
LIMIT sqlc.arg(req_limit)::int OFFSET sqlc.arg(req_offset)::int;

-- name: GetUsersCursorNext :many
SELECT 
    u.id, u.name, u.email, u.password, u.role_id, r.name as role_name
FROM s_users u
JOIN s_roles r ON u.role_id = r.id
WHERE u.deleted_at IS NULL
  AND (sqlc.arg(search_query)::text = '' OR u.name ILIKE '%' || sqlc.arg(search_query)::text || '%' OR u.email ILIKE '%' || sqlc.arg(search_query)::text || '%')
  AND (sqlc.arg(role_filter)::text = 'ALL' OR r.name = sqlc.arg(role_filter)::text)
  AND (sqlc.arg(cursor_id)::bigint = 0 OR u.id < sqlc.arg(cursor_id)::bigint)
ORDER BY u.id DESC
LIMIT sqlc.arg(req_limit)::int;

-- name: GetUsersCursorPrev :many
SELECT 
    u.id, u.name, u.email, u.password, u.role_id, r.name as role_name
FROM s_users u
JOIN s_roles r ON u.role_id = r.id
WHERE u.deleted_at IS NULL
  AND (sqlc.arg(search_query)::text = '' OR u.name ILIKE '%' || sqlc.arg(search_query)::text || '%' OR u.email ILIKE '%' || sqlc.arg(search_query)::text || '%')
  AND (sqlc.arg(role_filter)::text = 'ALL' OR r.name = sqlc.arg(role_filter)::text)
  AND u.id > sqlc.arg(cursor_id)::bigint
ORDER BY u.id ASC
LIMIT sqlc.arg(req_limit)::int;

-- name: CountUsers :one
SELECT COUNT(*) as total_count
FROM s_users u
JOIN s_roles r ON u.role_id = r.id
WHERE u.deleted_at IS NULL
  AND (sqlc.arg(search_query)::text = '' OR u.name ILIKE '%' || sqlc.arg(search_query)::text || '%' OR u.email ILIKE '%' || sqlc.arg(search_query)::text || '%')
  AND (sqlc.arg(role_filter)::text = 'ALL' OR r.name = sqlc.arg(role_filter)::text);

-- name: UpdateUser :execrows
-- role_id lama ikut menjadi syarat (compare-and-swap). Pemeriksaan wewenang
-- usecase membaca role target sebelum UPDATE; bila admin lain mengubah role
-- target di antara pembacaan dan penulisan, baris tidak boleh ikut ditimpa.
UPDATE s_users
SET name = $1, email = $2, role_id = $3, updated_at = NOW()
WHERE id = $4 AND deleted_at IS NULL AND role_id = $5;

-- name: DeleteUser :exec
UPDATE s_users
SET deleted_at = NOW(), updated_at = NOW()
WHERE id = $1 AND deleted_at IS NULL;


-- =========================================================================
-- 🔐 MODUL MASTER PERAN & OTORISASI (ROLES & PERMISSIONS)
-- =========================================================================

-- name: GetRoleByName :one
SELECT id, name FROM s_roles WHERE name = $1 AND deleted_at IS NULL LIMIT 1;

-- name: CreateRoleWithReturn :one
INSERT INTO s_roles (name) VALUES ($1) RETURNING id, name;

-- name: GetAllRolesMaster :many
-- 🌟 ONE-TRIP QUERY: List peran, JSON permissions, dan total count didapat simultan
SELECT 
    r.id, r.name,
    COALESCE(
        JSON_AGG(p.name ORDER BY p.name ASC) FILTER (WHERE p.name IS NOT NULL), 
        '[]'::json
    )::jsonb as permissions_json,
    COUNT(*) OVER() as total_count
FROM s_roles r
LEFT JOIN role_permissions rp ON rp.role_id = r.id
LEFT JOIN s_permissions p ON rp.permission_id = p.id
WHERE r.deleted_at IS NULL
  AND r.name ILIKE '%' || sqlc.arg(search_query)::text || '%'
GROUP BY r.id, r.name
ORDER BY r.id ASC
LIMIT sqlc.arg(req_limit)::int OFFSET sqlc.arg(req_offset)::int;

-- name: GetRolesCursorNext :many
SELECT 
    r.id, r.name,
    COALESCE(
        JSON_AGG(p.name ORDER BY p.name ASC) FILTER (WHERE p.name IS NOT NULL), 
        '[]'::json
    )::jsonb as permissions_json
FROM s_roles r
LEFT JOIN role_permissions rp ON rp.role_id = r.id
LEFT JOIN s_permissions p ON rp.permission_id = p.id
WHERE r.deleted_at IS NULL
  AND (sqlc.arg(search_query)::text = '' OR r.name ILIKE '%' || sqlc.arg(search_query)::text || '%')
  AND (sqlc.arg(cursor_id)::int = 0 OR r.id > sqlc.arg(cursor_id)::int)
GROUP BY r.id, r.name
ORDER BY r.id ASC
LIMIT sqlc.arg(req_limit)::int;

-- name: GetRolesCursorPrev :many
SELECT 
    r.id, r.name,
    COALESCE(
        JSON_AGG(p.name ORDER BY p.name ASC) FILTER (WHERE p.name IS NOT NULL), 
        '[]'::json
    )::jsonb as permissions_json
FROM s_roles r
LEFT JOIN role_permissions rp ON rp.role_id = r.id
LEFT JOIN s_permissions p ON rp.permission_id = p.id
WHERE r.deleted_at IS NULL
  AND (sqlc.arg(search_query)::text = '' OR r.name ILIKE '%' || sqlc.arg(search_query)::text || '%')
  AND r.id < sqlc.arg(cursor_id)::int
GROUP BY r.id, r.name
ORDER BY r.id DESC
LIMIT sqlc.arg(req_limit)::int;

-- name: CountRoles :one
SELECT COUNT(*) as total_count
FROM s_roles r
WHERE r.deleted_at IS NULL
  AND (sqlc.arg(search_query)::text = '' OR r.name ILIKE '%' || sqlc.arg(search_query)::text || '%');

-- name: GetAllMasterPermissions :many
SELECT name FROM s_permissions ORDER BY name ASC;

-- name: GetPermissionsByRoleID :many
SELECT p.name 
FROM s_permissions p
INNER JOIN role_permissions rp ON p.id = rp.permission_id
WHERE rp.role_id = sqlc.arg(role_id)::int
ORDER BY p.name ASC;

-- name: AssignPermissionToRole :exec
INSERT INTO role_permissions (role_id, permission_id) 
VALUES ($1, (SELECT id FROM s_permissions WHERE name = $2 LIMIT 1));

-- name: DeleteAllPermissionsFromRole :exec
DELETE FROM role_permissions WHERE role_id = $1;

-- name: DeleteRoleOnly :exec
DELETE FROM s_roles WHERE id = $1;

-- name: UpdateRoleName :exec
UPDATE s_roles SET name = $1 WHERE id = $2;

-- =========================================================================
-- 🗺️ MODUL MASTER REGIONAL WILAYAH INDONESIA (CASCADING API)
-- =========================================================================

-- name: GetProvinces :many
-- Mengambil seluruh daftar provinsi untuk dropdown pertama
SELECT id, name 
FROM m_provinces 
ORDER BY name ASC;

-- name: GetRegenciesByProvinceID :many
-- Mengambil daftar kota/kabupaten berdasarkan ID provinsi yang dipilih
SELECT id, name 
FROM m_regencies 
WHERE province_id = $1 
ORDER BY name ASC;

-- name: GetDistrictsByRegencyID :many
-- Mengambil daftar kecamatan berdasarkan ID kota/kabupaten yang dipilih
SELECT id, name 
FROM m_districts 
WHERE regency_id = $1 
ORDER BY name ASC;

-- name: GetVillagesByDistrictID :many
-- Mengambil daftar kelurahan/desa beserta kode posnya berdasarkan ID kecamatan yang dipilih
SELECT id, name, postal_code 
FROM m_villages 
WHERE district_id = $1 
ORDER BY name ASC;

-- =========================================================================
-- 📦 MODUL DATA MASTER SUPPLIER (CURSOR PAGINATION BASED)
-- =========================================================================

-- name: CreateSupplier :one
INSERT INTO m_suppliers (
    name, pic_name, phone, email, address, village_id, bank_id, account_number, account_name, default_tax_type, created_by, updated_by
) VALUES (
    sqlc.arg(name)::text,
    sqlc.arg(pic_name)::text,
    sqlc.arg(phone)::text,
    sqlc.arg(email)::text,
    sqlc.arg(address)::text,
    sqlc.arg(village_id)::text,
    sqlc.narg(bank_id)::int,         -- 🌟 Kolom Baru
    sqlc.narg(account_number)::text,
    sqlc.narg(account_name)::text,
    sqlc.arg(default_tax_type)::text,
    sqlc.narg(created_by)::bigint,
    sqlc.narg(updated_by)::bigint
)
RETURNING id, name, pic_name, phone, email, address, village_id, default_tax_type, is_active, created_at;

-- name: GetSuppliersCursorNext :many
-- 🌟 CURSOR NEXT: Menarik data halaman berikutnya (ID lebih kecil dari kursor terakhir)
SELECT 
    s.id, s.name, s.pic_name, s.phone, s.email, s.address, s.village_id,
    v.name as village_name, v.postal_code,
    d.id as district_id, d.name as district_name,
    r.id as regency_id, r.name as regency_name,
    p.id as province_id, p.name as province_name,
    s.bank_id, b.kode as bank_kode, b.short_name as bank_name, s.account_number, s.account_name,-- 🌟 LEFT JOIN BANK INFO
    s.default_tax_type,
    s.is_active, s.created_at
FROM m_suppliers s
LEFT JOIN m_villages v ON s.village_id = v.id
LEFT JOIN m_districts d ON v.district_id = d.id
LEFT JOIN m_regencies r ON d.regency_id = r.id
LEFT JOIN m_provinces p ON r.province_id = p.id
LEFT JOIN m_banks b ON s.bank_id = b.id
WHERE s.deleted_at IS NULL
  AND (sqlc.arg(search_query)::text = '' 
       OR s.name ILIKE '%' || sqlc.arg(search_query)::text || '%' 
       OR s.pic_name ILIKE '%' || sqlc.arg(search_query)::text || '%' 
       OR s.email ILIKE '%' || sqlc.arg(search_query)::text || '%')
  AND (sqlc.arg(province_filter)::text = 'ALL' OR p.id = sqlc.arg(province_filter)::text)
  AND (sqlc.arg(cursor_id)::bigint = 0 OR s.id < sqlc.arg(cursor_id)::bigint)
ORDER BY s.id DESC
LIMIT sqlc.arg(req_limit)::int;

-- name: GetSuppliersCursorPrev :many
-- 🌟 CURSOR PREV: Menarik data halaman sebelumnya (ID lebih besar dari kursor teratas)
SELECT 
    s.id, s.name, s.pic_name, s.phone, s.email, s.address, s.village_id,
    v.name as village_name, v.postal_code,
    d.id as district_id, d.name as district_name,
    r.id as regency_id, r.name as regency_name,
    p.id as province_id, p.name as province_name,
    s.bank_id, b.kode as bank_kode, b.short_name as bank_name, s.account_number, s.account_name,-- 🌟 LEFT JOIN BANK INFO
    s.default_tax_type,
    s.is_active, s.created_at
FROM m_suppliers s
LEFT JOIN m_villages v ON s.village_id = v.id
LEFT JOIN m_districts d ON v.district_id = d.id
LEFT JOIN m_regencies r ON d.regency_id = r.id
LEFT JOIN m_provinces p ON r.province_id = p.id
LEFT JOIN m_banks b ON s.bank_id = b.id
WHERE s.deleted_at IS NULL
  AND (sqlc.arg(search_query)::text = '' 
       OR s.name ILIKE '%' || sqlc.arg(search_query)::text || '%' 
       OR s.pic_name ILIKE '%' || sqlc.arg(search_query)::text || '%' 
       OR s.email ILIKE '%' || sqlc.arg(search_query)::text || '%')
  AND (sqlc.arg(province_filter)::text = 'ALL' OR p.id = sqlc.arg(province_filter)::text)
  AND s.id > sqlc.arg(cursor_id)::bigint
ORDER BY s.id ASC
LIMIT sqlc.arg(req_limit)::int;

-- name: CountSuppliers :one
-- Menghitung total data riil supplier untuk kebutuhan info kontrol pagination frontend
SELECT COUNT(*) as total_count
FROM m_suppliers s
LEFT JOIN m_villages v ON s.village_id = v.id
LEFT JOIN m_districts d ON v.district_id = d.id
LEFT JOIN m_regencies r ON d.regency_id = r.id
LEFT JOIN m_provinces p ON r.province_id = p.id
WHERE s.deleted_at IS NULL
  AND (sqlc.arg(search_query)::text = '' 
       OR s.name ILIKE '%' || sqlc.arg(search_query)::text || '%' 
       OR s.pic_name ILIKE '%' || sqlc.arg(search_query)::text || '%' 
       OR s.email ILIKE '%' || sqlc.arg(search_query)::text || '%')
  AND (sqlc.arg(province_filter)::text = 'ALL' OR p.id = sqlc.arg(province_filter)::text);

-- name: GetSupplierByID :one
SELECT 
    s.id, s.name, s.pic_name, s.phone, s.email, s.address, s.village_id,
    v.name as village_name, v.postal_code,
    d.id as district_id, d.name as district_name,
    r.id as regency_id, r.name as regency_name,
    p.id as province_id, p.name as province_name,
    s.bank_id, b.kode as bank_kode, b.short_name as bank_name, s.account_number, s.account_name,-- 🌟 MODIFIKASI: Tambahkan kolom bank info
    s.default_tax_type,
    s.is_active,
    s.created_at
FROM m_suppliers s
LEFT JOIN m_villages v ON s.village_id = v.id
LEFT JOIN m_districts d ON v.district_id = d.id
LEFT JOIN m_regencies r ON d.regency_id = r.id
LEFT JOIN m_provinces p ON r.province_id = p.id
LEFT JOIN m_banks b ON s.bank_id = b.id -- 🌟 MODIFIKASI: Tambahkan LEFT JOIN ke master bank
WHERE s.id = $1 AND s.deleted_at IS NULL;

-- name: UpdateSupplier :exec
UPDATE m_suppliers
SET 
    name = sqlc.arg(name)::text,
    pic_name = sqlc.arg(pic_name)::text,
    phone = sqlc.arg(phone)::text,
    email = sqlc.arg(email)::text,
    address = sqlc.arg(address)::text,
    village_id = sqlc.arg(village_id)::text,
    bank_id = sqlc.narg(bank_id)::int,                 -- 🌟 Kolom Baru
    account_number = sqlc.narg(account_number)::text,
    account_name = sqlc.narg(account_name)::text,
    is_active = sqlc.arg(is_active)::boolean,
    updated_at = NOW(),
    updated_by = sqlc.narg(updated_by)::bigint
WHERE id = sqlc.arg(id)::bigint AND deleted_at IS NULL;

-- name: DeleteSupplier :exec
UPDATE m_suppliers
SET 
    deleted_at = NOW(),
    updated_at = NOW(),
    deleted_by = sqlc.narg(deleted_by)::bigint
WHERE id = sqlc.arg(id)::bigint AND deleted_at IS NULL;

-- =========================================================================
-- 🏦 MODUL MASTER BANK (HTTP SELECT OPTIONS API)
-- =========================================================================

-- name: GetAllActiveBanks :many
-- Mengambil seluruh daftar bank aktif untuk kebutuhan dropdown transaksi finansial
SELECT id, kode, name, short_name 
FROM m_banks 
WHERE deleted_at IS NULL AND is_active = TRUE
ORDER BY short_name ASC;

-- name: GetBankByID :one
SELECT id, kode, name, short_name, is_active
FROM m_banks
WHERE id = $1 AND deleted_at IS NULL;

-- =========================================================================
-- 🗂️ MODUL MASTER KATEGORI PRODUK (CURSOR PAGINATION BASED) - REVISED
-- =========================================================================

-- name: CreateProductCategory :one
INSERT INTO m_product_categories (
    name, slug, description, parent_id, created_by, updated_by
) VALUES (
    sqlc.arg(name)::text,
    sqlc.arg(slug)::text,
    sqlc.narg(description)::text,
    sqlc.narg(parent_id)::int,
    sqlc.narg(created_by)::bigint,
    sqlc.narg(updated_by)::bigint
)
RETURNING id, name, slug, description, parent_id, is_active, created_at;

-- name: GetProductCategoriesCursorNext :many
SELECT 
    c.id, c.name, c.slug, c.description, c.parent_id, -- 🌟 FIX: Ubah 'r.name' menjadi 'c.name'
    p.name as parent_name, c.is_active, c.created_at
FROM m_product_categories c
LEFT JOIN m_product_categories p ON c.parent_id = p.id
WHERE c.deleted_at IS NULL
  AND (sqlc.arg(search_query)::text = '' OR c.name ILIKE '%' || sqlc.arg(search_query)::text || '%')
  AND (sqlc.arg(cursor_id)::int = 0 OR c.id < sqlc.arg(cursor_id)::int)
ORDER BY c.id DESC
LIMIT sqlc.arg(req_limit)::int;

-- name: GetProductCategoriesCursorPrev :many
SELECT 
    c.id, c.name, c.slug, c.description, c.parent_id,
    p.name as parent_name, c.is_active, c.created_at
FROM m_product_categories c
LEFT JOIN m_product_categories p ON c.parent_id = p.id
WHERE c.deleted_at IS NULL
  AND (sqlc.arg(search_query)::text = '' OR c.name ILIKE '%' || sqlc.arg(search_query)::text || '%')
  AND c.id > sqlc.arg(cursor_id)::int
ORDER BY c.id ASC
LIMIT sqlc.arg(req_limit)::int;

-- name: CountProductCategories :one
SELECT COUNT(*) as total_count
FROM m_product_categories c
WHERE c.deleted_at IS NULL
  AND (sqlc.arg(search_query)::text = '' OR c.name ILIKE '%' || sqlc.arg(search_query)::text || '%');

-- name: GetProductCategoryByID :one
SELECT 
    c.id, c.name, c.slug, c.description, c.parent_id,
    p.name as parent_name, c.is_active, c.created_at
FROM m_product_categories c
LEFT JOIN m_product_categories p ON c.parent_id = p.id
WHERE c.id = $1 AND c.deleted_at IS NULL;

-- name: UpdateProductCategory :exec
UPDATE m_product_categories
SET 
    name = sqlc.arg(name)::text,
    slug = sqlc.arg(slug)::text,
    description = sqlc.narg(description)::text,
    parent_id = sqlc.narg(parent_id)::int,
    is_active = sqlc.arg(is_active)::boolean,
    updated_at = NOW(),
    updated_by = sqlc.narg(updated_by)::bigint
WHERE id = sqlc.arg(id)::int AND deleted_at IS NULL; -- 🌟 FIX: Hilangkan prefix alias 'c.' pada klausa WHERE UPDATE karena PostgreSQL tidak mengizinkannya

-- name: DeleteProductCategory :execrows
UPDATE m_product_categories
SET 
    deleted_at = NOW(),
    updated_at = NOW(),
    deleted_by = sqlc.narg(deleted_by)::bigint
WHERE id = sqlc.arg(id)::int AND deleted_at IS NULL; -- 🌟 FIX: Hilangkan prefix alias 'c.' pada klausa WHERE DELETE

-- =========================================================================
-- 📦 MODUL MASTER PRODUK / BARANG (REVISED FOR UNIT RELATIONS)
-- =========================================================================

-- name: CreateProduct :one
INSERT INTO m_products (
    sku, barcode, name, description, category_id, supplier_id, 
    selling_unit_id, purchase_unit_id, -- 🌟 Menggunakan dual unit id
    conversion_rate, -- 🌟 Berapa satuan jual yang didapat dari satu satuan beli
    purchase_price, selling_price, min_stock_level, discount_type, discount_value, is_taxable, current_stock, created_by, updated_by
) VALUES (
    sqlc.arg(sku)::text,
    sqlc.narg(barcode)::text,
    sqlc.arg(name)::text,
    sqlc.narg(description)::text,
    sqlc.arg(category_id)::int,
    sqlc.arg(supplier_id)::bigint,
    sqlc.arg(selling_unit_id)::int,     -- 🌟 Satuan Jual
    sqlc.arg(purchase_unit_id)::int,    -- 🌟 Satuan Beli
    sqlc.arg(conversion_rate)::int,
    sqlc.arg(purchase_price)::numeric,
    sqlc.arg(selling_price)::numeric,
    sqlc.arg(min_stock_level)::int,
    sqlc.arg(discount_type)::varchar,
    sqlc.arg(discount_value)::numeric,
    sqlc.arg(is_taxable)::boolean,
    -- Stok awal boleh diisi saat produk dibuat, tapi hanya di sini. Setelah
    -- produk ada, penulisan stok jadi tugas modul penerimaan barang.
    sqlc.arg(current_stock)::int,
    sqlc.narg(created_by)::bigint,
    sqlc.narg(updated_by)::bigint
)
RETURNING id, sku, name, category_id, supplier_id, selling_unit_id, purchase_unit_id, is_taxable, is_active, created_at;

-- name: GetProductByID :one
SELECT 
    p.id, p.sku, p.barcode, p.name, p.description, p.category_id, p.supplier_id,
    p.selling_unit_id, p.purchase_unit_id, -- 🌟 SINKRONISASI KOLOM BARU
    p.conversion_rate,
    c.name as category_name, s.name as supplier_name,
    u_sell.kode as selling_unit_kode, u_sell.name as selling_unit_name,   -- 🌟 JOIN 1
    u_purch.kode as purchase_unit_kode, u_purch.name as purchase_unit_name, -- 🌟 JOIN 2
    p.purchase_price as prod_purchase_price, 
    p.selling_price as prod_selling_price,   
    p.min_stock_level as prod_min_stock,     
    p.discount_type as prod_discount_type,
    p.discount_value as prod_discount_value,
    p.current_stock as prod_current_stock,
    p.avg_cost as prod_avg_cost,
    p.is_taxable,
    p.is_active, p.created_at
FROM m_products p
LEFT JOIN m_product_categories c ON p.category_id = c.id
LEFT JOIN m_suppliers s ON p.supplier_id = s.id
LEFT JOIN m_units u_sell ON p.selling_unit_id = u_sell.id    -- 🌟 LEFT JOIN 1: Satuan Jual
LEFT JOIN m_units u_purch ON p.purchase_unit_id = u_purch.id  -- 🌟 LEFT JOIN 2: Satuan Beli
WHERE p.id = $1 AND p.deleted_at IS NULL;

-- name: GetProductsByIDs :many
-- Mengambil beberapa produk sekaligus (satu query) untuk alur yang sebelumnya
-- memanggil GetProductByID per baris, misalnya validasi harga PO.
SELECT
    p.id, p.sku, p.barcode, p.name, p.description, p.category_id, p.supplier_id,
    p.selling_unit_id, p.purchase_unit_id,
    p.conversion_rate,
    c.name as category_name, s.name as supplier_name,
    u_sell.kode as selling_unit_kode, u_sell.name as selling_unit_name,
    u_purch.kode as purchase_unit_kode, u_purch.name as purchase_unit_name,
    p.purchase_price as prod_purchase_price,
    p.selling_price as prod_selling_price,
    p.min_stock_level as prod_min_stock,
    p.discount_type as prod_discount_type,
    p.discount_value as prod_discount_value,
    p.current_stock as prod_current_stock,
    p.avg_cost as prod_avg_cost,
    p.is_taxable,
    p.is_active, p.created_at
FROM m_products p
LEFT JOIN m_product_categories c ON p.category_id = c.id
LEFT JOIN m_suppliers s ON p.supplier_id = s.id
LEFT JOIN m_units u_sell ON p.selling_unit_id = u_sell.id
LEFT JOIN m_units u_purch ON p.purchase_unit_id = u_purch.id
WHERE p.id = ANY(sqlc.arg(ids)::bigint[]) AND p.deleted_at IS NULL;

-- name: CountProducts :one
SELECT COUNT(*) as total_count
FROM m_products p
WHERE p.deleted_at IS NULL
  AND (sqlc.arg(search_query)::text = '' 
       OR p.name ILIKE '%' || sqlc.arg(search_query)::text || '%' 
       OR p.sku ILIKE '%' || sqlc.arg(search_query)::text || '%')
  AND (sqlc.arg(category_filter)::int = 0 OR p.category_id = sqlc.arg(category_filter)::int);

-- name: GetProductsCursorNext :many
SELECT 
    p.id, p.sku, p.barcode, p.name, p.description, p.category_id, p.supplier_id,
    p.selling_unit_id, p.purchase_unit_id,
    p.conversion_rate,
    c.name as category_name, c.slug as category_slug,
    s.name as supplier_name, s.pic_name as supplier_pic,
    u_sell.kode as selling_unit_kode, u_sell.name as selling_unit_name,   -- 🌟 Metadata Satuan Jual
    u_purch.kode as purchase_unit_kode, u_purch.name as purchase_unit_name, -- 🌟 Metadata Satuan Beli
    p.purchase_price as prod_purchase_price, 
    p.selling_price as prod_selling_price,   
    p.min_stock_level as prod_min_stock,     
    p.discount_type as prod_discount_type,
    p.discount_value as prod_discount_value,
    p.current_stock as prod_current_stock,   
    p.avg_cost as prod_avg_cost,
    p.is_taxable,
    p.is_active, p.created_at
FROM m_products p
LEFT JOIN m_product_categories c ON p.category_id = c.id
LEFT JOIN m_suppliers s ON p.supplier_id = s.id
LEFT JOIN m_units u_sell ON p.selling_unit_id = u_sell.id    -- 🌟 JOIN 1: Untuk Satuan Jual
LEFT JOIN m_units u_purch ON p.purchase_unit_id = u_purch.id  -- 🌟 JOIN 2: Untuk Satuan Beli
WHERE p.deleted_at IS NULL
  AND (sqlc.arg(search_query)::text = '' 
       OR p.name ILIKE '%' || sqlc.arg(search_query)::text || '%' 
       OR p.sku ILIKE '%' || sqlc.arg(search_query)::text || '%')
  AND (sqlc.arg(category_filter)::int = 0 OR p.category_id = sqlc.arg(category_filter)::int)
  AND (sqlc.arg(cursor_id)::bigint = 0 OR p.id < sqlc.arg(cursor_id)::bigint)
ORDER BY p.id DESC
LIMIT sqlc.arg(req_limit)::int;

-- name: GetProductsCursorPrev :many
SELECT 
    p.id, p.sku, p.barcode, p.name, p.description, p.category_id, p.supplier_id,
    p.selling_unit_id, p.purchase_unit_id,
    p.conversion_rate,
    c.name as category_name, c.slug as category_slug,
    s.name as supplier_name, s.pic_name as supplier_pic,
    u_sell.kode as selling_unit_kode, u_sell.name as selling_unit_name,   -- 🌟 Metadata Satuan Jual
    u_purch.kode as purchase_unit_kode, u_purch.name as purchase_unit_name, -- 🌟 Metadata Satuan Beli
    p.purchase_price as prod_purchase_price, 
    p.selling_price as prod_selling_price,   
    p.min_stock_level as prod_min_stock,     
    p.discount_type as prod_discount_type,
    p.discount_value as prod_discount_value,
    p.current_stock as prod_current_stock,   
    p.avg_cost as prod_avg_cost,
    p.is_taxable,
    p.is_active, p.created_at
FROM m_products p
LEFT JOIN m_product_categories c ON p.category_id = c.id
LEFT JOIN m_suppliers s ON p.supplier_id = s.id
LEFT JOIN m_units u_sell ON p.selling_unit_id = u_sell.id    -- 🌟 JOIN 1: Untuk Satuan Jual
LEFT JOIN m_units u_purch ON p.purchase_unit_id = u_purch.id  -- 🌟 JOIN 2: Untuk Satuan Beli
WHERE p.deleted_at IS NULL
  AND (sqlc.arg(search_query)::text = '' 
       OR p.name ILIKE '%' || sqlc.arg(search_query)::text || '%' 
       OR p.sku ILIKE '%' || sqlc.arg(search_query)::text || '%')
  AND (sqlc.arg(category_filter)::int = 0 OR p.category_id = sqlc.arg(category_filter)::int)
  -- 🌟 AMANKAN TYPE CASTING: Pastikan parameter cursor_id dilapisi kaku ::bigint agar sqlc tidak crash
  AND p.id > sqlc.arg(cursor_id)::bigint 
ORDER BY p.id ASC
LIMIT sqlc.arg(req_limit)::int;

-- name: UpdateProduct :exec
UPDATE m_products
SET 
    sku = sqlc.arg(sku)::text,
    barcode = sqlc.narg(barcode)::text,
    name = sqlc.arg(name)::text,
    description = sqlc.narg(description)::text,
    category_id = sqlc.arg(category_id)::int,
    supplier_id = sqlc.arg(supplier_id)::bigint,
    selling_unit_id = sqlc.arg(selling_unit_id)::int,   -- 🌟 Update Satuan Jual
    purchase_unit_id = sqlc.arg(purchase_unit_id)::int, -- 🌟 Update Satuan Beli
    conversion_rate = sqlc.arg(conversion_rate)::int,
    purchase_price = sqlc.arg(purchase_price)::numeric,
    selling_price = sqlc.arg(selling_price)::numeric,
    min_stock_level = sqlc.arg(min_stock_level)::int,
    discount_type = sqlc.arg(discount_type)::varchar,
    discount_value = sqlc.arg(discount_value)::numeric,
    is_taxable = sqlc.arg(is_taxable)::boolean,
    -- current_stock SENGAJA TIDAK ditulis di sini.
    --
    -- Kolom itu adalah hasil penerimaan barang: setiap kenaikan stok punya
    -- dokumen GRN yang menjadi bukti. Kalau UpdateProduct masih bisa menimpanya,
    -- ada dua penulis untuk satu kolom yang sama dan angka hasil hitungan bisa
    -- hilang tanpa jejak. Penulis tunggalnya sekarang modul penerimaan barang.
    --
    -- Stok awal produk baru tetap bisa diisi, tapi hanya sekali saat produk
    -- dibuat lewat CreateProduct.
    is_active = sqlc.arg(is_active)::boolean,
    updated_at = NOW(),
    updated_by = sqlc.narg(updated_by)::bigint
WHERE id = sqlc.arg(id)::bigint AND deleted_at IS NULL;

-- name: DeleteProduct :exec
UPDATE m_products
SET 
    deleted_at = NOW(),
    updated_at = NOW(),
    deleted_by = sqlc.narg(deleted_by)::bigint
WHERE id = sqlc.arg(id)::bigint AND deleted_at IS NULL;

-- =========================================================================
-- 📦 MODUL DATA MASTER SATUAN / UOM (SELECT OPTIONS API)
-- =========================================================================

-- name: GetAllActiveUnits :many
-- Mengambil seluruh daftar satuan aktif untuk kebutuhan dropdown form barang
SELECT id, kode, name, description 
FROM m_units 
WHERE deleted_at IS NULL AND is_active = TRUE
ORDER BY kode ASC;

-- name: GetUnitByID :one
SELECT id, kode, name, description, is_active
FROM m_units
WHERE id = $1 AND deleted_at IS NULL;

-- name: CreateUnit :one
INSERT INTO m_units (
    kode, name, description, created_by, updated_by
) VALUES (
    sqlc.arg(kode)::text,
    sqlc.arg(name)::text,
    sqlc.arg(description)::text,
    sqlc.narg(created_by)::bigint,
    sqlc.narg(updated_by)::bigint
)
RETURNING id, kode, name, description, is_active, created_at;

-- =========================================================================
-- 🛒 MODUL TRANSAKSIONAL PURCHASE ORDER (PO) - CURSOR PAGINATION BASED
-- =========================================================================

-- name: CreatePurchaseOrderHeader :one
INSERT INTO t_purchase_orders (
    po_number, supplier_id, po_date, tax_type, dpp_amount, tax_amount, total_amount, status, notes, created_by, updated_by
) VALUES (
    sqlc.arg(po_number)::text,
    sqlc.arg(supplier_id)::bigint,
    sqlc.arg(po_date)::date,
    sqlc.arg(tax_type)::text,       -- 🌟 Simpan Tipe Pajak PO
    sqlc.arg(dpp_amount)::numeric,   -- 🌟 Simpan Nilai DPP Netto
    sqlc.arg(tax_amount)::numeric,   -- 🌟 Simpan Nilai PPN 12%
    sqlc.arg(total_amount)::numeric,
    sqlc.arg(status)::text,
    sqlc.narg(notes)::text,
    sqlc.narg(created_by)::bigint,
    sqlc.narg(updated_by)::bigint
)
RETURNING id, po_number, supplier_id, total_amount, status, created_at;

-- name: GetPurchaseOrdersCursorNext :many
SELECT 
    po.id, po.po_number, po.po_date, 
    po.tax_type, po.dpp_amount, po.tax_amount, po.total_amount, -- 🌟 SUNTIKAN WAJIB 1: Pastikan list cursor menarik kolom pajak!
    po.status, po.notes,
    po.approved_by, po.approved_at, po.rejected_by, po.rejected_at, po.reject_reason,
    po.supplier_id AS po_supplier_id,
    s.name as supplier_name, s.pic_name as supplier_pic, s.phone as supplier_phone,
    po.created_at
FROM t_purchase_orders po
LEFT JOIN m_suppliers s ON po.supplier_id = s.id
WHERE po.deleted_at IS NULL
  AND (sqlc.arg(search_query)::text = '' 
       OR po.po_number ILIKE '%' || sqlc.arg(search_query)::text || '%' 
       OR s.name ILIKE '%' || sqlc.arg(search_query)::text || '%')
  AND (sqlc.arg(status_filter)::text = 'ALL' OR po.status = sqlc.arg(status_filter)::text)
  AND (sqlc.arg(cursor_id)::bigint = 0 OR po.id < sqlc.arg(cursor_id)::bigint)
ORDER BY po.id DESC
LIMIT sqlc.arg(req_limit)::int;

-- name: GetPurchaseOrdersCursorPrev :many
SELECT 
    po.id, po.po_number, po.po_date, 
    po.tax_type, po.dpp_amount, po.tax_amount, po.total_amount, -- 🌟 SUNTIKAN WAJIB 2: Pastikan list cursor menarik kolom pajak!
    po.status, po.notes,
    po.approved_by, po.approved_at, po.rejected_by, po.rejected_at, po.reject_reason,
    po.supplier_id AS po_supplier_id,
    s.name as supplier_name, s.pic_name as supplier_pic, s.phone as supplier_phone,
    po.created_at
FROM t_purchase_orders po
LEFT JOIN m_suppliers s ON po.supplier_id = s.id
WHERE po.deleted_at IS NULL
  AND (sqlc.arg(search_query)::text = '' 
       OR po.po_number ILIKE '%' || sqlc.arg(search_query)::text || '%' 
       OR s.name ILIKE '%' || sqlc.arg(search_query)::text || '%')
  AND (sqlc.arg(status_filter)::text = 'ALL' OR po.status = sqlc.arg(status_filter)::text)
  AND po.id > sqlc.arg(cursor_id)::bigint
ORDER BY po.id ASC
LIMIT sqlc.arg(req_limit)::int;

-- name: CountPurchaseOrders :one
SELECT COUNT(*) as total_count
FROM t_purchase_orders po
LEFT JOIN m_suppliers s ON po.supplier_id = s.id
WHERE po.deleted_at IS NULL
  AND (sqlc.arg(search_query)::text = '' 
       OR po.po_number ILIKE '%' || sqlc.arg(search_query)::text || '%' 
       OR s.name ILIKE '%' || sqlc.arg(search_query)::text || '%')
  AND (sqlc.arg(status_filter)::text = 'ALL' OR po.status = sqlc.arg(status_filter)::text);

-- name: GetPurchaseOrderByID :one
SELECT 
    po.id, po.po_number, po.po_date, 
    po.tax_type, po.dpp_amount, po.tax_amount, po.total_amount, -- 🌟 KUNCI AMAN: Tarik metadata pajak transaksional
    po.status, po.notes,
    po.approved_by, po.approved_at, po.rejected_by, po.rejected_at, po.reject_reason,
    po.supplier_id AS supplier_id,
    s.name as supplier_name, s.pic_name as supplier_pic, s.phone as supplier_phone, s.email as supplier_email,
    po.created_at
FROM t_purchase_orders po
LEFT JOIN m_suppliers s ON po.supplier_id = s.id
WHERE po.id = $1 AND po.deleted_at IS NULL;

-- name: GetPurchaseOrderDetailItems :many
--
-- conversion_rate ikut diambil supaya frontend dapat menampilkan dampaknya ke
-- stok (dalam satuan jual) sebelum pengguna konfirmasi penerimaan, bukan
-- hanya setelah GRN tersimpan.
SELECT 
    d.id, d.purchase_order_id, d.product_id, p.name as product_name, p.sku as product_sku,
    d.qty_ordered, d.qty_received, d.purchase_unit_id, u.kode as unit_kode,
    p.conversion_rate, p.selling_unit_id, su.kode as selling_unit_kode,
    d.unit_price, d.sub_total
FROM t_purchase_order_details d
JOIN m_products p ON d.product_id = p.id
JOIN m_units u ON d.purchase_unit_id = u.id
LEFT JOIN m_units su ON su.id = p.selling_unit_id
WHERE d.purchase_order_id = $1
ORDER BY d.id ASC;

-- name: TransitionPurchaseOrderStatus :execrows
-- Perpindahan status memakai compare-and-swap: baris hanya berubah kalau status
-- saat ini masih termasuk daftar asal yang diizinkan, dan bukan salah satu
-- status yang sedang dikunci untuk Transitions lain.
--
-- Versi lama membaca status lewat SELECT lalu menulis tanpa syarat status, jadi
-- approve dan cancel yang datang bersamaan bisa sama-sama berhasil dan dokumen
-- diam-diam berakhir pada status penulisan terakhir. nowrows() di lapisan
-- pemanggil menerjemahkan 0 baris terpengaruh menjadi ErrKonflikStatus, bukan
-- diam-diam dianggap berhasil.
--
-- allow_from dipakai saat transisi hanya boleh dari satu status (mis. approve
-- hanya dari PENDING). forbid_from dipakai untuk mengunci status yang sedang
--Competing: hanya dikirim saat transisi punya status asal tunggal.
UPDATE t_purchase_orders
SET
    status = sqlc.arg(new_status)::text,
    updated_at = NOW(),
    updated_by = sqlc.narg(updated_by)::bigint,
    approved_by = CASE WHEN sqlc.arg(new_status)::text = 'APPROVED' THEN sqlc.narg(updated_by)::bigint ELSE approved_by END,
    approved_at = CASE WHEN sqlc.arg(new_status)::text = 'APPROVED' THEN NOW() ELSE approved_at END,
    rejected_by = CASE WHEN sqlc.arg(new_status)::text = 'REJECTED' THEN sqlc.narg(updated_by)::bigint ELSE rejected_by END,
    rejected_at = CASE WHEN sqlc.arg(new_status)::text = 'REJECTED' THEN NOW() ELSE rejected_at END,
    reject_reason = CASE WHEN sqlc.arg(new_status)::text = 'REJECTED' THEN sqlc.narg(reject_reason)::text ELSE reject_reason END
WHERE id = sqlc.arg(id)::bigint
  AND deleted_at IS NULL
  AND status = ANY(sqlc.arg(allow_from)::text[])
  AND NOT (status = ANY(sqlc.arg(forbid_from)::text[]));

-- name: NextPurchaseOrderSeq :one
-- Meminta nomor dokumen berikutnya secara atomik.
--
-- Ini menggantikan COUNT(*)+1 terhadap tabel PO. Operasi ini aman terhadap
-- permintaan bersamaan karena PostgreSQL mengunci baris prefiks yang sama
-- selama transaksi, sehingga setiap pemanggil pasti mendapat angka berbeda dan
-- tidak ada yang bisa membaca nilai lalu memakainya bersamaan.
--
-- Pemanggil wajib membungkus pemakaiannya bersama INSERT PO dalam satu
-- transaksi: bila INSERT gagal karena alasan lain, pemanggil harus mengembalikan
-- last_seq (lihat DownPurchaseOrderSeq) supaya nomor yang tidak jadi terpakai
-- tidak pernah menggandakan nomor yang sah.
INSERT INTO t_po_number_counters (prefix, last_seq)
VALUES (sqlc.arg(prefix)::varchar, 1)
ON CONFLICT (prefix) DO UPDATE SET last_seq = t_po_number_counters.last_seq + 1
RETURNING last_seq;

-- name: DownPurchaseOrderSeq :exec
-- Membatalkan pemakaian satu nomor: dipakai ketika INSERT PO gagal di tengah
-- transaksi sehingga nomor tadi tidak pernah benar-benar terbit. Dipanggil dalam
-- transaksi yang sama, jadi kalau transaksi 전체 digulung, pengurangan ini juga
-- hilang bersama transaction.
UPDATE t_po_number_counters
SET last_seq = last_seq - 1
WHERE prefix = sqlc.arg(prefix)::varchar AND last_seq > 0;

-- name: CreatePurchaseOrderDetail :exec
INSERT INTO t_purchase_order_details (
    purchase_order_id, product_id, qty_ordered, purchase_unit_id, unit_price, sub_total
) VALUES (
    sqlc.arg(purchase_order_id)::bigint,
    sqlc.arg(product_id)::bigint,
    sqlc.arg(qty_ordered)::int,
    sqlc.arg(purchase_unit_id)::int,
    sqlc.arg(unit_price)::numeric,
    sqlc.arg(sub_total)::numeric
);

-- name: LockPurchaseOrderForUpdate :one
-- Mengunci header PO sebelum koreksi/pencabutan. Lock ini diserialkan dengan
-- approve (TransitionPurchaseOrderStatus), pembuatan GRN, dan penutupan
-- penerimaan yang semuanya juga mengunci baris header yang sama.
--
-- Tanpa lock ini, usecase bisa membaca PENDING lalu request approve menang
-- balapan, tetapi UPDATE detail tetap menimpa dokumen yang sudah APPROVED.
SELECT id, status
FROM t_purchase_orders
WHERE id = sqlc.arg(id)::bigint AND deleted_at IS NULL
FOR UPDATE;

-- name: UpdatePurchaseOrderHeader :one
-- Koreksi header PO. supplier_id SENGAJA tidak disentuh: di form edit kolom
-- supplier terkunci (rekanan tidak boleh dialihkan setelah PO terbit), jadi
-- kueri ini tidak menerima parameter supplier sama sekali.
--
-- Syarat status=PENDING diulang di sini sebagai pagar terakhir. Pemeriksaan
-- utama dilakukan setelah LockPurchaseOrderForUpdate, tetapi menulis syaratnya
-- juga di statement UPDATE membuat jalur ini tetap aman kalau suatu saat ada
-- pemanggil baru yang lupa mengunci header lebih dulu.
UPDATE t_purchase_orders
SET
    tax_type     = sqlc.arg(tax_type)::text,
    dpp_amount   = sqlc.arg(dpp_amount)::numeric,
    tax_amount   = sqlc.arg(tax_amount)::numeric,
    total_amount = sqlc.arg(total_amount)::numeric,
    notes        = sqlc.narg(notes)::text,
    updated_at   = NOW(),
    updated_by   = sqlc.narg(updated_by)::bigint
WHERE id = sqlc.arg(id)::bigint
  AND deleted_at IS NULL
  AND status = 'PENDING'
RETURNING id, po_number, supplier_id, tax_type, dpp_amount, tax_amount, total_amount, status, updated_at;

-- name: DeletePurchaseOrderDetails :exec
-- Hapus seluruh baris detail milik satu PO. Dipakai pada alur koreksi: detail
-- diganti utuh (delete + insert) di dalam satu transaksi. Detail GRN
-- mereferensikan baris ini dengan ON DELETE RESTRICT, jadi koreksi baru aman
-- setelah repository memverifikasi qty_received = 0 (artinya belum ada GRN).
DELETE FROM t_purchase_order_details
WHERE purchase_order_id = sqlc.arg(purchase_order_id)::bigint;

-- name: CountPurchaseOrderReceivedQty :one
-- Menghitung total kuantitas yang sudah diterima pada satu PO. Dipakai sebagai
-- pengaman sebelum koreksi: bila nilainya > 0, detail tidak boleh dihapus.
SELECT COALESCE(SUM(d.qty_received), 0)::bigint AS total_received
FROM t_purchase_order_details d
JOIN t_purchase_orders po ON po.id = d.purchase_order_id
WHERE d.purchase_order_id = sqlc.arg(purchase_order_id)::bigint
  AND po.deleted_at IS NULL;

-- name: SoftDeletePurchaseOrder :execrows
-- Pencabutan dokumen PO memakai soft delete: baris TIDAK dihapus fisik, hanya
-- diisi deleted_at. Seluruh query pembacaan PO sudah memfilter deleted_at IS NULL,
-- jadi dokumen ini langsung hilang dari daftar tanpa merusak riwayat keuangan.
-- Baris detail sengaja dibiarkan agar jejak audit barang tetap utuh.
--
-- Syarat status=PENDING adalah pagar terakhir setelah header dikunci. Dokumen
-- yang sudah APPROVED/RECEIVED tidak boleh hilang dari daftar aktif.
UPDATE t_purchase_orders
SET
    deleted_at = NOW(),
    updated_at = NOW(),
    updated_by = sqlc.narg(updated_by)::bigint
WHERE id = sqlc.arg(id)::bigint
  AND deleted_at IS NULL
  AND status = 'PENDING';

-- =========================================================================
-- LOOKUP KHUSUS FORM PURCHASE ORDER (Endpoint Ringan, Tanpa RBAC Modul Master)
-- =========================================================================
-- Form PO butuh daftar supplier & barang untuk dropdown, tapi tidak boleh
-- bergantung pada permission 'suppliers:read' / 'products:read'. Endpoint ini
-- hanya butuh 'po:read' supaya operator purchasing bisa menerbitkan dokumen
-- tanpa hak akses penuh ke modul master.
--
-- Query di bawah SENGAJA tanpa cursor pagination dan tanpa filter search:
-- tujuannya kirim SELURUH kandidat sekali jalan supaya nilai yang sedang
-- disunting di form edit tidak pernah hilang dari daftar opsi. Lookup lama
-- memakai ORDER BY id DESC + LIMIT sehingga record lama hilang dari dropdown.

-- name: LookupSuppliersForPO :many
-- Supplier aktif untuk dropdown form PO, diurutkan nama agar mudah dipindai
SELECT
    s.id,
    s.name,
    s.default_tax_type
FROM m_suppliers s
WHERE s.deleted_at IS NULL
  AND s.is_active = TRUE
ORDER BY s.name ASC;

-- name: LookupProductsForPO :many
-- Barang untuk dropdown form PO. Kalau supplier_id = 0 maka seluruh katalog
-- ditarik, selain itu hanya barang milik supplier tersebut. LEFT JOIN ke
-- m_units agar barang tetap terkirim walau satuan belinya belum terlampir.
SELECT
    p.id,
    p.sku,
    p.name,
    p.supplier_id,
    p.purchase_price,
    u.kode AS purchase_unit_kode
FROM m_products p
LEFT JOIN m_units u ON p.purchase_unit_id = u.id
WHERE p.deleted_at IS NULL
  AND p.is_active = TRUE
  AND (sqlc.arg(supplier_id)::bigint = 0 OR p.supplier_id = sqlc.arg(supplier_id)::bigint)
ORDER BY p.name ASC;


-- =========================================================================
-- 📦 MODUL PENERIMAAN BARANG (GOODS RECEIPT NOTES / GRN)
-- =========================================================================

-- name: NextGRNSeq :one
-- Meminta nomor urut GRN berikutnya secara atomik berdasarkan prefiks tanggal.
--
-- Pola yang sama dengan PO: menggunakan tabel counter dengan UPSERT agar aman
-- terhadap permintaan bersamaan. Pemanggil wajib membungkus pemakaian ini
-- bersama INSERT GRN dalam satu transaksi. Jika INSERT gagal, pemanggil dapat
-- menekan DownGRNSeq dalam transaksi yang sama untuk mengembalikan counter.
INSERT INTO t_grn_number_counters (prefix, last_seq)
VALUES (sqlc.arg(prefix)::varchar, 1)
ON CONFLICT (prefix) DO UPDATE SET last_seq = t_grn_number_counters.last_seq + 1
RETURNING last_seq;

-- name: DownGRNSeq :exec
-- Membatalkan pemakaian satu nomor GRN ketika INSERT gagal di tengah transaksi.
UPDATE t_grn_number_counters
SET last_seq = last_seq - 1
WHERE prefix = sqlc.arg(prefix)::varchar AND last_seq > 0;

-- name: LockPurchaseOrderForReceiving :one
-- Mengunci header PO untuk menghindari dua GRN bersamaan yang menerima barang
-- pada PO yang sama. Transaksi yang lebih cepat akan berhasil, yang lain akan
-- menunggu lock dan kemudian validasi akan menolak jika jumlah melebihi sisa.
SELECT id, po_number, status
FROM t_purchase_orders
WHERE id = sqlc.arg(id)::bigint AND deleted_at IS NULL
FOR UPDATE;

-- name: GetPurchaseOrderDetailLines :many
-- Mengambil seluruh baris detail PO beserta info produk dan satuan untuk proses
-- penerimaan barang. Digunakan untuk validasi sebelum membuat GRN.
SELECT
    pod.id,
    pod.purchase_order_id,
    pod.product_id,
    pod.qty_ordered,
    pod.qty_received,
    pod.purchase_unit_id,
    u.kode AS purchase_unit_kode,
    p.selling_unit_id,
    us.kode AS selling_unit_kode,
    p.purchase_unit_id AS product_purchase_unit_id,
    p.selling_unit_id AS product_selling_unit_id,
    p.conversion_rate
FROM t_purchase_order_details pod
JOIN m_products p ON p.id = pod.product_id
JOIN m_units u ON u.id = pod.purchase_unit_id
LEFT JOIN m_units us ON us.id = p.selling_unit_id
WHERE pod.purchase_order_id = sqlc.arg(purchase_order_id)::bigint
ORDER BY pod.id ASC;

-- name: IncrementDetailQtyReceived :one
-- Menambah qty_received pada baris detail PO, dengan penjaga over-receipt.
-- Klausa WHERE memastikan jumlah baru tidak melebihi qty_ordered. Jika 0 baris
-- yang terpengaruh, berarti melebihi sisa penerimaan — harus diterjemahkan
-- menjadi error bisnis (422).
UPDATE t_purchase_order_details
SET qty_received = qty_received + sqlc.arg(qty_to_add)::int
WHERE id = sqlc.arg(id)::bigint
  AND qty_received + sqlc.arg(qty_to_add)::int <= qty_ordered
RETURNING qty_received;

-- name: ApplyPurchaseOrderReceivedStock :one
-- Menerapkan seluruh delta stok sebuah PO dalam SATU pernyataan UPDATE.
--
-- Sebelumnya repository membaca GetPOReceivedDeltas lalu memanggil
-- IncrementProductStock per produk, sehingga satu PO dengan N produk menulis
-- N+1 query. Dengan agregat sebagai sumber join, satu UPDATE ... FROM
-- menambahkan stok semua produk sekaligus dan tetap menghitung harga pokok
-- rata-rata pada baris yang sama:
--
--     avg_cost = (stok_lama * avg_cost_lama + nilai_diterima) / stok_baru
--
-- expected_count vs updated_count dipakai pemanggil untuk mendeteksi produk
-- yang hilang/soft-delete; dulu kondisi itu tampak sebagai pgx.ErrNoRows dari
-- IncrementProductStock sehingga penutupan penerimaan ikut batal.
WITH deltas AS (
    SELECT
        gd.product_id,
        SUM(gd.qty_received * gd.conversion_rate)::int AS delta_selling,
        SUM(gd.qty_received * pod.unit_price * sqlc.arg(net_cost_factor)::numeric)::numeric AS net_value
    FROM t_goods_receipt_note_details gd
    JOIN t_goods_receipt_notes g ON g.id = gd.grn_id
    JOIN t_purchase_order_details pod ON pod.id = gd.purchase_order_detail_id
    WHERE g.purchase_order_id = sqlc.arg(purchase_order_id)::bigint
    GROUP BY gd.product_id
    HAVING SUM(gd.qty_received * gd.conversion_rate) > 0
),
updated AS (
    UPDATE m_products p
    SET current_stock = p.current_stock + d.delta_selling,
        avg_cost = CASE
            WHEN p.current_stock + d.delta_selling = 0 THEN 0
            ELSE (p.current_stock * p.avg_cost + d.net_value)
                 / (p.current_stock + d.delta_selling)
        END,
        updated_at = NOW(),
        updated_by = sqlc.narg(updated_by)::bigint
    FROM deltas d
    WHERE p.id = d.product_id AND p.deleted_at IS NULL
    RETURNING p.id
)
SELECT
    (SELECT COUNT(*) FROM deltas)::bigint AS expected_count,
    (SELECT COUNT(*) FROM updated)::bigint AS updated_count;

-- name: CreateGRN :one
INSERT INTO t_goods_receipt_notes (
    grn_number,
    purchase_order_id,
    received_date,
    total_qty,
    notes,
    created_by,
    updated_by
) VALUES (
    sqlc.arg(grn_number)::varchar,
    sqlc.arg(purchase_order_id)::bigint,
    sqlc.arg(received_date)::date,
    sqlc.arg(total_qty)::int,
    sqlc.narg(notes)::text,
    sqlc.narg(created_by)::bigint,
    sqlc.narg(updated_by)::bigint
)
RETURNING id, grn_number, purchase_order_id, received_date, total_qty, created_at;

-- name: IncrementDetailQtyReceivedBatch :one
-- Menambah qty_received semua baris PO dalam SATU UPDATE ... FROM. qty
-- dijumlahkan per baris lebih dulu agar baris ganda tetap setara dengan
-- penambahan berurutan. Penjaga over-receipt dan qty_to_add > 0 tetap
-- dipertahankan; selisih expected_count/updated_count berarti ada baris yang
-- melebihi sisa atau qty-nya tidak positif.
WITH items AS (
    SELECT p.purchase_order_detail_id, q.qty_to_add
    FROM unnest(sqlc.arg(purchase_order_detail_ids)::bigint[]) WITH ORDINALITY AS p(purchase_order_detail_id, ord)
    JOIN unnest(sqlc.arg(qtys_to_add)::int[]) WITH ORDINALITY AS q(qty_to_add, ord) USING (ord)
),
grouped AS (
    SELECT purchase_order_detail_id, SUM(qty_to_add)::int AS qty_to_add
    FROM items
    GROUP BY purchase_order_detail_id
),
updated AS (
    UPDATE t_purchase_order_details d
    SET qty_received = d.qty_received + g.qty_to_add
    FROM grouped g
    WHERE d.id = g.purchase_order_detail_id
      AND g.qty_to_add > 0
      AND d.qty_received + g.qty_to_add <= d.qty_ordered
    RETURNING d.id
)
SELECT
    (SELECT COUNT(*) FROM grouped)::bigint AS expected_count,
    (SELECT COUNT(*) FROM updated)::bigint AS updated_count;

-- name: CreateGRNDetails :exec
-- Menyisipkan seluruh baris GRN dalam SATU INSERT ... SELECT, bukan satu
-- INSERT per baris. Urutan array menentukan pasangan kolom lewat ORDINALITY.
WITH items AS (
    SELECT p.purchase_order_detail_id, q.product_id, r.qty_received, c.conversion_rate
    FROM unnest(sqlc.arg(purchase_order_detail_ids)::bigint[]) WITH ORDINALITY AS p(purchase_order_detail_id, ord)
    JOIN unnest(sqlc.arg(product_ids)::bigint[]) WITH ORDINALITY AS q(product_id, ord) USING (ord)
    JOIN unnest(sqlc.arg(qty_receiveds)::int[]) WITH ORDINALITY AS r(qty_received, ord) USING (ord)
    JOIN unnest(sqlc.arg(conversion_rates)::int[]) WITH ORDINALITY AS c(conversion_rate, ord) USING (ord)
)
INSERT INTO t_goods_receipt_note_details (
    grn_id,
    purchase_order_detail_id,
    product_id,
    qty_received,
    conversion_rate
)
SELECT
    sqlc.arg(grn_id)::bigint,
    purchase_order_detail_id,
    product_id,
    qty_received,
    conversion_rate
FROM items;

-- name: CountGRNsTotal :one
SELECT COUNT(*) AS total_count
FROM t_goods_receipt_notes g
JOIN t_purchase_orders po ON po.id = g.purchase_order_id
WHERE (sqlc.arg(search_query)::text = ''
       OR g.grn_number ILIKE '%' || sqlc.arg(search_query)::text || '%'
       OR po.po_number ILIKE '%' || sqlc.arg(search_query)::text || '%'
       OR CAST(g.purchase_order_id AS TEXT) ILIKE '%' || sqlc.arg(search_query)::text || '%');

-- name: GetGRNsCursorNext :many
SELECT
    g.id,
    g.grn_number,
    g.purchase_order_id,
    g.received_date,
    g.total_qty,
    g.notes,
    g.created_at,
    g.created_by,
    po.po_number
FROM t_goods_receipt_notes g
JOIN t_purchase_orders po ON po.id = g.purchase_order_id
WHERE (sqlc.arg(search_query)::text = ''
       OR g.grn_number ILIKE '%' || sqlc.arg(search_query)::text || '%'
       OR po.po_number ILIKE '%' || sqlc.arg(search_query)::text || '%'
       OR CAST(g.purchase_order_id AS TEXT) ILIKE '%' || sqlc.arg(search_query)::text || '%')
  AND (sqlc.arg(cursor_id)::bigint = 0 OR g.id < sqlc.arg(cursor_id)::bigint)
ORDER BY g.id DESC
LIMIT sqlc.arg(req_limit)::int;

-- name: GetGRNsCursorPrev :many
SELECT
    g.id,
    g.grn_number,
    g.purchase_order_id,
    g.received_date,
    g.total_qty,
    g.notes,
    g.created_at,
    g.created_by,
    po.po_number
FROM t_goods_receipt_notes g
JOIN t_purchase_orders po ON po.id = g.purchase_order_id
WHERE (sqlc.arg(search_query)::text = ''
       OR g.grn_number ILIKE '%' || sqlc.arg(search_query)::text || '%'
       OR po.po_number ILIKE '%' || sqlc.arg(search_query)::text || '%'
       OR CAST(g.purchase_order_id AS TEXT) ILIKE '%' || sqlc.arg(search_query)::text || '%')
  AND g.id > sqlc.arg(cursor_id)::bigint
ORDER BY g.id ASC
LIMIT sqlc.arg(req_limit)::int;

-- name: GetGRNByID :one
SELECT
    g.id,
    g.grn_number,
    g.purchase_order_id,
    g.received_date,
    g.total_qty,
    g.notes,
    g.created_at,
    g.created_by,
    po.po_number
FROM t_goods_receipt_notes g
JOIN t_purchase_orders po ON po.id = g.purchase_order_id
WHERE g.id = sqlc.arg(id)::bigint;

-- name: GetGRNDetails :many
SELECT
    gd.id,
    gd.grn_id,
    gd.purchase_order_detail_id,
    gd.product_id,
    gd.qty_received,
    gd.conversion_rate,
    gd.created_at,
    p.sku,
    p.name AS product_name,
    u.kode AS purchase_unit_kode
FROM t_goods_receipt_note_details gd
JOIN m_products p ON p.id = gd.product_id
JOIN m_units u ON u.id = p.purchase_unit_id
WHERE gd.grn_id = sqlc.arg(grn_id)::bigint
ORDER BY gd.id ASC;

-- name: GetGRNDetailsByPOID :many
-- Mengambil detail SELURUH GRN milik satu PO dalam satu query. Dipakai saat
-- menutup penerimaan supaya tidak perlu memanggil GetGRNDetails per GRN.
SELECT
    gd.id,
    gd.grn_id,
    gd.purchase_order_detail_id,
    gd.product_id,
    gd.qty_received,
    gd.conversion_rate,
    gd.created_at,
    p.sku,
    p.name AS product_name,
    u.kode AS purchase_unit_kode
FROM t_goods_receipt_note_details gd
JOIN t_goods_receipt_notes g ON g.id = gd.grn_id
JOIN m_products p ON p.id = gd.product_id
JOIN m_units u ON u.id = p.purchase_unit_id
WHERE g.purchase_order_id = sqlc.arg(purchase_order_id)::bigint
ORDER BY gd.grn_id ASC, gd.id ASC;

-- name: LockProductsForReceiving :many
-- Mengunci produk yang stoknya akan ditambah saat menutup penerimaan, dalam
-- urutan id yang deterministik. Penguncian eksplisit ini menjaga agar dua
-- penutupan PO yang memuat produk sama selalu meminta lock dalam urutan yang
-- sama, sehingga tidak saling menunggu (deadlock). Bulk UPDATE setelahnya boleh
-- memakai urutan apa pun karena barisnya sudah terkunci di sini.
SELECT p.id
FROM m_products p
JOIN (
    SELECT DISTINCT gd.product_id
    FROM t_goods_receipt_note_details gd
    JOIN t_goods_receipt_notes g ON g.id = gd.grn_id
    WHERE g.purchase_order_id = sqlc.arg(purchase_order_id)::bigint
) d ON d.product_id = p.id
WHERE p.deleted_at IS NULL
ORDER BY p.id ASC
FOR UPDATE OF p;

-- name: CountGRNsByPOID :one
-- Menghitung GRN yang menempel pada satu PO. Dipakai saat MENUTUP penerimaan
-- untuk memastikan daftar GRN yang jurnalnya sudah diterbitkan usecase masih
-- sama dengan daftar yang stoknya akan ditambahkan repository.
--
-- Tanpa pemeriksaan ini, GRN baru yang dibuat setelah usecase membaca daftar
-- tetapi sebelum repository mengunci header akan ikut dijumlahkan stoknya
-- padahal jurnalnya belum pernah diterbitkan. Lock header membuat GRN baru
-- setelah pemeriksaan ini menunggu sampai transaksi selesai, lalu ditolak
-- karena status PO sudah RECEIVED.
SELECT COUNT(*)::bigint AS total
FROM t_goods_receipt_notes g
WHERE g.purchase_order_id = sqlc.arg(purchase_order_id)::bigint;

-- name: GetGRNsByPOID :many
SELECT
    g.id,
    g.grn_number,
    g.purchase_order_id,
    g.received_date,
    g.total_qty,
    g.notes,
    g.created_at,
    g.created_by
FROM t_goods_receipt_notes g
WHERE g.purchase_order_id = sqlc.arg(purchase_order_id)::bigint
ORDER BY g.id DESC;

-- =========================================================================
-- 📊 MODUL DASHBOARD (AGREGAT READ-ONLY)
-- =========================================================================

-- name: GetDashboardStats :one
-- Ringkasan angka untuk kartu KPI dashboard dalam satu perjalanan query.
-- total_po_value sengaja mengecualikan PO REJECTED karena dokumen itu batal.
-- Nilai uang tetap NUMERIC::numeric agar dipindai sebagai pgtype.Numeric dan
-- dikonversi ke domain.Money tanpa melewati float64.
SELECT
    COALESCE((SELECT SUM(total_amount) FROM t_purchase_orders
              WHERE deleted_at IS NULL AND status <> 'REJECTED'), 0)::numeric AS total_po_value,
    (SELECT COUNT(*) FROM t_purchase_orders WHERE deleted_at IS NULL) AS total_po,
    (SELECT COUNT(*) FROM t_purchase_orders WHERE deleted_at IS NULL AND status = 'PENDING') AS po_pending,
    (SELECT COUNT(*) FROM t_purchase_orders WHERE deleted_at IS NULL AND status = 'APPROVED') AS po_approved,
    (SELECT COUNT(*) FROM t_purchase_orders WHERE deleted_at IS NULL AND status = 'RECEIVED') AS po_received,
    (SELECT COUNT(*) FROM t_goods_receipt_notes) AS total_grn,
    (SELECT COUNT(*) FROM m_products WHERE deleted_at IS NULL AND is_active = TRUE) AS total_products,
    (SELECT COUNT(*) FROM m_suppliers WHERE deleted_at IS NULL AND is_active = TRUE) AS total_suppliers,
    (SELECT COUNT(*) FROM m_products
     WHERE deleted_at IS NULL AND is_active = TRUE AND current_stock <= min_stock_level) AS low_stock_count;

-- name: GetDashboardLowStock :many
-- Produk aktif yang stoknya sudah menyentuh atau di bawah ambang reorder.
-- Diurutkan dari defisit paling dalam (current_stock - min_stock_level) agar
-- item yang paling kritis muncul lebih dulu.
SELECT
    p.id,
    p.sku,
    p.name,
    p.current_stock,
    p.min_stock_level,
    COALESCE(u.kode, '')::text AS selling_unit_kode
FROM m_products p
LEFT JOIN m_units u ON u.id = p.selling_unit_id
WHERE p.deleted_at IS NULL
  AND p.is_active = TRUE
  AND p.current_stock <= p.min_stock_level
ORDER BY (p.current_stock - p.min_stock_level) ASC, p.name ASC
LIMIT sqlc.arg(req_limit)::int;

-- name: GetDashboardRecentPOs :many
SELECT
    po.id,
    po.po_number,
    s.name AS supplier_name,
    po.total_amount::numeric AS total_amount,
    po.status,
    po.po_date
FROM t_purchase_orders po
JOIN m_suppliers s ON s.id = po.supplier_id
WHERE po.deleted_at IS NULL
ORDER BY po.id DESC
LIMIT sqlc.arg(req_limit)::int;

-- name: GetDashboardRecentGRNs :many
SELECT
    g.id,
    g.grn_number,
    po.po_number,
    g.total_qty,
    g.received_date
FROM t_goods_receipt_notes g
JOIN t_purchase_orders po ON po.id = g.purchase_order_id
ORDER BY g.id DESC
LIMIT sqlc.arg(req_limit)::int;

-- =========================================================================
-- MODUL AKUNTANSI (FASE 1)
-- =========================================================================

-- name: CreateChartOfAccount :one
INSERT INTO m_chart_of_accounts (
    code, name, account_type, normal_balance, parent_id, is_header,
    created_by, updated_by
) VALUES (
    sqlc.arg(code)::varchar,
    sqlc.arg(name)::varchar,
    sqlc.arg(account_type)::text,
    sqlc.arg(normal_balance)::text,
    sqlc.narg(parent_id)::bigint,
    sqlc.arg(is_header)::boolean,
    sqlc.narg(created_by)::bigint,
    sqlc.narg(updated_by)::bigint
)
RETURNING id, code, name, account_type, normal_balance, parent_id, is_header, is_active, created_at, updated_at;

-- name: GetChartOfAccountsCursorNext :many
SELECT
    a.id, a.code, a.name, a.account_type, a.normal_balance,
    a.parent_id, a.is_header, a.is_active,
    a.created_at, a.updated_at, a.deleted_at,
    p.code AS parent_code, p.name AS parent_name
FROM m_chart_of_accounts a
LEFT JOIN m_chart_of_accounts p ON p.id = a.parent_id
WHERE a.deleted_at IS NULL
  AND (sqlc.arg(search_query)::text = ''
       OR a.code ILIKE '%' || sqlc.arg(search_query)::text || '%'
       OR a.name ILIKE '%' || sqlc.arg(search_query)::text || '%')
  AND (sqlc.arg(cursor_id)::bigint = 0 OR a.id < sqlc.arg(cursor_id)::bigint)
ORDER BY a.id DESC
LIMIT sqlc.arg(req_limit)::int;

-- name: GetChartOfAccountsCursorPrev :many
SELECT
    a.id, a.code, a.name, a.account_type, a.normal_balance,
    a.parent_id, a.is_header, a.is_active,
    a.created_at, a.updated_at, a.deleted_at,
    p.code AS parent_code, p.name AS parent_name
FROM m_chart_of_accounts a
LEFT JOIN m_chart_of_accounts p ON p.id = a.parent_id
WHERE a.deleted_at IS NULL
  AND (sqlc.arg(search_query)::text = ''
       OR a.code ILIKE '%' || sqlc.arg(search_query)::text || '%'
       OR a.name ILIKE '%' || sqlc.arg(search_query)::text || '%')
  AND a.id > sqlc.arg(cursor_id)::bigint
ORDER BY a.id ASC
LIMIT sqlc.arg(req_limit)::int;

-- name: CountChartOfAccounts :one
SELECT COUNT(*) AS total_count
FROM m_chart_of_accounts a
WHERE a.deleted_at IS NULL
  AND (sqlc.arg(search_query)::text = ''
       OR a.code ILIKE '%' || sqlc.arg(search_query)::text || '%'
       OR a.name ILIKE '%' || sqlc.arg(search_query)::text || '%');

-- name: GetChartOfAccountByID :one
SELECT
    a.id, a.code, a.name, a.account_type, a.normal_balance,
    a.parent_id, a.is_header, a.is_active,
    a.created_at, a.updated_at, a.deleted_at,
    p.code AS parent_code, p.name AS parent_name
FROM m_chart_of_accounts a
LEFT JOIN m_chart_of_accounts p ON p.id = a.parent_id
WHERE a.id = sqlc.arg(id)::bigint AND a.deleted_at IS NULL;

-- name: UpdateChartOfAccount :exec
UPDATE m_chart_of_accounts
SET code = sqlc.arg(code)::varchar,
    name = sqlc.arg(name)::varchar,
    account_type = sqlc.arg(account_type)::text,
    normal_balance = sqlc.arg(normal_balance)::text,
    parent_id = sqlc.narg(parent_id)::bigint,
    is_header = sqlc.arg(is_header)::boolean,
    is_active = sqlc.arg(is_active)::boolean,
    updated_at = NOW(),
    updated_by = sqlc.narg(updated_by)::bigint
WHERE id = sqlc.arg(id)::bigint AND deleted_at IS NULL;

-- name: DeleteChartOfAccount :exec
UPDATE m_chart_of_accounts
SET deleted_at = NOW(),
    updated_at = NOW(),
    deleted_by = sqlc.narg(deleted_by)::bigint
WHERE id = sqlc.arg(id)::bigint AND deleted_at IS NULL;

-- name: GetChartOfAccountsForPosting :many
SELECT id, code, parent_id, is_header, is_active, account_type, normal_balance
FROM m_chart_of_accounts
WHERE id = ANY(sqlc.arg(ids)::bigint[]) AND deleted_at IS NULL;

-- name: GetAccountPeriod :one
SELECT id, period_year, period_month, status, closed_at, closed_by, created_at, updated_at
FROM m_accounting_periods
WHERE period_year = sqlc.arg(period_year)::int AND period_month = sqlc.arg(period_month)::int;

-- name: GetAccountPeriodForUpdate :one
-- Mengunci periode untuk operasi tutup/buka buku. Semua penulisan jurnal pada
-- periode ini memakai FOR SHARE, sehingga tidak bisa melewati status CLOSED.
SELECT id, period_year, period_month, status, closed_at, closed_by, created_at, updated_at
FROM m_accounting_periods
WHERE period_year = sqlc.arg(period_year)::int
  AND period_month = sqlc.arg(period_month)::int
FOR UPDATE;

-- name: GetAccountPeriodForShare :one
-- Dipakai saat jurnal POSTED/pembalik akan ditulis. Kunci ini kompatibel
-- dengan beberapa penulis paralel, tetapi menunggu transaksi tutup buku yang
-- memegang FOR UPDATE.
SELECT id, period_year, period_month, status, closed_at, closed_by, created_at, updated_at
FROM m_accounting_periods
WHERE period_year = sqlc.arg(period_year)::int
  AND period_month = sqlc.arg(period_month)::int
FOR SHARE;

-- name: LockJournalEntryPeriodForShare :one
-- Mengunci periode milik jurnal DRAFT yang akan diposting. Baris jurnal sendiri
-- tidak dikunci di sini karena UPDATE ... status = 'DRAFT' menjadi CAS-nya.
SELECT p.id, p.period_year, p.period_month, p.status, p.closed_at, p.closed_by,
       p.created_at, p.updated_at
FROM t_journal_entries j
JOIN m_accounting_periods p
  ON p.period_year = j.period_year
 AND p.period_month = j.period_month
WHERE j.id = sqlc.arg(id)::bigint
  AND p.status = 'OPEN'
FOR SHARE OF p;

-- name: CreateAccountPeriod :one
INSERT INTO m_accounting_periods (period_year, period_month, status, created_by, updated_by)
VALUES (
    sqlc.arg(period_year)::int,
    sqlc.arg(period_month)::int,
    'OPEN',
    sqlc.narg(created_by)::bigint,
    sqlc.narg(updated_by)::bigint
)
ON CONFLICT (period_year, period_month) DO UPDATE
    SET updated_at = NOW()
RETURNING id, period_year, period_month, status, closed_at, closed_by, created_at, updated_at;

-- name: CloseAccountPeriod :execrows
UPDATE m_accounting_periods
SET status = 'CLOSED',
    closed_at = NOW(),
    closed_by = sqlc.arg(actor_user_id)::bigint,
    updated_at = NOW()
WHERE period_year = sqlc.arg(period_year)::int
  AND period_month = sqlc.arg(period_month)::int
  AND status = 'OPEN';

-- name: ReopenAccountPeriod :execrows
UPDATE m_accounting_periods
SET status = 'OPEN',
    closed_at = NULL,
    closed_by = NULL,
    updated_at = NOW()
WHERE period_year = sqlc.arg(period_year)::int
  AND period_month = sqlc.arg(period_month)::int
  AND status = 'CLOSED';

-- name: ListAccountPeriods :many
SELECT id, period_year, period_month, status, closed_at, closed_by, created_at, updated_at
FROM m_accounting_periods
ORDER BY period_year DESC, period_month DESC
LIMIT sqlc.arg(req_limit)::int;

-- name: CreateJournalEntryHeader :one
INSERT INTO t_journal_entries (
    entry_number, entry_date, description, source_type, source_id, source_reference,
    status, total_debit, total_credit, period_year, period_month, reversal_of_id,
    created_by, updated_by
) VALUES (
    sqlc.arg(entry_number)::varchar,
    sqlc.arg(entry_date)::date,
    sqlc.narg(description)::text,
    sqlc.narg(source_type)::text,
    sqlc.narg(source_id)::bigint,
    sqlc.narg(source_reference)::varchar,
    sqlc.arg(status)::text,
    sqlc.arg(total_debit)::numeric,
    sqlc.arg(total_credit)::numeric,
    sqlc.arg(period_year)::int,
    sqlc.arg(period_month)::int,
    sqlc.narg(reversal_of_id)::bigint,
    sqlc.narg(created_by)::bigint,
    sqlc.narg(updated_by)::bigint
)
RETURNING id, entry_number, entry_date, description, status, total_debit, total_credit,
          period_year, period_month, posted_at, posted_by, voided_at, voided_by,
          void_reason, reversal_of_id, reversed_by_id, created_at, updated_at, created_by;

-- name: CreateJournalEntryLine :exec
INSERT INTO t_journal_entry_lines (journal_entry_id, line_no, account_id, description, debit, credit)
VALUES (
    sqlc.arg(journal_entry_id)::bigint,
    sqlc.arg(line_no)::int,
    sqlc.arg(account_id)::bigint,
    sqlc.narg(description)::text,
    sqlc.arg(debit)::numeric,
    sqlc.arg(credit)::numeric
);

-- name: NextJournalSeq :one
-- Nomor urut berikutnya untuk prefiks tanggal (pola sama dengan PO/GRN).
INSERT INTO t_journal_number_counters (prefix, last_seq)
VALUES (sqlc.arg(prefix)::varchar, 1)
ON CONFLICT (prefix) DO UPDATE SET last_seq = t_journal_number_counters.last_seq + 1
RETURNING last_seq;

-- name: DownJournalSeq :exec
-- Mengembalikan nomor yang tidak jadi terpakai ketika INSERT jurnal gagal
-- di tengah transaksi.
UPDATE t_journal_number_counters
SET last_seq = last_seq - 1
WHERE prefix = sqlc.arg(prefix)::varchar AND last_seq > 0;

-- name: GetJournalEntriesCursorNext :many
SELECT
    j.id, j.entry_number, j.entry_date, j.description, j.source_type, j.source_id,
    j.source_reference, j.status, j.total_debit, j.total_credit,
    j.period_year, j.period_month, j.posted_at, j.voided_at, j.reversal_of_id,
    j.reversed_by_id, j.created_at, j.updated_at, j.created_by
FROM t_journal_entries j
WHERE (sqlc.arg(search_query)::text = ''
       OR j.entry_number ILIKE '%' || sqlc.arg(search_query)::text || '%'
       OR j.description ILIKE '%' || sqlc.arg(search_query)::text || '%'
       OR j.source_reference ILIKE '%' || sqlc.arg(search_query)::text || '%')
  AND (sqlc.arg(status_filter)::text = 'ALL' OR j.status = sqlc.arg(status_filter)::text)
  AND (sqlc.arg(cursor_id)::bigint = 0 OR j.id < sqlc.arg(cursor_id)::bigint)
ORDER BY j.id DESC
LIMIT sqlc.arg(req_limit)::int;

-- name: GetJournalEntriesCursorPrev :many
SELECT
    j.id, j.entry_number, j.entry_date, j.description, j.source_type, j.source_id,
    j.source_reference, j.status, j.total_debit, j.total_credit,
    j.period_year, j.period_month, j.posted_at, j.voided_at, j.reversal_of_id,
    j.reversed_by_id, j.created_at, j.updated_at, j.created_by
FROM t_journal_entries j
WHERE (sqlc.arg(search_query)::text = ''
       OR j.entry_number ILIKE '%' || sqlc.arg(search_query)::text || '%'
       OR j.description ILIKE '%' || sqlc.arg(search_query)::text || '%'
       OR j.source_reference ILIKE '%' || sqlc.arg(search_query)::text || '%')
  AND (sqlc.arg(status_filter)::text = 'ALL' OR j.status = sqlc.arg(status_filter)::text)
  AND j.id > sqlc.arg(cursor_id)::bigint
ORDER BY j.id ASC
LIMIT sqlc.arg(req_limit)::int;

-- name: CountJournalEntries :one
SELECT COUNT(*) AS total_count
FROM t_journal_entries j
WHERE (sqlc.arg(search_query)::text = ''
       OR j.entry_number ILIKE '%' || sqlc.arg(search_query)::text || '%'
       OR j.description ILIKE '%' || sqlc.arg(search_query)::text || '%'
       OR j.source_reference ILIKE '%' || sqlc.arg(search_query)::text || '%')
  AND (sqlc.arg(status_filter)::text = 'ALL' OR j.status = sqlc.arg(status_filter)::text);

-- name: GetJournalEntryByID :one
SELECT
    j.id, j.entry_number, j.entry_date, j.description, j.source_type, j.source_id,
    j.source_reference, j.status, j.total_debit, j.total_credit,
    j.period_year, j.period_month, j.posted_at, j.voided_at, j.reversal_of_id,
    j.reversed_by_id, j.created_at, j.updated_at, j.created_by
FROM t_journal_entries j
WHERE j.id = sqlc.arg(id)::bigint;

-- name: GetJournalEntryLines :many
SELECT
    l.id, l.journal_entry_id, l.line_no, l.account_id, l.description, l.debit, l.credit,
    a.code AS account_code, a.name AS account_name
FROM t_journal_entry_lines l
JOIN m_chart_of_accounts a ON a.id = l.account_id
WHERE l.journal_entry_id = sqlc.arg(journal_entry_id)::bigint
ORDER BY l.line_no ASC;

-- name: PostJournalEntry :execrows
-- Posting memakai compare-and-swap terhadap status DRAFT dan menolak bila:
--   * status berubah (bukan DRAFT lagi, misal sudah diposting orang lain);
--   * periode sudah CLOSED;
--   * ada baris yang menunjuk akun header/non-aktif/terhapus;
--   * jurnal kosong (total_debit = 0).
-- Semua syarat dievaluasi dalam satu statement sehingga dua request posting
-- paralel tidak bisa dua-duanya berhasil.
UPDATE t_journal_entries j
SET status = 'POSTED',
    posted_at = NOW(),
    posted_by = sqlc.arg(actor_user_id)::bigint,
    updated_at = NOW()
FROM m_accounting_periods p
WHERE j.id = sqlc.arg(id)::bigint
  AND j.status = 'DRAFT'
  AND j.total_debit > 0
  AND p.period_year = j.period_year
  AND p.period_month = j.period_month
  AND p.status = 'OPEN'
  AND NOT EXISTS (
      SELECT 1 FROM t_journal_entry_lines l
      JOIN m_chart_of_accounts a ON a.id = l.account_id
      WHERE l.journal_entry_id = j.id
        AND (a.is_header OR a.is_active = FALSE OR a.deleted_at IS NOT NULL)
  );

-- name: VoidJournalEntry :execrows
-- Membatalkan jurnal yang masih DRAFT (belum pernah diposting). Jurnal yang
-- sudah POSTED tidak boleh lewat jalur ini: pembatalannya wajib jurnal
-- pembalik (reversal).
UPDATE t_journal_entries
SET status = 'VOID',
    voided_at = NOW(),
    voided_by = sqlc.arg(actor_user_id)::bigint,
    void_reason = sqlc.narg(void_reason)::text,
    updated_at = NOW()
WHERE id = sqlc.arg(id)::bigint
  AND status = 'DRAFT';

-- name: SetOriginalJournalReversed :execrows
-- Mencatat pada jurnal asli bahwa ia sudah dibatalkan oleh jurnal pembalik.
-- Guard reversed_by_id IS NULL + status POSTED menjadikan dua permintaan
-- pembalikan paralel hanya satu yang menang; sisanya RowAffected = 0.
UPDATE t_journal_entries
SET reversed_by_id = sqlc.arg(reversal_id)::bigint,
    updated_at = NOW()
WHERE id = sqlc.arg(id)::bigint
  AND reversed_by_id IS NULL
  AND status = 'POSTED';

-- name: ListAccountingSettings :many
-- Seluruh pemetaan kunci transaksi ke akun CoA untuk auto-posting. Pemetaan
-- sengaja dibaca dari tabel supaya akun tujuan bisa diubah dari database tanpa
-- merombak logika program.
SELECT mapping_key::text, account_id, description
FROM s_accounting_settings
ORDER BY mapping_key;

-- name: ListAccountingSettingDetails :many
-- Pemetaan akun otomatis beserta kode/nama akun untuk halaman pengaturan.
SELECT
    s.mapping_key::text,
    s.description,
    s.account_id,
    a.code AS account_code,
    a.name AS account_name,
    a.is_active AS account_active
FROM s_accounting_settings s
JOIN m_chart_of_accounts a ON a.id = s.account_id
ORDER BY s.mapping_key;

-- name: UpsertAccountingSetting :exec
-- Mengubah akun tujuan satu kunci pemetaan; updated_by diisi user penyimpan.
INSERT INTO s_accounting_settings (mapping_key, account_id, updated_at, updated_by)
VALUES (
    sqlc.arg(mapping_key)::varchar,
    sqlc.arg(account_id)::bigint,
    NOW(),
    sqlc.arg(updated_by)::bigint
)
ON CONFLICT (mapping_key) DO UPDATE SET
    account_id = EXCLUDED.account_id,
    updated_at = NOW(),
    updated_by = EXCLUDED.updated_by;

-- =========================================================================
-- ⚙️ PENGATURAN APLIKASI UMUM (KEY-VALUE)
-- =========================================================================

-- name: ListAppSettings :many
-- Seluruh pengaturan aplikasi (mis. PPN_RATE, PPN_PRICE_MODE).
SELECT key, value, description, updated_at, updated_by
FROM s_app_settings
ORDER BY key;

-- name: GetAppSetting :one
SELECT key, value, description, updated_at, updated_by
FROM s_app_settings
WHERE key = $1;

-- name: UpsertAppSetting :exec
-- Menyimpan satu pengaturan; bila key belum ada maka dibuat.
INSERT INTO s_app_settings (key, value, description, updated_at, updated_by)
VALUES (
    sqlc.arg(key)::varchar,
    sqlc.arg(value)::text,
    sqlc.arg(description)::text,
    NOW(),
    sqlc.narg(updated_by)::bigint
)
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value,
    description = EXCLUDED.description,
    updated_at = NOW(),
    updated_by = EXCLUDED.updated_by;

-- name: CreatePostedJournalEntryHeader :one
-- Menulis header jurnal yang LANGSUNG berstatus POSTED dalam satu statement.
-- Dipakai auto-posting (GRN, kas, penjualan) supaya tidak pernah ada jurnal
-- DRAFT yang menggantung bila penulisan baris gagal: bila transaksi ini batal,
-- seluruhnya batal. posted_at/posted_by diisi bersamaan sehingga kolom audit
-- jurnal otomatis tidak pernah kosong.
--
-- Pelanggaran unique pada idx_journal_source_unique berarti (source_type,
-- source_id) sudah punya jurnal POSTED — memakai unique index parsial itulah
-- auto-posting dijamin idempoten. Pelanggaran pada nomor jurnal meminta
-- pemanggil mengambil nomor berikutnya.
--
-- INSERT ... SELECT ini juga mengunci baris periode dengan FOR SHARE dan hanya
-- menghasilkan baris bila periode masih OPEN. Dengan begitu penutupan buku yang
-- memegang FOR UPDATE tidak bisa berlomba dengan auto-posting: salah satunya
-- menunggu, dan bila periode sudah ditutup, INSERT tidak menulis apa pun.
INSERT INTO t_journal_entries (
    entry_number, entry_date, description, source_type, source_id, source_reference,
    status, total_debit, total_credit, period_year, period_month, reversal_of_id,
    posted_at, posted_by, created_by, updated_by
)
SELECT
    sqlc.arg(entry_number)::varchar,
    sqlc.arg(entry_date)::date,
    sqlc.narg(description)::text,
    sqlc.narg(source_type)::text,
    sqlc.narg(source_id)::bigint,
    sqlc.narg(source_reference)::varchar,
    'POSTED',
    sqlc.arg(total_debit)::numeric,
    sqlc.arg(total_credit)::numeric,
    sqlc.arg(period_year)::int,
    sqlc.arg(period_month)::int,
    sqlc.narg(reversal_of_id)::bigint,
    NOW(),
    sqlc.narg(posted_by)::bigint,
    sqlc.narg(created_by)::bigint,
    sqlc.narg(updated_by)::bigint
FROM m_accounting_periods p
WHERE p.period_year = sqlc.arg(period_year)::int
  AND p.period_month = sqlc.arg(period_month)::int
  AND p.status = 'OPEN'
FOR SHARE
RETURNING id, entry_number, entry_date, description, status, total_debit, total_credit,
          period_year, period_month, posted_at, posted_by, voided_at, voided_by,
          void_reason, reversal_of_id, reversed_by_id, created_at, updated_at, created_by;

-- =========================================================================
-- MODUL AKUNTANSI (FASE 2C): LAPORAN KEUANGAN
-- =========================================================================
--
-- Laporan hanya membaca jurnal berstatus POSTED. Join terhadap
-- t_journal_entry_lines bersifat INNER sehingga akun yang tidak pernah
-- diposting (termasuk akun header) tidak pernah muncul sebagai baris laporan.

-- name: GetPostedAccountBalances :many
-- Menjumlahkan debit/kredit per akun dari jurnal POSTED pada rentang tanggal
-- inklusif. Bahan mentah Neraca, Laba Rugi, dan Neraca Saldo. Rentang tanggal
-- panjang (dari awal tahun buku) menghasilkan saldo kumulatif untuk Neraca.
SELECT
    a.id AS account_id,
    a.code,
    a.name,
    a.account_type,
    a.normal_balance,
    COALESCE(SUM(l.debit), 0)::numeric AS total_debit,
    COALESCE(SUM(l.credit), 0)::numeric AS total_credit
FROM m_chart_of_accounts a
JOIN t_journal_entry_lines l ON l.account_id = a.id
JOIN t_journal_entries j ON j.id = l.journal_entry_id
WHERE a.deleted_at IS NULL
  AND j.status = 'POSTED'
  AND j.entry_date >= sqlc.arg(from_date)::date
  AND j.entry_date <= sqlc.arg(to_date)::date
GROUP BY a.id, a.code, a.name, a.account_type, a.normal_balance
ORDER BY a.code;

-- name: GetAccountOpeningBalance :one
-- Saldo awal buku besar: total debit/kredit satu akun dari jurnal POSTED
-- sebelum tanggal rentang dimulai.
SELECT
    COALESCE(SUM(l.debit), 0)::numeric AS total_debit,
    COALESCE(SUM(l.credit), 0)::numeric AS total_credit
FROM t_journal_entry_lines l
JOIN t_journal_entries j ON j.id = l.journal_entry_id
WHERE l.account_id = sqlc.arg(account_id)::bigint
  AND j.status = 'POSTED'
  AND j.entry_date < sqlc.arg(before_date)::date;

-- name: GetAccountLedgerLines :many
-- Mutasi rinci satu akun pada rentang tanggal, terurut kronologis. Nomor jurnal
-- dan tanggal ikut ditampilkan supaya pembaca bisa menelusuri asal angka.
SELECT
    j.entry_number,
    j.entry_date,
    COALESCE(l.description, j.description, '') AS description,
    l.debit,
    l.credit
FROM t_journal_entry_lines l
JOIN t_journal_entries j ON j.id = l.journal_entry_id
WHERE l.account_id = sqlc.arg(account_id)::bigint
  AND j.status = 'POSTED'
  AND j.entry_date >= sqlc.arg(from_date)::date
  AND j.entry_date <= sqlc.arg(to_date)::date
ORDER BY j.entry_date ASC, j.id ASC, l.line_no ASC;

-- =========================================================================
-- MODUL AKUNTANSI (FASE 2D): JURNAL PENUTUP
-- =========================================================================

-- name: GetPostedJournalIDBySource :one
-- Mencari id jurnal POSTED yang belum dibalik untuk pasangan (source_type,
-- source_id). Dipakai untuk menemukan jurnal penutup sebuah periode saat
-- periode itu dibuka kembali. Jurnal yang sudah dibalik dikecualikan supaya
-- konsisten dengan definisi idx_journal_source_unique: setelah pembalikan,
-- sumber yang sama boleh menerbitkan jurnal baru.
SELECT id
FROM t_journal_entries
WHERE source_type = sqlc.arg(source_type)::text
  AND source_id = sqlc.arg(source_id)::bigint
  AND status = 'POSTED'
  AND reversed_by_id IS NULL
ORDER BY id DESC
LIMIT 1;

-- =========================================================================
-- MODUL PENJUALAN / KASIR (POS)
-- =========================================================================

-- ------------------------------ MASTER MEMBER ----------------------------

-- name: CreateMember :one
INSERT INTO m_members (code, name, phone, email, address, default_term_days, created_by, updated_by)
VALUES (
    sqlc.arg(code)::varchar,
    sqlc.arg(name)::varchar,
    sqlc.arg(phone)::varchar,
    sqlc.arg(email)::varchar,
    sqlc.arg(address)::text,
    sqlc.arg(default_term_days)::int,
    sqlc.narg(created_by)::bigint,
    sqlc.narg(updated_by)::bigint
)
RETURNING id, code, name, phone, email, address, default_term_days, points, is_active, created_at;

-- name: GetMembersCursorNext :many
SELECT c.id, c.code, c.name, c.phone, c.email, c.address, c.default_term_days, c.points, c.is_active, c.created_at
FROM m_members c
WHERE c.deleted_at IS NULL
  AND (sqlc.arg(search_query)::text = ''
       OR c.name ILIKE '%' || sqlc.arg(search_query)::text || '%'
       OR c.code ILIKE '%' || sqlc.arg(search_query)::text || '%'
       OR c.phone ILIKE '%' || sqlc.arg(search_query)::text || '%')
  AND (sqlc.arg(cursor_id)::bigint = 0 OR c.id < sqlc.arg(cursor_id)::bigint)
ORDER BY c.id DESC
LIMIT sqlc.arg(req_limit)::int;

-- name: GetMembersCursorPrev :many
SELECT c.id, c.code, c.name, c.phone, c.email, c.address, c.default_term_days, c.points, c.is_active, c.created_at
FROM m_members c
WHERE c.deleted_at IS NULL
  AND (sqlc.arg(search_query)::text = ''
       OR c.name ILIKE '%' || sqlc.arg(search_query)::text || '%'
       OR c.code ILIKE '%' || sqlc.arg(search_query)::text || '%'
       OR c.phone ILIKE '%' || sqlc.arg(search_query)::text || '%')
  AND c.id > sqlc.arg(cursor_id)::bigint
ORDER BY c.id ASC
LIMIT sqlc.arg(req_limit)::int;

-- name: CountMembers :one
SELECT COUNT(*)::bigint AS total_count
FROM m_members c
WHERE c.deleted_at IS NULL
  AND (sqlc.arg(search_query)::text = ''
       OR c.name ILIKE '%' || sqlc.arg(search_query)::text || '%'
       OR c.code ILIKE '%' || sqlc.arg(search_query)::text || '%'
       OR c.phone ILIKE '%' || sqlc.arg(search_query)::text || '%');

-- name: GetMemberByID :one
SELECT c.id, c.code, c.name, c.phone, c.email, c.address, c.default_term_days, c.points, c.is_active, c.created_at
FROM m_members c
WHERE c.id = sqlc.arg(id)::bigint AND c.deleted_at IS NULL;

-- name: UpdateMember :exec
UPDATE m_members SET
    code = sqlc.arg(code)::varchar,
    name = sqlc.arg(name)::varchar,
    phone = sqlc.arg(phone)::varchar,
    email = sqlc.arg(email)::varchar,
    address = sqlc.arg(address)::text,
    default_term_days = sqlc.arg(default_term_days)::int,
    is_active = sqlc.arg(is_active)::boolean,
    updated_at = NOW(),
    updated_by = sqlc.narg(updated_by)::bigint
WHERE id = sqlc.arg(id)::bigint AND deleted_at IS NULL;

-- name: DeleteMember :exec
UPDATE m_members
SET deleted_at = NOW(), updated_at = NOW(), deleted_by = sqlc.narg(deleted_by)::bigint
WHERE id = sqlc.arg(id)::bigint AND deleted_at IS NULL;

-- name: LockMemberPoints :one
SELECT id, points FROM m_members WHERE id = sqlc.arg(id)::bigint AND deleted_at IS NULL FOR UPDATE;

-- name: SetMemberPoints :exec
UPDATE m_members SET points = sqlc.arg(points)::bigint, updated_at = NOW(), updated_by = sqlc.narg(updated_by)::bigint
WHERE id = sqlc.arg(id)::bigint;

-- name: InsertMemberPointEntry :exec
INSERT INTO t_member_points (member_id, sale_id, entry_type, points, balance_after, note, created_by)
VALUES (
    sqlc.arg(member_id)::bigint,
    sqlc.narg(sale_id)::bigint,
    sqlc.arg(entry_type)::varchar,
    sqlc.arg(points)::bigint,
    sqlc.arg(balance_after)::bigint,
    sqlc.arg(note)::text,
    sqlc.narg(created_by)::bigint
);

-- name: GetMemberPointEntriesBySale :many
SELECT id, member_id, sale_id, entry_type, points, balance_after, note, created_at
FROM t_member_points
WHERE sale_id = sqlc.arg(sale_id)::bigint
ORDER BY id ASC;

-- name: ListMemberPointEntries :many
SELECT e.id, e.member_id, e.sale_id, e.entry_type, e.points, e.balance_after, e.note, e.created_at,
       s.sale_number
FROM t_member_points e
LEFT JOIN t_sales s ON s.id = e.sale_id
WHERE e.member_id = sqlc.arg(member_id)::bigint
ORDER BY e.id DESC
LIMIT sqlc.arg(req_limit)::int;

-- name: LookupMembersForPOS :many
-- Pencarian member dari layar kasir (kode/telepon/nama). Kode yang cocok persis
-- ditampilkan lebih dulu supaya hasil scan barcode/kartu langsung teratas.
SELECT c.id, c.code, c.name, c.phone, c.default_term_days, c.points
FROM m_members c
WHERE c.deleted_at IS NULL AND c.is_active = TRUE
  AND (sqlc.arg(search_query)::text = ''
       OR c.name ILIKE '%' || sqlc.arg(search_query)::text || '%'
       OR c.code ILIKE '%' || sqlc.arg(search_query)::text || '%'
       OR c.phone ILIKE '%' || sqlc.arg(search_query)::text || '%')
ORDER BY (LOWER(c.code) = LOWER(sqlc.arg(search_query)::text)) DESC, c.name ASC
LIMIT sqlc.arg(req_limit)::int;

-- -------------------------- METODE PEMBAYARAN ----------------------------

-- name: CreatePaymentMethod :one
INSERT INTO m_payment_methods (code, name, channel, account_id, is_cash, sort_order, created_by, updated_by)
VALUES (
    sqlc.arg(code)::varchar,
    sqlc.arg(name)::varchar,
    sqlc.arg(channel)::varchar,
    sqlc.narg(account_id)::bigint,
    sqlc.arg(is_cash)::boolean,
    sqlc.arg(sort_order)::int,
    sqlc.narg(created_by)::bigint,
    sqlc.narg(updated_by)::bigint
)
RETURNING id, code, name, channel, account_id, is_cash, sort_order, is_active, created_at;

-- name: GetPaymentMethodByID :one
SELECT m.id, m.code, m.name, m.channel, m.account_id, m.is_cash, m.sort_order, m.is_active, m.created_at
FROM m_payment_methods m
WHERE m.id = sqlc.arg(id)::bigint AND m.deleted_at IS NULL;

-- name: GetPaymentMethodsByIDs :many
-- Mengambil beberapa metode pembayaran sekaligus supaya pemetaan akun pada
-- transaksi penjualan tidak menembak satu query per metode.
SELECT m.id, m.code, m.name, m.channel, m.account_id, m.is_cash, m.sort_order, m.is_active
FROM m_payment_methods m
WHERE m.id = ANY(sqlc.arg(ids)::bigint[]) AND m.deleted_at IS NULL;

-- name: UpdatePaymentMethod :exec
UPDATE m_payment_methods SET
    code = sqlc.arg(code)::varchar,
    name = sqlc.arg(name)::varchar,
    channel = sqlc.arg(channel)::varchar,
    account_id = sqlc.narg(account_id)::bigint,
    is_cash = sqlc.arg(is_cash)::boolean,
    sort_order = sqlc.arg(sort_order)::int,
    is_active = sqlc.arg(is_active)::boolean,
    updated_at = NOW(),
    updated_by = sqlc.narg(updated_by)::bigint
WHERE id = sqlc.arg(id)::bigint AND deleted_at IS NULL;

-- name: DeletePaymentMethod :exec
UPDATE m_payment_methods
SET deleted_at = NOW(), updated_at = NOW(), deleted_by = sqlc.narg(deleted_by)::bigint
WHERE id = sqlc.arg(id)::bigint AND deleted_at IS NULL;

-- name: GetActivePaymentMethods :many
SELECT m.id, m.code, m.name, m.channel, m.account_id, m.is_cash, m.sort_order, m.is_active
FROM m_payment_methods m
WHERE m.deleted_at IS NULL AND m.is_active = TRUE
ORDER BY m.sort_order ASC, m.name ASC;

-- name: ListPaymentMethods :many
SELECT m.id, m.code, m.name, m.channel, m.account_id, m.is_cash, m.sort_order, m.is_active
FROM m_payment_methods m
WHERE m.deleted_at IS NULL
ORDER BY m.sort_order ASC, m.name ASC;

-- ------------------------------ MASTER VOUCHER ---------------------------

-- name: CreateVoucher :one
INSERT INTO m_vouchers (
    code, name, description, discount_type, discount_value,
    min_purchase, max_discount, valid_from, valid_until, is_active,
    created_by, updated_by
) VALUES (
    sqlc.arg(code)::varchar,
    sqlc.arg(name)::varchar,
    sqlc.arg(description)::text,
    sqlc.arg(discount_type)::varchar,
    sqlc.arg(discount_value)::numeric,
    sqlc.arg(min_purchase)::numeric,
    sqlc.arg(max_discount)::numeric,
    sqlc.narg(valid_from)::date,
    sqlc.narg(valid_until)::date,
    sqlc.arg(is_active)::boolean,
    sqlc.narg(created_by)::bigint,
    sqlc.narg(updated_by)::bigint
)
RETURNING id, code, name, description, discount_type, discount_value,
          min_purchase, max_discount, valid_from, valid_until, is_active,
          used_at, created_at;

-- name: GetVoucherByID :one
SELECT id, code, name, description, discount_type, discount_value,
       min_purchase, max_discount, valid_from, valid_until, is_active,
       used_at, created_at
FROM m_vouchers
WHERE id = sqlc.arg(id)::bigint AND deleted_at IS NULL;

-- name: GetVoucherByCode :one
-- Pencarian kode untuk scan barcode; tidak sensitif huruf besar/kecil.
SELECT id, code, name, description, discount_type, discount_value,
       min_purchase, max_discount, valid_from, valid_until, is_active,
       used_at, created_at
FROM m_vouchers
WHERE UPPER(code) = UPPER(sqlc.arg(code)::varchar) AND deleted_at IS NULL;

-- name: GetVouchersCursorNext :many
SELECT id, code, name, description, discount_type, discount_value,
       min_purchase, max_discount, valid_from, valid_until, is_active,
       used_at, created_at
FROM m_vouchers
WHERE deleted_at IS NULL
  AND (sqlc.arg(search_query)::text = ''
       OR name ILIKE '%' || sqlc.arg(search_query)::text || '%'
       OR code ILIKE '%' || sqlc.arg(search_query)::text || '%')
  AND (sqlc.arg(cursor_id)::bigint = 0 OR id < sqlc.arg(cursor_id)::bigint)
ORDER BY id DESC
LIMIT sqlc.arg(req_limit)::int;

-- name: GetVouchersCursorPrev :many
SELECT id, code, name, description, discount_type, discount_value,
       min_purchase, max_discount, valid_from, valid_until, is_active,
       used_at, created_at
FROM m_vouchers
WHERE deleted_at IS NULL
  AND (sqlc.arg(search_query)::text = ''
       OR name ILIKE '%' || sqlc.arg(search_query)::text || '%'
       OR code ILIKE '%' || sqlc.arg(search_query)::text || '%')
  AND id > sqlc.arg(cursor_id)::bigint
ORDER BY id ASC
LIMIT sqlc.arg(req_limit)::int;

-- name: CountVouchers :one
SELECT COUNT(*)::bigint AS total_count
FROM m_vouchers
WHERE deleted_at IS NULL
  AND (sqlc.arg(search_query)::text = ''
       OR name ILIKE '%' || sqlc.arg(search_query)::text || '%'
       OR code ILIKE '%' || sqlc.arg(search_query)::text || '%');

-- name: UpdateVoucher :exec
UPDATE m_vouchers SET
    code = sqlc.arg(code)::varchar,
    name = sqlc.arg(name)::varchar,
    description = sqlc.arg(description)::text,
    discount_type = sqlc.arg(discount_type)::varchar,
    discount_value = sqlc.arg(discount_value)::numeric,
    min_purchase = sqlc.arg(min_purchase)::numeric,
    max_discount = sqlc.arg(max_discount)::numeric,
    valid_from = sqlc.narg(valid_from)::date,
    valid_until = sqlc.narg(valid_until)::date,
    is_active = sqlc.arg(is_active)::boolean,
    updated_at = NOW(),
    updated_by = sqlc.narg(updated_by)::bigint
WHERE id = sqlc.arg(id)::bigint AND deleted_at IS NULL;

-- name: DeleteVoucher :exec
UPDATE m_vouchers
SET deleted_at = NOW(), updated_at = NOW(), deleted_by = sqlc.narg(deleted_by)::bigint
WHERE id = sqlc.arg(id)::bigint AND deleted_at IS NULL;

-- name: LockVoucherForSale :one
-- Mengunci voucher di dalam transaksi nota agar single-use tidak bisa
-- dilanggar oleh dua transaksi bersamaan.
SELECT id, code, discount_type, discount_value, min_purchase, max_discount,
       valid_from, valid_until, is_active, used_at
FROM m_vouchers
WHERE id = sqlc.arg(id)::bigint AND deleted_at IS NULL
FOR UPDATE;

-- name: MarkVoucherUsed :execrows
UPDATE m_vouchers
SET used_at = NOW(), updated_at = NOW(), updated_by = sqlc.narg(updated_by)::bigint
WHERE id = sqlc.arg(id)::bigint AND used_at IS NULL;

-- name: ReleaseVoucher :exec
UPDATE m_vouchers
SET used_at = NULL, updated_at = NOW(), updated_by = sqlc.narg(updated_by)::bigint
WHERE id = sqlc.arg(id)::bigint;

-- ------------------------------ PRODUK POS ------------------------------

-- name: LookupProductsForPOS :many
-- Pencarian produk untuk grid kasir. Barcode ikut dicari agar scanner yang
-- mengirim kode lalu Enter langsung menemukan produk.
SELECT
    p.id, p.sku, p.barcode, p.name, p.selling_price, p.current_stock, p.discount_type, p.discount_value,
    p.is_taxable,
    COALESCE(c.name, '') AS category_name,
    COALESCE(u.kode, '') AS selling_unit
FROM m_products p
LEFT JOIN m_product_categories c ON c.id = p.category_id
LEFT JOIN m_units u ON u.id = p.selling_unit_id
WHERE p.deleted_at IS NULL AND p.is_active = TRUE
  AND (sqlc.arg(search_query)::text = ''
       OR p.name ILIKE '%' || sqlc.arg(search_query)::text || '%'
       OR p.sku ILIKE '%' || sqlc.arg(search_query)::text || '%'
       OR COALESCE(p.barcode, '') ILIKE '%' || sqlc.arg(search_query)::text || '%')
  AND (sqlc.arg(category_id)::int = 0 OR p.category_id = sqlc.arg(category_id)::int)
ORDER BY p.name ASC
LIMIT sqlc.arg(req_limit)::int;

-- name: GetActiveProductCategories :many
SELECT id, name
FROM m_product_categories
WHERE deleted_at IS NULL AND is_active = TRUE
ORDER BY name ASC;

-- name: GetProductsForSale :many
-- Membaca snapshot produk untuk menghitung nilai penjualan SEBELUM transaksi
-- dimulai. Penguncian dan validasi stok final tetap dilakukan di dalam
-- transaksi oleh LockProductsForSale.
SELECT id, sku, name, selling_price, current_stock, avg_cost, discount_type, discount_value, is_taxable
FROM m_products
WHERE id = ANY(sqlc.arg(product_ids)::bigint[]) AND deleted_at IS NULL;

-- name: LockProductsForSale :many
-- Mengunci baris produk dalam transaksi penjualan supaya stok tidak bisa
-- dikurangi dua kali secara bersamaan untuk produk yang sama.
SELECT id, sku, name, selling_price, current_stock, avg_cost, discount_type, discount_value, is_taxable
FROM m_products
WHERE id = ANY(sqlc.arg(product_ids)::bigint[]) AND deleted_at IS NULL
ORDER BY id
FOR UPDATE;

-- name: DecrementProductStocksForSale :one
-- Mengurangi stok semua produk pada satu nota dalam SATU UPDATE ... FROM.
-- Sebelumnya setiap baris penjualan memanggil DecrementProductStock sendiri
-- (N+1 query). qty dijumlahkan per produk lebih dulu karena satu nota boleh
-- memuat produk yang sama di beberapa baris.
--
-- Penjaga current_stock >= qty dan g.qty > 0 tetap dipertahankan. Baris
-- dengan qty non-positif tidak pernah mengubah stok (defense-in-depth di luar
-- validasi domain/repository). expected_count vs updated_count memberi tahu
-- pemanggil bahwa ada produk yang stoknya tidak cukup, tanpa perlu menebak
-- dari nol baris satu per satu.
WITH items AS (
    SELECT p.product_id, q.qty
    FROM unnest(sqlc.arg(product_ids)::bigint[]) WITH ORDINALITY AS p(product_id, ord)
    JOIN unnest(sqlc.arg(qtys)::int[]) WITH ORDINALITY AS q(qty, ord) USING (ord)
),
grouped AS (
    SELECT product_id, SUM(qty)::int AS qty
    FROM items
    GROUP BY product_id
),
updated AS (
    UPDATE m_products p
    SET current_stock = p.current_stock - g.qty,
        updated_at = NOW(),
        updated_by = sqlc.narg(updated_by)::bigint
    FROM grouped g
    WHERE p.id = g.product_id
      AND g.qty > 0
      AND p.current_stock >= g.qty
    RETURNING p.id
)
SELECT
    (SELECT COUNT(*) FROM grouped)::bigint AS expected_count,
    (SELECT COUNT(*) FROM updated)::bigint AS updated_count;

-- name: LockProductsForVoid :many
-- Mengunci produk yang stoknya harus dikembalikan saat void, termasuk produk
-- yang sudah dinonaktifkan/di-soft-delete. ORDER BY id menjaga urutan kunci
-- tetap deterministik antar transaksi.
SELECT id, current_stock, avg_cost
FROM m_products
WHERE id = ANY(sqlc.arg(product_ids)::bigint[])
ORDER BY id
FOR UPDATE;

-- name: RestoreProductStocksWithValuation :execrows
-- Mengembalikan stok seluruh produk pada satu nota dalam SATU UPDATE ... FROM,
-- bukan satu UPDATE per produk. Rata-rata tertimbang memakai harga pokok asli
-- penjualan (cost_total) dan nilai baris SEBELUM update.
WITH items AS (
    SELECT p.product_id, q.qty, c.cost_total
    FROM unnest(sqlc.arg(product_ids)::bigint[]) WITH ORDINALITY AS p(product_id, ord)
    JOIN unnest(sqlc.arg(qtys)::int[]) WITH ORDINALITY AS q(qty, ord) USING (ord)
    JOIN unnest(sqlc.arg(cost_totals)::numeric[]) WITH ORDINALITY AS c(cost_total, ord) USING (ord)
)
UPDATE m_products p
SET current_stock = p.current_stock + i.qty,
    avg_cost = CASE
        WHEN p.current_stock + i.qty > 0
        THEN (p.current_stock * p.avg_cost + i.cost_total)
             / (p.current_stock + i.qty)
        ELSE p.avg_cost
    END,
    updated_at = NOW(),
    updated_by = sqlc.narg(updated_by)::bigint
FROM items i
WHERE p.id = i.product_id;

-- --------------------------- TRANSAKSI PENJUALAN -------------------------

-- name: NextSaleSeq :one
INSERT INTO t_sales_number_counters (prefix, last_seq)
VALUES (sqlc.arg(prefix)::varchar, 1)
ON CONFLICT (prefix) DO UPDATE SET last_seq = t_sales_number_counters.last_seq + 1
RETURNING last_seq;

-- name: CreateSale :one
INSERT INTO t_sales (
    sale_number, sale_date, member_id, tax_type,
    subtotal, item_discount, discount_type, discount_value, discount_amount,
    dpp_amount, tax_amount, total_amount, paid_amount, change_amount, cost_total,
    points_earned, redeem_points, redeem_amount,
    voucher_id, voucher_code, voucher_discount,
    payment_status, status, due_date, notes, created_by, updated_by
) VALUES (
    sqlc.arg(sale_number)::varchar,
    sqlc.arg(sale_date)::date,
    sqlc.narg(member_id)::bigint,
    sqlc.arg(tax_type)::varchar,
    sqlc.arg(subtotal)::numeric,
    sqlc.arg(item_discount)::numeric,
    sqlc.arg(discount_type)::varchar,
    sqlc.arg(discount_value)::numeric,
    sqlc.arg(discount_amount)::numeric,
    sqlc.arg(dpp_amount)::numeric,
    sqlc.arg(tax_amount)::numeric,
    sqlc.arg(total_amount)::numeric,
    sqlc.arg(paid_amount)::numeric,
    sqlc.arg(change_amount)::numeric,
    sqlc.arg(cost_total)::numeric,
    sqlc.arg(points_earned)::bigint,
    sqlc.arg(redeem_points)::bigint,
    sqlc.arg(redeem_amount)::numeric,
    sqlc.narg(voucher_id)::bigint,
    sqlc.arg(voucher_code)::varchar,
    sqlc.arg(voucher_discount)::numeric,
    sqlc.arg(payment_status)::varchar,
    sqlc.arg(status)::varchar,
    sqlc.narg(due_date)::date,
    sqlc.arg(notes)::text,
    sqlc.narg(created_by)::bigint,
    sqlc.narg(updated_by)::bigint
)
RETURNING id, sale_number, sale_date, member_id, tax_type, subtotal, item_discount,
          discount_type, discount_value, discount_amount, dpp_amount, tax_amount,
          total_amount, paid_amount, change_amount, cost_total, points_earned, redeem_points,
          redeem_amount, voucher_id, voucher_code, voucher_discount,
          payment_status, status, due_date, notes, created_at;

-- name: CreateSaleDetail :exec
INSERT INTO t_sales_details (
    sale_id, product_id, sku, product_name, qty, unit_price,
    discount_type, discount_value, discount_amount, gross_amount, line_total,
    dpp_amount, tax_amount,
    cost_unit, cost_total
) VALUES (
    sqlc.arg(sale_id)::bigint,
    sqlc.arg(product_id)::bigint,
    sqlc.arg(sku)::varchar,
    sqlc.arg(product_name)::varchar,
    sqlc.arg(qty)::int,
    sqlc.arg(unit_price)::numeric,
    sqlc.arg(discount_type)::varchar,
    sqlc.arg(discount_value)::numeric,
    sqlc.arg(discount_amount)::numeric,
    sqlc.arg(gross_amount)::numeric,
    sqlc.arg(line_total)::numeric,
    sqlc.arg(dpp_amount)::numeric,
    sqlc.arg(tax_amount)::numeric,
    sqlc.arg(cost_unit)::numeric,
    sqlc.arg(cost_total)::numeric
);

-- name: CreateSalePayment :exec
INSERT INTO t_sales_payments (sale_id, payment_method_id, amount, reference_number)
VALUES (
    sqlc.arg(sale_id)::bigint,
    sqlc.arg(payment_method_id)::bigint,
    sqlc.arg(amount)::numeric,
    sqlc.arg(reference_number)::varchar
);

-- name: SetSaleJournal :exec
UPDATE t_sales
SET journal_entry_id = sqlc.arg(journal_entry_id)::bigint, updated_at = NOW()
WHERE id = sqlc.arg(id)::bigint;

-- name: GetSaleByID :one
SELECT
    s.id, s.sale_number, s.sale_date, s.member_id, COALESCE(c.name, '') AS member_name,
    s.tax_type, s.subtotal, s.item_discount, s.discount_type, s.discount_value,
    s.discount_amount, s.dpp_amount, s.tax_amount, s.total_amount, s.paid_amount,
    s.change_amount, s.cost_total, s.points_earned, s.redeem_points, s.redeem_amount,
    s.voucher_id, s.voucher_code, s.voucher_discount,
    s.payment_status, s.status, s.due_date, s.notes, s.journal_entry_id,
    s.voided_at, s.void_reason, s.created_at, s.created_by
FROM t_sales s
LEFT JOIN m_members c ON c.id = s.member_id
WHERE s.id = sqlc.arg(id)::bigint;

-- name: GetSaleDetails :many
SELECT
    d.id, d.sale_id, d.product_id, d.sku, d.product_name, d.qty, d.unit_price,
    d.discount_type, d.discount_value, d.discount_amount, d.gross_amount,
    d.line_total, d.dpp_amount, d.tax_amount, d.cost_unit, d.cost_total
FROM t_sales_details d
WHERE d.sale_id = sqlc.arg(sale_id)::bigint
ORDER BY d.id ASC;

-- name: GetSalePayments :many
SELECT
    p.id, p.sale_id, p.payment_method_id, m.name AS payment_method_name,
    m.channel, p.amount, p.reference_number, p.created_at
FROM t_sales_payments p
JOIN m_payment_methods m ON m.id = p.payment_method_id
WHERE p.sale_id = sqlc.arg(sale_id)::bigint
ORDER BY p.id ASC;

-- name: GetSalesCursorNext :many
SELECT
    s.id, s.sale_number, s.sale_date, s.member_id, COALESCE(c.name, '') AS member_name,
    s.total_amount, s.paid_amount, s.voucher_code, s.payment_status, s.status, s.tax_type, s.created_at
FROM t_sales s
LEFT JOIN m_members c ON c.id = s.member_id
WHERE (sqlc.arg(search_query)::text = ''
       OR s.sale_number ILIKE '%' || sqlc.arg(search_query)::text || '%'
       OR c.name ILIKE '%' || sqlc.arg(search_query)::text || '%')
  AND (sqlc.arg(status_filter)::text = 'ALL' OR s.status = sqlc.arg(status_filter)::text)
  AND (sqlc.arg(cursor_id)::bigint = 0 OR s.id < sqlc.arg(cursor_id)::bigint)
ORDER BY s.id DESC
LIMIT sqlc.arg(req_limit)::int;

-- name: GetSalesCursorPrev :many
SELECT
    s.id, s.sale_number, s.sale_date, s.member_id, COALESCE(c.name, '') AS member_name,
    s.total_amount, s.paid_amount, s.voucher_code, s.payment_status, s.status, s.tax_type, s.created_at
FROM t_sales s
LEFT JOIN m_members c ON c.id = s.member_id
WHERE (sqlc.arg(search_query)::text = ''
       OR s.sale_number ILIKE '%' || sqlc.arg(search_query)::text || '%'
       OR c.name ILIKE '%' || sqlc.arg(search_query)::text || '%')
  AND (sqlc.arg(status_filter)::text = 'ALL' OR s.status = sqlc.arg(status_filter)::text)
  AND s.id > sqlc.arg(cursor_id)::bigint
ORDER BY s.id ASC
LIMIT sqlc.arg(req_limit)::int;

-- name: CountSales :one
SELECT COUNT(*)::bigint AS total_count
FROM t_sales s
LEFT JOIN m_members c ON c.id = s.member_id
WHERE (sqlc.arg(search_query)::text = ''
       OR s.sale_number ILIKE '%' || sqlc.arg(search_query)::text || '%'
       OR c.name ILIKE '%' || sqlc.arg(search_query)::text || '%')
  AND (sqlc.arg(status_filter)::text = 'ALL' OR s.status = sqlc.arg(status_filter)::text);

-- name: LockSaleForVoid :one
SELECT s.id, s.sale_number, s.member_id, s.points_earned, s.redeem_points,
       s.voucher_id, s.status, s.journal_entry_id
FROM t_sales s
WHERE s.id = sqlc.arg(id)::bigint
FOR UPDATE;

-- name: VoidSale :execrows
UPDATE t_sales
SET status = 'VOID',
    payment_status = 'UNPAID',
    voided_at = NOW(),
    voided_by = sqlc.narg(voided_by)::bigint,
    void_reason = sqlc.arg(void_reason)::text,
    updated_at = NOW(),
    updated_by = sqlc.narg(voided_by)::bigint
WHERE id = sqlc.arg(id)::bigint AND status = 'POSTED';
