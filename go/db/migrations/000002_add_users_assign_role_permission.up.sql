-- =========================================================================
-- MENAMBAHKAN PERMISSION users:assign-role (PERBAIKAN PRIVILEGE ESCALATION)
-- =========================================================================
--
-- Latar belakang:
-- CreateUser dan UpdateUser menerima `role_name` dari body request lalu
-- memetakannya tanpa batas wewenang. Akibatnya siapa pun yang hanya punya
-- permission `users:create` bisa membuat akun Super Admin, dan siapa pun yang
-- punya `users:update` bisa menaikkan role user lain. Ini privilege escalation
-- vertikal.
--
-- Perbaikan: ditetapkan role hanya boleh oleh actor yang
--   1. punya permission `users:assign-role`, dan
--   2. permission role tujuan adalah subset dari permission actor sendiri.
--
-- Migrasi ini membuat permission tersebut ada dan berikan hanya ke Super Admin.
-- Role lain tidak diberi otomatis: pemberian role adalah keputusan sadar admin,
-- bukan default yang aman diasumsikan.
--
-- Perintah: make migrateup

-- 0. Perbaiki sequence yang tertinggal dari data yang sudah ada.
--
-- s_permissions_id_seq masih di angka 2 padahal max(id) sudah 30, jadi INSERT
-- berikutnya gagal dengan "duplicate key value violates unique constraint
-- s_permissions_pkey". Ini bukan efek samping migrasi ini, tapi migrasi ini tidak
-- akan bisa jalan tanpa memperbaikinya lebih dulu.
--
-- setval memakai true supaya angka berikutnya = max(id) + 1.
SELECT setval(
    's_permissions_id_seq',
    GREATEST((SELECT COALESCE(MAX(id), 1) FROM s_permissions), 1),
    true
);

-- 1. Daftarkan permission baru.
INSERT INTO s_permissions (name, description)
VALUES (
    'users:assign-role',
    'Menetapkan atau mengubah role milik user. Wajib ada sebelum role bisa ditetapkan.'
)
ON CONFLICT (name) DO NOTHING;

-- 2. Berikan ke Super Admin saja.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM s_roles r, s_permissions p
WHERE r.name = 'Super Admin'
  AND p.name = 'users:assign-role'
ON CONFLICT (role_id, permission_id) DO NOTHING;