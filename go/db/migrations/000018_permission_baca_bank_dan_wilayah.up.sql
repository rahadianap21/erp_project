-- =========================================================================
-- PERMISSION BACA UNTUK DATA REFERENSI /api/banks DAN /api/regions
-- =========================================================================
--
-- Latar belakang:
-- Endpoint /api/banks dan /api/regions hanya GET, tetapi tidak dijaga
-- permission apa pun — hanya JWT. Keduanya membaca data referensi yang dipakai
-- form Supplier (dropdown bank dan wilayah cascading). Karena ketiga route
-- master lain (products, categories, suppliers) semuanya lewat RequirePermission,
-- celah ini membuat dua endpoint referensi jadi satu-satunya jalur baca tanpa
-- izin.
--
-- Perbaikan: dua permission baca dibuat, lalu router Go membungkus kedua
-- endpoint dengan RequirePermission. Migrasi ini hanya menyiapkan permission dan
-- memberikannya ke role yang memang sudah boleh membaca supplier — karena
-- dropdown bank & wilayah hanya muncul di form supplier.
--
-- Perintah: make migrateup

-- 0. Perbaiki sequence bila tertinggal, supaya INSERT di bawah tidak gagal
--    dengan duplicate key (pernah terjadi pada migrasi 000002).
SELECT setval(
    's_permissions_id_seq',
    GREATEST((SELECT COALESCE(MAX(id), 1) FROM s_permissions), 1),
    true
);

-- 1. Daftarkan kedua permission baca.
INSERT INTO s_permissions (name, description)
VALUES
    ('banks:read', 'Membaca daftar bank referensi untuk form supplier.'),
    ('regions:read', 'Membaca data wilayah (provinsi/kabupaten/kecamatan/desa) untuk form supplier.')
ON CONFLICT (name) DO NOTHING;

-- 2. Berikan ke setiap role yang sudah memiliki suppliers:read.
--    Dropdown bank & wilayah hanya dipakai di form supplier, jadi sumber
--    wewenangnya adalah hak baca supplier itu sendiri. Dengan begitu role
--    pembelian yang ditambahkan di masa depan otomatis ikut, tanpa perlu
--    daftar nama role yang di-hardcode.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM s_roles r
CROSS JOIN s_permissions p
WHERE p.name IN ('banks:read', 'regions:read')
  AND EXISTS (
      SELECT 1
      FROM role_permissions rp
      JOIN s_permissions sp ON sp.id = rp.permission_id
      WHERE rp.role_id = r.id
        AND sp.name = 'suppliers:read'
  )
ON CONFLICT (role_id, permission_id) DO NOTHING;
