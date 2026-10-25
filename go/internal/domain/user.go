package domain

import (
	"context"
	"errors"
)

type User struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	RoleID   int32  `json:"role_id"`
	RoleName string `json:"role_name"` // Kolom baru untuk menampung nama role

	// RoleApp menandai aplikasi yang melekat pada role user: 'erp' atau 'pos'.
	// Dipakai frontend untuk mengarahkan pengguna setelah login dan mengunci
	// kasir agar hanya bisa membuka area /pos.
	RoleApp string `json:"role_app"`

	// IsActive menandai akun yang masih boleh dipakai. Query sudah selalu
	// mengambil kolom ini, tapi sebelumnya nilainya dibuang di pemetaan, sehingga
	// lapisan otorisasi tidak punya cara menolak akun yang dinonaktifkan.
	IsActive bool `json:"is_active"`
}

// 🌟 TAMBAHKAN: Definisikan struct UpdateUserParams di level domain agar dikenali
type UpdateUserParams struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	RoleID int32  `json:"role_id"`
	// ExpectedRoleID adalah role yang dibaca saat pemeriksaan wewenang. UPDATE
	// hanya boleh berhasil bila role target belum berubah (compare-and-swap).
	ExpectedRoleID int32 `json:"expected_role_id"`
}

// Actor adalah identitas pelaku yang sedang berinteraksi.
//
// Sengaja HANYA berisi UserID. Role dan permission TIDAK ikut diambil dari
// token JWT, karena token berlaku 24 jam: kalau role actor dicabut, token lamanya
// masih membawa permission lama dan cek akan lolos memakai hak yang sudah dicabut.
//
// UserID aman diambil dari token karena nilainya tidak membawa wewenang dan
// tidak berubah. Role serta permission aktual selalu dibaca ulang dari database.
type Actor struct {
	UserID int64
}

// PermissionAssignRole adalah permission yang mengizinkan menetapkan role.
const PermissionAssignRole = "users:assign-role"

// Punya melapor apakah daftar permission memuat permission tertentu.
func Punya(permissions []string, permission string) bool {
	for _, p := range permissions {
		if p == permission {
			return true
		}
	}
	return false
}

// Subset melapor apakah semua permission di `smaller` juga ada di `larger`.
//
// Role dengan 0 permission (mis. Customer) selalu subset dari apa pun, jadi
// actor dengan hak akses apa pun boleh membuat user Customer.
func Subset(smaller, larger []string) bool {
	have := make(map[string]struct{}, len(larger))
	for _, p := range larger {
		have[p] = struct{}{}
	}
	for _, p := range smaller {
		if _, ok := have[p]; !ok {
			return false
		}
	}
	return true
}

// BolehMenAssignRole menerapkan dua aturan sekaligus:
//
//  1. Actor harus punya permission users:assign-role.
//  2. Permission role tujuan harus subset dari permission actor.
//
// Aturan 2 menutup jalur eskalasi bahkan kalau users:assign-role somehow
// bocor ke role yang sebenarnya tidak punya wewenang penuh.
func BolehMenAssignRole(permissionActor, permissionTujuan []string) bool {
	if !Punya(permissionActor, PermissionAssignRole) {
		return false
	}
	return Subset(permissionTujuan, permissionActor)
}

type UserCursorParams struct {
	Search    string
	Role      string
	CursorID  int64
	Direction string // "next" or "prev"
	Limit     int
}

type UserCursorResult struct {
	Users       []User   `json:"users"`
	NextCursor  *int64   `json:"next_cursor"`
	PrevCursor  *int64   `json:"prev_cursor"`
	HasNext     bool     `json:"has_next"`
	HasPrev     bool     `json:"has_prev"`
	Total       int64    `json:"total"`
	RoleOptions []string `json:"role_options,omitempty"`
}

type UserRepository interface {
	Create(ctx context.Context, user *User) error
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id int64) (*User, error)
	GetPermissionsByRoleID(ctx context.Context, roleID int32) ([]string, error)
	GetAllUsers(ctx context.Context, search, role string, offset, limit int) ([]User, int64, error)
	GetUsersCursor(ctx context.Context, params UserCursorParams) (*UserCursorResult, error)
	DeleteUser(ctx context.Context, id int64) error
	GetRoleByName(ctx context.Context, name string) (Role, error)
	UpdateUser(ctx context.Context, arg UpdateUserParams) error
}

// OtorisasiUser adalah wewenang yang berlaku untuk satu user pada saat
// request datang, dibaca langsung dari database.
type OtorisasiUser struct {
	UserID      int64
	RoleID      int32
	App         string
	Permissions []string
}

// ErrAkunTidakBerhak menandai bahwa sebuah akun tidak boleh dipakai untuk
// menunggu keputusan: akun tidak ada, sudah dihapus, atau dinonaktifkan.
//
// Dipakai middleware dan usecase supaya keduanya menolak dengan cara yang sama
// dan tanpa membocorkan informasi yang berbeda antara "tidak ada" dan
// "nonaktif".
var ErrAkunTidakBerhak = errors.New("akun pelaku tidak ditemukan atau sudah dinonaktifkan")

// ErrUserChanged menandai bahwa role/jenis data user berubah setelah pemeriksaan
// wewenang tetapi sebelum penulisan. Pemanggil harus memuat ulang data supaya
// keputusan otorisasi tidak diterapkan pada user yang sudah berbeda.
var ErrUserChanged = errors.New("data pengguna berubah oleh permintaan lain, silakan muat ulang")

// PermissionResolver menyediakan wewenang yang berlaku SAAT INI untuk satu
// user, dibaca langsung dari database.
//
// Dipakai middleware otorisasi. Alasannya: token JWT berumur 24 jam dan role_id
// serta permission ikut ditanam di dalamnya. Akibatnya pencabutan permission
// maupun perpindahan role tidak berlaku sebelum semua token lama kedaluwarsa —
// pada `/api/users` yang sudah ditutup di usecase, tapi pada endpoint lain yang
// dijaga RequirePermission di handler, klaim itulah yang dipakai. Resolve dari
// database membuat perubahan langsung berlaku pada request berikutnya.
//
// Interface ini sengaja kecil supaya layer middleware tidak perlu bergantung
// pada repository konkret.
type PermissionResolver interface {
	// OtorisasiAktif mengembalikan role dan permission milik user berdasarkan
	// Condition yang sedang berlaku di database.
	//
	// ErrAkunTidakBerhak dikembalikan bila akun tidak ada atau sudah
	// dinonaktifkan. Error lain (mis. database gagal) dibiarkan apa adanya
	// supaya middleware membalas 500, bukan 403 — 403 akan menyamar jadi
	// keputusan wewenang padahal masalahnya kegagalan sistem.
	OtorisasiAktif(ctx context.Context, userID int64) (*OtorisasiUser, error)
}

type UserUsecase interface {
	CreateUser(ctx context.Context, actor Actor, name, email, password, roleName string) error
	Login(ctx context.Context, email, password string) (string, *User, []string, error)
	Logout(ctx context.Context) error
	GetAllUsers(ctx context.Context, search, role string, page, limit int) ([]User, int64, []string, error)
	GetUsersCursor(ctx context.Context, params UserCursorParams) (*UserCursorResult, error)
	DeleteUser(ctx context.Context, id int64) error
	UpdateUser(ctx context.Context, actor Actor, id int64, name string, email string, roleName string) error
}
