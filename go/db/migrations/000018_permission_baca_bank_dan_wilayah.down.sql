-- Kebalikan 000018: lepaskan dan hapus permission banks:read & regions:read.
-- Hapus relasi role_permissions lebih dulu agar tidak melanggar FK.

DELETE FROM role_permissions
WHERE permission_id IN (
    SELECT id FROM s_permissions WHERE name IN ('banks:read', 'regions:read')
);

DELETE FROM s_permissions WHERE name IN ('banks:read', 'regions:read');
