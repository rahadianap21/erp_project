-- Kembalikan skema role ke bentuk sebelum penanda aplikasi.
--
-- Peringatan: role 'Kasir' dan permission 'pos:read' dihapus. Pengguna yang
-- memakai role Kasir harus dipindahkan ke role lain lebih dulu, jika tidak
-- migrasi turun ini gagal karena foreign key s_users.role_id.

DELETE FROM role_permissions
WHERE role_id IN (SELECT id FROM s_roles WHERE name = 'Kasir')
   OR permission_id IN (SELECT id FROM s_permissions WHERE name IN ('pos:read', 'dashboard:read'));

DELETE FROM s_roles WHERE name = 'Kasir';
DELETE FROM s_permissions WHERE name IN ('pos:read', 'dashboard:read');

ALTER TABLE s_roles DROP CONSTRAINT IF EXISTS s_roles_app_check;
ALTER TABLE s_roles DROP COLUMN IF EXISTS app;
