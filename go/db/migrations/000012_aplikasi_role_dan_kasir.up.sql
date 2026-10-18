-- =========================================================================
-- APLIKASI PER ROLE (ERP vs POS) + ROLE KASIR
-- =========================================================================
--
-- Latar belakang:
-- Seluruh role saat ini melekat pada satu aplikasi ERP. Ke depan, aplikasi
-- kasir (POS) akan berdiri sendiri dan hanya boleh diakses oleh petugas
-- kasir. Supaya batas itu tegas dan tidak rapuh (mis. bergantung pada nama
-- role), setiap role diberi penanda aplikasi lewat kolom `app`.
--
--   * app = 'erp' -> role untuk aplikasi ERP (default semua role lama).
--   * app = 'pos' -> role untuk aplikasi kasir (POS).
--
-- Frontend memakai nilai ini untuk mengarahkan pengguna setelah login dan
-- mengunci kasir agar hanya bisa membuka area /pos. Penegakan yang sebenarnya
-- tetap di backend melalui permission: role Kasir sengaja tidak diberi satu
-- pun permission ERP, hanya `pos:read`.
--
-- Perintah: make migrateup

-- 0. Perbaiki sequence yang mungkin tertinggal dari data lama.
SELECT setval(
    's_permissions_id_seq',
    GREATEST((SELECT COALESCE(MAX(id), 1) FROM s_permissions), 1),
    true
);
SELECT setval(
    's_roles_id_seq',
    GREATEST((SELECT COALESCE(MAX(id), 1) FROM s_roles), 1),
    true
);

-- 1. Kolom penanda aplikasi pada role.
ALTER TABLE s_roles
    ADD COLUMN IF NOT EXISTS app VARCHAR(20) NOT NULL DEFAULT 'erp';

ALTER TABLE s_roles
    DROP CONSTRAINT IF EXISTS s_roles_app_check;

ALTER TABLE s_roles
    ADD CONSTRAINT s_roles_app_check CHECK (app IN ('erp', 'pos'));

-- 2. Permission penanda izin membuka aplikasi POS.
INSERT INTO s_permissions (name, description)
VALUES ('pos:read', 'Membuka aplikasi kasir (POS)')
ON CONFLICT (name) DO NOTHING;

-- 3. Role Kasir khusus aplikasi POS.
INSERT INTO s_roles (name, app)
VALUES ('Kasir', 'pos')
ON CONFLICT (name) DO NOTHING;

-- 4. Kasir HANYA punya pos:read — tanpa satu pun permission ERP.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM s_roles r, s_permissions p
WHERE r.name = 'Kasir'
  AND p.name = 'pos:read'
ON CONFLICT DO NOTHING;

-- 5. Berikan pos:read ke peran admin agar tetap bisa membuka POS.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM s_roles r, s_permissions p
WHERE r.name IN ('Super Admin', 'Admin')
  AND p.name = 'pos:read'
ON CONFLICT DO NOTHING;

-- 6. Endpoint dashboard sebelumnya hanya dijaga login (tanpa permission),
--    sehingga kasir bisa membaca angka ringkasan ERP lewat API. Tambahkan
--    permission dashboard:read dan berikan ke seluruh role ERP yang sudah
--    punya minimal satu permission, supaya perilaku lama tidak berubah.
--    Role tanpa permission (mis. Customer) dan role aplikasi POS sengaja
--    tidak diberi.
INSERT INTO s_permissions (name, description)
VALUES ('dashboard:read', 'Melihat data ringkasan dashboard')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT DISTINCT rp.role_id, p.id
FROM role_permissions rp
JOIN s_roles r ON r.id = rp.role_id
CROSS JOIN s_permissions p
WHERE r.app = 'erp'
  AND p.name = 'dashboard:read'
ON CONFLICT DO NOTHING;
