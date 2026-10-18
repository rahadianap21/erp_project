package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"go_backend/internal/domain"
	"go_backend/internal/repository/postgres/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Test integrasi untuk sqlRoleRepository.
//
// Ketiga test di bawah membuktikan perilaku rollback: kalau salah satu statement
// di tengah transaksi gagal, database harus kembali ke kondisi sebelum transaksi
// dimulai. Sebelum ada transaksi, kondisi itu tidak pernah terpenuhi — role
// tercipta tanpa permission, atau role kehilangan seluruh permissionnya.
//
// Test ini butuh PostgreSQL nyata. Kalau DATABASE_URL tidak tersedia, test
// dilewati (skip), bukan gagal, supaya `go test ./...` tetap jalan di mesin
// tanpa database.

const permissionNgawang = "permission-yang-tidak-pernah-ada-xyz"

func newTestPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL kosong, test integrasi dilewati")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("gagal membuat pool: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("database tidak merespon, test integrasi dilewati: %v", err)
	}

	t.Cleanup(pool.Close)
	return pool
}

// permissionValid mengambil satu permission yang benar-benar ada di master.
func permissionValid(t *testing.T, pool *pgxpool.Pool) string {
	t.Helper()

	var name string
	err := pool.QueryRow(context.Background(),
		`SELECT name FROM s_permissions WHERE deleted_at IS NULL ORDER BY id LIMIT 1`).Scan(&name)
	if err != nil {
		t.Skipf("tidak ada permission master, test dilewati: %v", err)
	}
	return name
}

func roleIDByName(t *testing.T, pool *pgxpool.Pool, name string) (int32, bool) {
	t.Helper()

	var id int32
	err := pool.QueryRow(context.Background(),
		`SELECT id FROM s_roles WHERE name = $1`, name).Scan(&id)
	if err != nil {
		return 0, false
	}
	return id, true
}

func permissionCount(t *testing.T, pool *pgxpool.Pool, roleID int32) int {
	t.Helper()

	var n int
	err := pool.QueryRow(context.Background(),
		`SELECT count(*) FROM role_permissions WHERE role_id = $1`, roleID).Scan(&n)
	if err != nil {
		t.Fatalf("gagal menghitung permission: %v", err)
	}
	return n
}

// buatRoleUji membuat role dengan satu permission valid, dan mendaftarkan
// pembersihannya supaya data uji tidak pernah tertinggal.
func buatRoleUji(t *testing.T, pool *pgxpool.Pool, repo *sqlRoleRepository, permission string) (int32, string) {
	t.Helper()

	ctx := context.Background()
	nama := fmt.Sprintf("b4-test-role-%d", time.Now().UnixNano())

	if err := repo.CreateRoleWithPermissions(ctx, nama, []string{permission}); err != nil {
		t.Fatalf("gagal membuat role uji: %v", err)
	}

	roleID, ada := roleIDByName(t, pool, nama)
	if !ada {
		t.Fatalf("role uji %q tidak ditemukan setelah create", nama)
	}

	t.Cleanup(func() {
		pool.Exec(context.Background(), `DELETE FROM role_permissions WHERE role_id = $1`, roleID)
		pool.Exec(context.Background(), `DELETE FROM s_roles WHERE id = $1`, roleID)
	})

	return roleID, nama
}

// TestCreateRoleGagalDiTengahHarusRollback adalah regresi untuk
// CreateRoleWithPermissions: dulu role di-commit lebih dulu, lalu permission
// di-insert satu per satu. Permission kedua yang invalid membuat role tertinggal
// dengan hanya satu permission.
func TestCreateRoleGagalDiTengahHarusRollback(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()
	repo := &sqlRoleRepository{pool: pool, q: db.New(pool)}

	valid := permissionValid(t, pool)
	nama := fmt.Sprintf("b4-test-create-%d", time.Now().UnixNano())

	err := repo.CreateRoleWithPermissions(ctx, nama, []string{valid, permissionNgawang})
	if err == nil {
		t.Fatal("CreateRoleWithPermissions harus error saat permission tidak ada di master")
	}

	if _, ada := roleIDByName(t, pool, nama); ada {
		t.Fatal("role masih ada di database padahal create gagal: transaksi tidak rollback")
	}

	var sisa int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM role_permissions rp
		 JOIN s_roles r ON r.id = rp.role_id
		 WHERE r.name = $1`, nama).Scan(&sisa); err != nil {
		t.Fatalf("gagal memeriksa sisa relasi: %v", err)
	}
	if sisa != 0 {
		t.Fatalf("masih ada %d relasi permission yatim, harus 0", sisa)
	}
}

// TestUpdateRoleGagalDiTengahHarusRollback adalah regresi untuk
// UpdateRoleWithPermissions: dulu semua permission lama dihapus dan di-commit
// dulu, baru permission baru di-insert. Kalau insert gagal, role kehilangan
// semua permission dan semua user yang memakainya terkunci keluar dari aplikasi.
func TestUpdateRoleGagalDiTengahHarusRollback(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()
	repo := &sqlRoleRepository{pool: pool, q: db.New(pool)}

	valid := permissionValid(t, pool)
	roleID, namaAwal := buatRoleUji(t, pool, repo, valid)

	sebelum := permissionCount(t, pool, roleID)
	if sebelum != 1 {
		t.Fatalf("permission awal = %d, want 1", sebelum)
	}

	// Update dengan permission kedua yang tidak ada di master: delete lama
	// sudah dieksekusi, insert baru gagal.
	err := repo.UpdateRoleWithPermissions(ctx, roleID, "b4-test-role-diubah", []string{valid, permissionNgawang})
	if err == nil {
		t.Fatal("UpdateRoleWithPermissions harus error saat permission tidak ada di master")
	}

	sesudah := permissionCount(t, pool, roleID)
	if sesudah != sebelum {
		t.Fatalf("permission berubah setelah update gagal: sebelum=%d sesudah=%d, harus sama",
			sebelum, sesudah)
	}

	// Rollback harus mengembalikan nama lama juga, bukan hanya permission.
	var nama string
	if err := pool.QueryRow(ctx, `SELECT name FROM s_roles WHERE id = $1`, roleID).Scan(&nama); err != nil {
		t.Fatalf("gagal membaca nama role: %v", err)
	}
	if nama != namaAwal {
		t.Fatalf("nama role berubah setelah update gagal: %q, want %q (rollback tidak sempurna)", nama, namaAwal)
	}
}

// TestDeleteRoleGagalHarusRollback memakai FK RESTRICT s_users.role_id ->
// s_roles.id sebagai trigger kegagalan: role yang masih dipakai user tidak
// boleh dihapus. Tanpa transaksi, semua permission role sudah terhapus permanen
// sebelum statement kedua gagal.
func TestDeleteRoleGagalHarusRollback(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()
	repo := &sqlRoleRepository{pool: pool, q: db.New(pool)}

	valid := permissionValid(t, pool)
	roleID, namaAwal := buatRoleUji(t, pool, repo, valid)

	// Pasang user uji yang memakai role ini.
	email := fmt.Sprintf("b4-test-user-%d@erp.local", time.Now().UnixNano())
	if _, err := pool.Exec(ctx,
		`INSERT INTO s_users (email, password, role_id) VALUES ($1, 'x', $2)`,
		email, roleID); err != nil {
		t.Fatalf("gagal membuat user uji: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(context.Background(), `DELETE FROM s_users WHERE email = $1`, email)
	})

	err := repo.DeleteRoleWithPermissions(ctx, roleID)
	if err == nil {
		t.Fatal("DeleteRoleWithPermissions harus error saat role masih dipakai user")
	}

	if n := permissionCount(t, pool, roleID); n != 1 {
		t.Fatalf("permission role = %d setelah delete gagal, want 1: relasi hilang permanen", n)
	}
	// Role-nya sendiri juga harus masih ada.
	if _, ada := roleIDByName(t, pool, namaAwal); !ada {
		t.Fatal("role hilang setelah delete gagal, padahal rollback seharusnya memulihkannya")
	}
}

// TestUpdateRoleSuksesMenyimpanSemuaPermission memastikan transaksi yang sukses
// benar-benar commit, jadi perbaikan ini tidak sekadar "selalu rollback".
func TestUpdateRoleSuksesMenyimpanSemuaPermission(t *testing.T) {
	pool := newTestPool(t)
	ctx := context.Background()
	repo := &sqlRoleRepository{pool: pool, q: db.New(pool)}

	valid := permissionValid(t, pool)
	roleID, _ := buatRoleUji(t, pool, repo, valid)

	var kedua string
	if err := pool.QueryRow(ctx,
		`SELECT name FROM s_permissions WHERE deleted_at IS NULL AND name <> $1 ORDER BY id LIMIT 1`,
		valid).Scan(&kedua); err != nil {
		t.Skipf("butuh minimal dua permission master: %v", err)
	}

	if err := repo.UpdateRoleWithPermissions(ctx, roleID, "b4-test-role-sukses", []string{kedua}); err != nil {
		t.Fatalf("update yang seharusnya sukses gagal: %v", err)
	}

	var nama string
	if err := pool.QueryRow(ctx, `SELECT name FROM s_roles WHERE id = $1`, roleID).Scan(&nama); err != nil {
		t.Fatalf("gagal membaca nama role: %v", err)
	}
	if nama != "b4-test-role-sukses" {
		t.Fatalf("nama role = %q, want %q", nama, "b4-test-role-sukses")
	}

	var got string
	if err := pool.QueryRow(ctx,
		`SELECT p.name FROM role_permissions rp
		 JOIN s_permissions p ON p.id = rp.permission_id
		 WHERE rp.role_id = $1`, roleID).Scan(&got); err != nil {
		t.Fatalf("gagal membaca permission role: %v", err)
	}
	if got != kedua {
		t.Fatalf("permission role = %q, want %q (permission lama belum dibersihkan)", got, kedua)
	}
}

// ---------------------------------------------------------------------------
// OtorisasiAktif — sumber kebenaran wewenang untuk middleware
//
// Middleware memanggil method ini pada setiap request. Karena itu tiga hal di
// bawah wajib benar, dan ketiganya tidak bisa dibuktikan tanpa database nyata:
// akun nonaktif ditolak, akun terhapus ditolak, dan permission yang dikembalikan
// benar-benar milik role yang sedang dipakuuser.
// ---------------------------------------------------------------------------

// TestOtorisasiAktif_UntukUserAktif mengembalikan permission sesuai role.
func TestOtorisasiAktif_UntukUserAktif(t *testing.T) {
	pool := newTestPool(t)
	repo := NewSQLUserRepository(pool)
	ctx := context.Background()

	var userID int64
	if err := pool.QueryRow(ctx,
		`SELECT u.id FROM s_users u
		 JOIN s_roles r ON r.id = u.role_id
		 WHERE u.deleted_at IS NULL AND u.is_active
		   AND EXISTS (SELECT 1 FROM role_permissions rp WHERE rp.role_id = r.id)
		 LIMIT 1`).Scan(&userID); err != nil {
		t.Skipf("tidak ada user aktif dengan permission, test dilewati: %v", err)
	}

	hasil, err := repo.OtorisasiAktif(ctx, userID)
	if err != nil {
		t.Fatalf("user aktif harus berhasil: %v", err)
	}
	if hasil.UserID != userID {
		t.Errorf("UserID harus %d, dapat %d", userID, hasil.UserID)
	}
	if len(hasil.Permissions) == 0 {
		t.Fatal("user dengan permission tidak boleh menghasilkan daftar kosong")
	}

	// Daftar harus sama persis dengan milik role-nya di database.
	var expect []string
	rows, err := pool.Query(ctx, `
		SELECT p.name FROM role_permissions rp
		JOIN s_permissions p ON p.id = rp.permission_id
		JOIN s_users u ON u.role_id = rp.role_id
		WHERE u.id = $1 ORDER BY p.name`, userID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var nama string
		if err := rows.Scan(&nama); err != nil {
			t.Fatal(err)
		}
		expect = append(expect, nama)
	}
	if len(hasil.Permissions) != len(expect) {
		t.Fatalf("jumlah permission harus %d, dapat %d (%v)", len(expect), len(hasil.Permissions), hasil.Permissions)
	}
	for i := range expect {
		if hasil.Permissions[i] != expect[i] {
			t.Errorf("permission[%d] harus %s, mendapat %s", i, expect[i], hasil.Permissions[i])
		}
	}
}

// TestOtorisasiAktif_MenolakAkunNonAktif adalah inti dari Fix #3 yang
// sekarang berlaku di seluruh endpoint, bukan hanya /api/users.
//
// User dibuat sendiri lalu dinonaktifkan, karena menyandarkan test pada data
// nonaktif yang kebetulan ada di database berarti test diam-diam dilewati saat
// datanya cleaned up — persis bug yangExclude dari test yang kita perbaiki.
func TestOtorisasiAktif_MenolakAkunNonAktif(t *testing.T) {
	pool := newTestPool(t)
	repo := NewSQLUserRepository(pool)
	ctx := context.Background()

	roleID := roleUntukUji(t, pool)
	userID := userUjiDenganStatus(t, pool, roleID, false)

	t.Cleanup(func() {
		// Hard delete: user uji tidak boleh meninggalkan jejak di baseline.
		if _, err := pool.Exec(context.Background(), `DELETE FROM s_users WHERE id = $1`, userID); err != nil {
			t.Errorf("gagal membersihkan user uji: %v", err)
		}
	})

	_, err := repo.OtorisasiAktif(ctx, userID)
	if !errors.Is(err, domain.ErrAkunTidakBerhak) {
		t.Fatalf("akun nonaktif harus ErrAkunTidakBerhak, mendapat: %v", err)
	}
}

// TestOtorisasiAktif_MenolakAkunTidakAda memastikan user id asing ditolak
// dengan error yang sama, sehingga middleware tidak bisa membedakan
// "tidak ada" dari "nonaktif" lewat pesan error.
func TestOtorisasiAktif_MenolakAkunTidakAda(t *testing.T) {
	pool := newTestPool(t)
	repo := NewSQLUserRepository(pool)
	ctx := context.Background()

	// ID mustahil dipakai user sungguhan.
	var maksID int64
	if err := pool.QueryRow(ctx, `SELECT COALESCE(MAX(id), 0) FROM s_users`).Scan(&maksID); err != nil {
		t.Fatal(err)
	}

	if _, err := repo.OtorisasiAktif(ctx, maksID+999999); !errors.Is(err, domain.ErrAkunTidakBerhak) {
		t.Fatalf("user tidak ada harus ErrAkunTidakBerhak, mendapat: %v", err)
	}
	// UserID tidak valid juga harus ditolak, bukan leading ke query sia-sia.
	if _, err := repo.OtorisasiAktif(ctx, 0); !errors.Is(err, domain.ErrAkunTidakBerhak) {
		t.Fatalf("user id 0 harus ditolak, mendapat: %v", err)
	}
	if _, err := repo.OtorisasiAktif(ctx, -1); !errors.Is(err, domain.ErrAkunTidakBerhak) {
		t.Fatalf("user id negatif harus ditolak, mendapat: %v", err)
	}
}

// roleUntukUji mengambil role pertama yang ada, untuk dipakai user uji.
func roleUntukUji(t *testing.T, pool *pgxpool.Pool) int32 {
	t.Helper()
	var roleID int32
	err := pool.QueryRow(context.Background(), `SELECT id FROM s_roles ORDER BY id LIMIT 1`).Scan(&roleID)
	if err != nil {
		t.Skipf("tidak ada role, test dilewati: %v", err)
	}
	return roleID
}

// userUjiDenganStatus membuat user sementara. User uji diberi is_active=false
// untuk menguji penolakan akun nonaktif.
func userUjiDenganStatus(t *testing.T, pool *pgxpool.Pool, roleID int32, aktif bool) int64 {
	t.Helper()
	var userID int64
	err := pool.QueryRow(context.Background(), `
		INSERT INTO s_users (name, email, password, role_id, is_active, created_at, updated_at)
		VALUES ('Uji Otorisasi P1', $1, 'x', $2, $3, NOW(), NOW())
		RETURNING id`,
		fmt.Sprintf("uji-otorisasi-p1-%d@contoh.test", time.Now().UnixNano()), roleID, aktif).Scan(&userID)
	if err != nil {
		t.Skipf("gagal membuat user uji: %v", err)
	}
	return userID
}

// TestOtorisasiAktif_UserTanpaNama menutup risiko nyata query yang dipanggil di
// setiap request.
//
// Kolom s_users.name nullable di database, sedangkan field Go-nya string. Tanpa
// COALESCE, pemindaian baris itu gagal dan OtorisasiAktif mengembalikan error —
// yang berarti satu user tanpa nama kehilangan akses ke seluruh endpoint
// terproteksi, bukan cuma tampilnya nama yang kosong.
func TestOtorisasiAktif_UserTanpaNama(t *testing.T) {
	pool := newTestPool(t)
	repo := NewSQLUserRepository(pool)
	ctx := context.Background()

	roleID := roleUntukUji(t, pool)

	// Sengaja name diisi NULL, bukan string kosong, supaya test benar-benar
	// menguji jalur NULL di database.
	var userID int64
	err := pool.QueryRow(ctx, `
		INSERT INTO s_users (name, email, password, role_id, is_active, created_at, updated_at)
		VALUES (NULL, $1, 'x', $2, TRUE, NOW(), NOW())
		RETURNING id`,
		fmt.Sprintf("uji-tanpa-nama-p1-%d@contoh.test", time.Now().UnixNano()), roleID).Scan(&userID)
	if err != nil {
		t.Skipf("gagal membuat user uji tanpa nama: %v", err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), `DELETE FROM s_users WHERE id = $1`, userID); err != nil {
			t.Errorf("gagal membersihkan user uji: %v", err)
		}
	})

	hasil, err := repo.OtorisasiAktif(ctx, userID)
	if err != nil {
		t.Fatalf("user dengan name NULL harus tetap bisa mengotorisasi: %v", err)
	}
	if hasil.UserID != userID {
		t.Errorf("UserID harus %d, mendapat %d", userID, hasil.UserID)
	}
}
