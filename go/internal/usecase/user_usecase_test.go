package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"

	"go_backend/internal/config"
	"go_backend/internal/domain"
)

// repoPalsu adalah implementasi UserRepository berbasis slice, cukup untuk
// menguji logika wewenang tanpa database.
type repoPalsu struct {
	users      map[string]domain.User // keyed by email
	roles      map[string]domain.Role // keyed by name
	perms      map[int32][]string     // role_id -> permissions
	dibuat     []string               // email yang berhasil dibuat
	diperbarui []domain.UpdateUserParams
}

func newRepoPalsu() *repoPalsu {
	return &repoPalsu{
		users: map[string]domain.User{},
		roles: map[string]domain.Role{},
		perms: map[int32][]string{},
	}
}

func (r *repoPalsu) Create(_ context.Context, u *domain.User) error {
	r.dibuat = append(r.dibuat, u.Email)
	u.ID = int64(len(r.dibuat))
	r.users[u.Email] = *u
	return nil
}

func (r *repoPalsu) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	u, ada := r.users[email]
	if !ada {
		return nil, errors.New("tidak ditemukan")
	}
	return &u, nil
}

func (r *repoPalsu) GetByID(_ context.Context, id int64) (*domain.User, error) {
	for _, u := range r.users {
		if u.ID == id {
			cp := u
			return &cp, nil
		}
	}
	return nil, errors.New("tidak ditemukan")
}

func (r *repoPalsu) GetPermissionsByRoleID(_ context.Context, roleID int32) ([]string, error) {
	return r.perms[roleID], nil
}

func (r *repoPalsu) GetAllUsers(context.Context, string, string, int, int) ([]domain.User, int64, error) {
	return nil, 0, nil
}

func (r *repoPalsu) GetUsersCursor(context.Context, domain.UserCursorParams) (*domain.UserCursorResult, error) {
	return nil, nil
}

func (r *repoPalsu) DeleteUser(context.Context, int64) error { return nil }

func (r *repoPalsu) GetRoleByName(_ context.Context, name string) (domain.Role, error) {
	role, ada := r.roles[name]
	if !ada {
		return domain.Role{}, errors.New("tidak ditemukan")
	}
	return role, nil
}

func (r *repoPalsu) UpdateUser(_ context.Context, p domain.UpdateUserParams) error {
	r.diperbarui = append(r.diperbarui, p)
	return nil
}

// skenario menyusun dunia uji yang sudah ditata.
func skenario() *repoPalsu {
	r := newRepoPalsu()

	// Super Admin: memegang SEMUA permission yang dipakai di skenario ini,
	// jadi ia benar-benar superset dari role lain dan boleh menetapkan siapa pun.
	r.roles["Super Admin"] = domain.Role{ID: 1, Name: "Super Admin"}
	r.perms[1] = []string{"users:read", "users:create", "po:read", domain.PermissionAssignRole}

	// Admin 4: cuma users:create, tanpa users:assign-role.
	r.roles["Admin 4"] = domain.Role{ID: 6, Name: "Admin 4"}
	r.perms[6] = []string{"users:create"}

	// Customer: 0 permission.
	r.roles["Customer"] = domain.Role{ID: 2, Name: "Customer"}
	r.perms[2] = nil

	// Admin Kabag: punya users:assign-role tapi hanya 1 permission.
	r.roles["Admin Kabag"] = domain.Role{ID: 7, Name: "Admin Kabag"}
	r.perms[7] = []string{domain.PermissionAssignRole, "po:read"}

	// Pelaku.
	r.users["super@mail.com"] = domain.User{ID: 1, Email: "super@mail.com", RoleID: 1, RoleName: "Super Admin", IsActive: true}
	r.users["admin4@mail.com"] = domain.User{ID: 2, Email: "admin4@mail.com", RoleID: 6, RoleName: "Admin 4", IsActive: true}
	r.users["kabag@mail.com"] = domain.User{ID: 3, Email: "kabag@mail.com", RoleID: 7, RoleName: "Admin Kabag", IsActive: true}

	return r
}

func ucBaru(r *repoPalsu) *userUsecase {
	return &userUsecase{
		userRepo: r,
		roleRepo: nil,
		cfg:      &config.Config{JWTSecret: "secret-khusus-test-000000000"},
	}
}

var (
	actorSuper = domain.Actor{UserID: 1}
	actorA4    = domain.Actor{UserID: 2}
	actorKabag = domain.Actor{UserID: 3}
)

// TestCreateUserEkskalasiDitolak adalah regresi untuk bug yang allows
// user dengan `users:create` membuat akun Super Admin.
func TestCreateUserEkskalasiDitolak(t *testing.T) {
	r := skenario()
	u := ucBaru(r)

	// Admin 4 hanya punya users:create. Dia mencoba membuat Super Admin.
	err := u.CreateUser(context.Background(), actorA4, "Backdoor", "backdoor@mail.com", "Rahasia123!", "Super Admin")

	if err == nil {
		t.Fatal("CreateUser harus menolak pemberian role Super Admin oleh actor berkuasa rendah")
	}
	if !strings.Contains(err.Error(), domain.PermissionAssignRole) {
		t.Fatalf("pesan error harus menyebut %s, dapat: %v", domain.PermissionAssignRole, err)
	}
	if len(r.dibuat) != 0 {
		t.Fatalf("user tetap dibuat padahal ditolak: %v", r.dibuat)
	}
}

// TestCreateUserDenganAssignRoleTapiSubsetMelebihiDitolak menguji aturan kedua:
// punya users:assign-role saja tidak cukup kalau permission tujuan melebihi.
func TestCreateUserDenganAssignRoleTapiSubsetMelebihiDitolak(t *testing.T) {
	r := skenario()
	u := ucBaru(r)

	// Admin Kabag punya users:assign-role tapi Super Admin punya 3 permission.
	err := u.CreateUser(context.Background(), actorKabag, "Naik", "naik@mail.com", "Rahasia123!", "Super Admin")

	if err == nil {
		t.Fatal("CreateUser harus menolak role yang permission-nya melebihi hak akses actor")
	}
	if !strings.Contains(err.Error(), "melebihi") {
		t.Fatalf("pesan error harus menyebut 'melebihi', dapat: %v", err)
	}
	if len(r.dibuat) != 0 {
		t.Fatalf("user tetap dibuat padahal ditolak: %v", r.dibuat)
	}
}

// TestCreateUserSubsetYangLebihRendahDiizinkan memastikan fix tidak memblokir
// penggunaan sah: Super Admin (3 perm) boleh membuat Admin Kabag (2 perm).
func TestCreateUserSubsetYangLebihRendahDiizinkan(t *testing.T) {
	r := skenario()
	u := ucBaru(r)

	if err := u.CreateUser(context.Background(), actorSuper, "Kabag Baru", "kabag.baru@mail.com", "Rahasia123!", "Admin Kabag"); err != nil {
		t.Fatalf("Super Admin harus boleh membuat user dengan role ber-permission lebih rendah: %v", err)
	}
	if len(r.dibuat) != 1 || r.dibuat[0] != "kabag.baru@mail.com" {
		t.Fatalf("user tidak dibuat: %v", r.dibuat)
	}
	if got := r.users["kabag.baru@mail.com"].RoleID; got != 7 {
		t.Fatalf("RoleID = %d, want 7", got)
	}
}

// TestCreateUserRoleTanpaPermissionTetapButuhAssignRole mengunci aturan ketat
// dari "keduanya digabung": role tujuan tidak punya permission apa pun,
// pun, actor TANPA users:assign-role tetap tidak boleh menetapkannya.
//
// Yang bebas hanya role_name kosong, yang memakai DefaultRoleID tanpa perlu
// permission apa pun (lihat TestCreateUserRoleKosongPakaiDefault).
func TestCreateUserRoleTanpaPermissionTetapButuhAssignRole(t *testing.T) {
	r := skenario()
	u := ucBaru(r)

	// Admin 4 tidak punya users:assign-role.
	err := u.CreateUser(context.Background(), actorA4, "Pembeli", "pembeli@mail.com", "Rahasia123!", "Customer")
	if err == nil {
		t.Fatal("tanpa users:assign-role, role_name tidak boleh ditetapkan meski role-nya tanpa permission")
	}
	if !strings.Contains(err.Error(), domain.PermissionAssignRole) {
		t.Fatalf("pesan error harus menyebut %s, dapat: %v", domain.PermissionAssignRole, err)
	}
	if len(r.dibuat) != 0 {
		t.Fatalf("user tetap dibuat padahal ditolak: %v", r.dibuat)
	}
}

// TestCreateUserRoleTanpaPermissionBolehJikaPunyaAssignRole: begitu actor punya
// users:assign-role, role tanpa permission bebas ditetapkan karena 0 permission
// selalu subset dari apa pun.
func TestCreateUserRoleTanpaPermissionBolehJikaPunyaAssignRole(t *testing.T) {
	r := skenario()
	u := ucBaru(r)

	// Admin Kabag punya users:assign-role.
	if err := u.CreateUser(context.Background(), actorKabag, "Pembeli", "pembeli@mail.com", "Rahasia123!", "Customer"); err != nil {
		t.Fatalf("role tanpa permission harus boleh ditetapkan oleh holder users:assign-role: %v", err)
	}
	if got := r.users["pembeli@mail.com"].RoleID; got != 2 {
		t.Fatalf("RoleID = %d, want 2 (Customer)", got)
	}
}

// TestCreateUserRoleKosongPakaiDefault memastikan jalur lama (tanpa role_name)
// tetap bekerja dan tidak butuh permission apa pun.
func TestCreateUserRoleKosongPakaiDefault(t *testing.T) {
	r := skenario()
	u := ucBaru(r)

	if err := u.CreateUser(context.Background(), actorA4, "Tanpa Role", "tanpa.role@mail.com", "Rahasia123!", ""); err != nil {
		t.Fatalf("role_name kosong harus memakai role default: %v", err)
	}
	if got := r.users["tanpa.role@mail.com"].RoleID; got != DefaultRoleID {
		t.Fatalf("RoleID = %d, want %d", got, DefaultRoleID)
	}
}

// TestUpdateUserEkskalasiDitolak adalah regresi untuk jalur PUT.
func TestUpdateUserEkskalasiDitolak(t *testing.T) {
	r := skenario()
	u := ucBaru(r)

	// Admin Kabag (2 perm, punya assign-role) mencoba naikkan Admin 4 jadi Super Admin.
	err := u.UpdateUser(context.Background(), actorKabag, 2, "Admin 4", "admin4@mail.com", "Super Admin")

	if err == nil {
		t.Fatal("UpdateUser harus menolak kenaikan role melewati batas wewenang")
	}
	if len(r.diperbarui) != 0 {
		t.Fatalf("data tetap ter-update padahal ditolak: %v", r.diperbarui)
	}
}

// TestUpdateUserEditUserBerkuasaTinggiDitolak menguji lapis kedua: actor tidak
// boleh menyentuh user yang role-nya di atas dengannya, walau role tujuan
// update sendiri tidak naik privilege.
func TestUpdateUserEditUserBerkuasaTinggiDitolak(t *testing.T) {
	r := skenario()
	u := ucBaru(r)

	// Admin Kabag mencoba mengedit Super Admin, tetap dengan role Super Admin.
	// Role tujuannya "diizinkan" oleh cek pertama, tapi user targetnya
	// berkuasa lebih tinggi sehingga harus ditolak.
	err := u.UpdateUser(context.Background(), actorKabag, 1, "Super", "super@mail.com", "Super Admin")

	if err == nil {
		t.Fatal("UpdateUser harus menolak edit atas user dengan role yang lebih tinggi")
	}
	if !strings.Contains(err.Error(), "melebihi") {
		t.Fatalf("pesan error harus menjelaskan role target melebihi, dapat: %v", err)
	}
	if len(r.diperbarui) != 0 {
		t.Fatalf("data tetap ter-update padahal ditolak: %v", r.diperbarui)
	}
}

// TestUpdateUserSendiriDiizinkan: actor boleh mengedit
// dirinya sendiri selama tidak menaikkan privilege.
func TestUpdateUserSendiriDiizinkan(t *testing.T) {
	r := skenario()
	u := ucBaru(r)

	// Super Admin mengedit dirinya sendiri, tetap Super Admin.
	if err := u.UpdateUser(context.Background(), actorSuper, 1, "Super Admin Baru", "super@mail.com", "Super Admin"); err != nil {
		t.Fatalf("Super Admin harus boleh mengedit dirinya sendiri: %v", err)
	}
	if len(r.diperbarui) != 1 {
		t.Fatalf("data tidak ter-update: %v", r.diperbarui)
	}
	if r.diperbarui[0].RoleID != 1 {
		t.Fatalf("RoleID = %d, want 1", r.diperbarui[0].RoleID)
	}
}

// TestAktorTanpaLoginDitolak memastikan Actor kosong tidak bisa sneaking lewat.
func TestAktorTanpaLoginDitolak(t *testing.T) {
	r := skenario()
	u := ucBaru(r)

	err := u.CreateUser(context.Background(), domain.Actor{}, "X", "x@mail.com", "Rahasia123!", "Super Admin")
	if err == nil {
		t.Fatal("tanpa identitas pelaku, penambahan user harus ditolak")
	}
	if !strings.Contains(err.Error(), "identitas pelaku") {
		t.Fatalf("pesan error harus menyebut identitas pelaku, dapat: %v", err)
	}
}

// TestPermissionDicekDariDatabaseBukanToken memastikan keputusan memakai role
// dan permission aktual. Actors dengan UserID yang role-nya sudah berubah harus
// memakai role terbaru, bukan role yang pernah ada di token lama.
func TestPermissionDicekDariDatabaseBukanToken(t *testing.T) {
	r := skenario()
	u := ucBaru(r)

	// Actor 2 awalnya punya 1 permission (users:create).
	// Sekarang role-nya dinaikkan jadi Super Admin tanpa melalui API.
	r.users["admin4@mail.com"] = domain.User{ID: 2, Email: "admin4@mail.com", RoleID: 1, RoleName: "Super Admin", IsActive: true}

	if err := u.CreateUser(context.Background(), actorA4, "Setelah Naik", "sesudah.naik@mail.com", "Rahasia123!", "Super Admin"); err != nil {
		t.Fatalf("actor yang role-nya sudah naik di DB harus memakai wewenang terbaru: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Uji_EQ: akun nonaktif & target guard
//
// Dua hal ini tidaktertangkap test sebelumnya:
//
//  1. GetUserByIDWithRole hanya memfilter deleted_at, bukan is_active, dan
//     pemetaan repository membuang kolom is_active. Akibatnya akun yang
//     dinonaktifkan tetap bisa memakai token lamanya sampai kedaluwarsa.
//     Perbaikannya dipasang di permissionActor, bukan di query, supaya admin
//     tetap bisa mengelola akun yang sedang dinonaktifkan.
//  2. Guard target harus berdasarkan role target, dan role tanpa permission sama sekali
//     (Customer) adalah kasus paling_remote: actor yang tidak punya
//     users:assign-role seharusnya ditolak, bukan lolos karena "role target
//     selalu subset".
// ---------------------------------------------------------------------------

func TestAkunNonAktifTidakBisaBertindakLagi(t *testing.T) {
	r := skenario()
	u := ucBaru(r)

	// Akun Super Admin yang sama, sekarang dinonaktifkan.
	r.users["super@mail.com"] = domain.User{
		ID: 1, Email: "super@mail.com", RoleID: 1, RoleName: "Super Admin", IsActive: false,
	}

	err := u.CreateUser(context.Background(), domain.Actor{UserID: 1},
		"Baru", "baru@mail.com", "Rahasia123!", "Admin 4")
	if err == nil {
		t.Fatal("akun nonaktif harus ditolak, bukan boleh acted")
	}
	if !strings.Contains(err.Error(), "dinonaktifkan") {
		t.Errorf("pesan harus menyebut akun nonaktif, dapat: %s", err.Error())
	}
}

func TestRoleTargetTanpaPermissionTetapPerluAssignRole(t *testing.T) {
	r := skenario()
	u := ucBaru(r)

	// Super Admin memegang users:assign-role dan role Customer tidak punya
	// permission, jadi penetapan role Customer harus berhasil.
	if err := u.CreateUser(context.Background(), domain.Actor{UserID: 1},
		"CS", "cs@mail.com", "Rahasia123!", "Customer"); err != nil {
		t.Fatalf("Super Admin boleh membuat user Customer: %v", err)
	}

	// Admin 4 hanya punya users:create. Role Customer tidak punya permission
	// apa pun sehingga selalu jadi subset dari actor — kalau hanya aturan subset
	// yang dipakai, permintaan ini akan lolos. bordering: penamaan role yang
	// eksplisit tetap memerlukan users:assign-role.
	err := u.CreateUser(context.Background(), actorA4,
		"CS2", "cs2@mail.com", "Rahasia123!", "Customer")
	if err == nil {
		t.Fatal("tanpa users:assign-role, role Customer eksplisit harus ditolak")
	}
	if !strings.Contains(err.Error(), "users:assign-role") {
		t.Errorf("pesan harus menyebut users:assign-role, dapat: %s", err.Error())
	}
}

// TestRoleTidakAdaTidakBocorkanKePenggunaTanpaHak memastikan actor tanpa
// users:assign-role tidak bisa memakai pesan error untuk menebak role mana yang
// ada di sistem.
func TestRoleTidakAdaTidakBocorkanKePenggunaTanpaHak(t *testing.T) {
	r := skenario()
	u := ucBaru(r)

	// Admin 4 hanya punya users:create.
	err := u.CreateUser(context.Background(), actorA4,
		"X", "x@mail.com", "Rahasia123!", "RoleYangTidakAdaSekali")
	if err == nil {
		t.Fatal("actor tanpa users:assign-role harus ditolak")
	}
	if !strings.Contains(err.Error(), domain.PermissionAssignRole) {
		t.Fatalf("pesan harus menyebut %s, bukan membocorkan nama role: %v",
			domain.PermissionAssignRole, err)
	}
	if len(r.dibuat) != 0 {
		t.Fatalf("user tetap dibuat padahal ditolak: %v", r.dibuat)
	}
}
