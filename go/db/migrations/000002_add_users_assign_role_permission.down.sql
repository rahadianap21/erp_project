-- =========================================================================
-- MUNDURKAN PERMISSION users:assign-role
-- =========================================================================
--
-- Hapus dulu relasi di role_permissions (permission_id-nya sudah tidak ada
-- setelah baris di s_permissions dihapus, jadi urutannya penting).

DELETE FROM role_permissions
WHERE permission_id IN (SELECT id FROM s_permissions WHERE name = 'users:assign-role');

DELETE FROM s_permissions WHERE name = 'users:assign-role';