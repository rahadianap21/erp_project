package usecase

import (
	"context"
	"errors"
	"fmt"
	"go_backend/internal/config"
	"go_backend/internal/domain"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type userUsecase struct {
	userRepo domain.UserRepository
	roleRepo domain.RoleRepository
	cfg      *config.Config // Tambahkan config untuk membaca JWTSecret
}

// Perbarui Constructor untuk menerima config
func NewUserUsecase(userRepo domain.UserRepository, roleRepo domain.RoleRepository, cfg *config.Config) domain.UserUsecase {
	return &userUsecase{userRepo: userRepo, roleRepo: roleRepo, cfg: cfg}
}

// DefaultRoleID adalah role paling rendah (Customer) yang dipakai kalau
// pemanggil tidak mengirim role_name sama sekali.
const DefaultRoleID int32 = 2

// CreateUser membuat user baru.
//
// actor WAJIB diisi. Sebelumnya roleName diteruskan mentah dari body request,
// sehingga siapa pun yang punya users:create bisa membuat akun Super Admin.
func (u *userUsecase) CreateUser(ctx context.Context, actor domain.Actor, name, email, password, roleName string) error {
	existingUser, _ := u.userRepo.GetByEmail(ctx, email)
	if existingUser != nil {
		return errors.New("email sudah terdaftar")
	}

	roleID, err := u.resolveRoleID(ctx, actor, roleName)
	if err != nil {
		return err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return errors.New("gagal memproses enkripsi password: " + err.Error())
	}

	newUser := &domain.User{
		Name:     name,
		Email:    email,
		Password: string(hashedPassword),
		RoleID:   roleID,
	}

	return u.userRepo.Create(ctx, newUser)
}

// permissionActor membaca permission actor yang SAAT INI dari database.
//
// Sengaja tidak memakai permission dari token JWT: token berlaku 24 jam, jadi
// permission di dalamnya bisa sudah dicabut tanpa ikut berubah.
// Membaca ulang dari database menutup celah itu dan sekaligus memastikan
// role_id yang dipakai benar-benar role yang sedang berlaku.
func (u *userUsecase) permissionActor(ctx context.Context, actor domain.Actor) ([]string, error) {
	if actor.UserID <= 0 {
		return nil, domain.NewAuthorizationError("identitas pelaku tidak valid, silakan login ulang")
	}

	actorRow, err := u.userRepo.GetByID(ctx, actor.UserID)
	if err != nil {
		return nil, domain.NewAuthorizationError("akun pelaku tidak ditemukan atau sudah dinonaktifkan")
	}

	// Status aktif diperiksa di sini, bukan di query GetByID, karena query yang
	// sama juga dipakai untuk mencari target yang akan dikelola admin. Admin
	// harus tetap bisa mengubah role akun yang sedang dinonaktifkan; kalau
	// filter is_active dipasang di query, perubahan itu jadi mustahil.
	//
	// Tanpa pemeriksaan ini, akun yang dinonaktifkan masih bisa memakai token
	// lamanya sampai kedaluwarsa karena permission tetap dimuat dari DB.
	if !actorRow.IsActive {
		return nil, domain.NewAuthorizationError("akun pelaku tidak ditemukan atau sudah dinonaktifkan")
	}

	permissions, err := u.userRepo.GetPermissionsByRoleID(ctx, actorRow.RoleID)
	if err != nil {
		return nil, fmt.Errorf("gagal memuat permission akun pelaku: %w", err)
	}

	return permissions, nil
}

// resolveRoleID menerjemahkan roleName menjadi role_id dengan tetap
// mempertahankan aturan privilege escalation.
//
// Kalau roleName kosong, hasilnya DefaultRoleID tanpa perlu permission apa pun,
// karena role Customer tidak punya satu pun permission sehingga mustahil
// menjadi eskalasi.
func (u *userUsecase) resolveRoleID(ctx context.Context, actor domain.Actor, roleName string) (int32, error) {
	if roleName == "" {
		return DefaultRoleID, nil
	}

	// Wewenang pelaku diperiksa lebih dulu, baru role tujuan dicari.
	//
	// Urutan ini penting: kalau role dicari dulu, setiap pengguna terautentikasi
	// bisa menebak nama role yang ada lewat pesan error "role tidak valid atau
	// tidak ditemukan", termasuk pengguna yang sama sekali tidak berhak
	// menetapkan role.
	permissionActor, err := u.permissionActor(ctx, actor)
	if err != nil {
		return 0, err
	}
	if !domain.Punya(permissionActor, domain.PermissionAssignRole) {
		return 0, domain.NewAuthorizationError(
			"Anda tidak punya hak %s, sehingga tidak boleh menetapkan role %q",
			domain.PermissionAssignRole, roleName,
		)
	}

	role, err := u.userRepo.GetRoleByName(ctx, roleName)
	if err != nil {
		return 0, fmt.Errorf("role tidak valid atau tidak ditemukan: %w", err)
	}

	permissionTujuan, err := u.userRepo.GetPermissionsByRoleID(ctx, role.ID)
	if err != nil {
		return 0, fmt.Errorf("gagal memuat permission role: %w", err)
	}

	if !domain.BolehMenAssignRole(permissionActor, permissionTujuan) {
		if !domain.Punya(permissionActor, domain.PermissionAssignRole) {
			return 0, domain.NewAuthorizationError(
				"Anda tidak punya hak %s, sehingga tidak boleh menetapkan role %q",
				domain.PermissionAssignRole, role.Name,
			)
		}
		return 0, domain.NewAuthorizationError(
			"tidak boleh memberi role %q karena permission-nya melebihi hak akses Anda",
			role.Name,
		)
	}

	return role.ID, nil
}

func (u *userUsecase) Login(ctx context.Context, email, password string) (string, *domain.User, []string, error) {
	// 1. Cari user berdasarkan email
	user, err := u.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return "", nil, nil, errors.New("email atau password salah")
	}

	// 2. Bandingkan password mentah dengan hash Bcrypt di database
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return "", nil, nil, errors.New("email atau password salah")
	}

	permissions, err := u.userRepo.GetPermissionsByRoleID(ctx, user.RoleID)
	if err != nil {
		return "", nil, nil, errors.New("gagal memuat hak akses akun")
	}

	// 3. Jika cocok, buat klaim data (payload) untuk JWT
	claims := jwt.MapClaims{
		"user_id":     user.ID,
		"name":        user.Name,
		"email":       user.Email,
		"role_id":     user.RoleID,
		"permissions": permissions,
		"exp":         time.Now().Add(24 * time.Hour).Unix(), // Token kedaluwarsa dalam 24 jam
	}

	// 4. Tanda tangani token menggunakan Secret Key
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(u.cfg.JWTSecret))
	if err != nil {
		return "", nil, nil, errors.New("gagal menerbitkan token keamanan")
	}

	return tokenString, user, permissions, nil
}

func (u *userUsecase) Logout(ctx context.Context) error {
	// Untuk token stateless (JWT), tidak ada operasi database yang diperlukan di sini.
	// Cukup return nil sebagai indikasi proses di layer bisnis selesai tanpa error.
	return nil
}

// GetAllUsers menangani logika pagination, search, dan filter role dari handler.
func (u *userUsecase) GetAllUsers(ctx context.Context, search, role string, page, limit int) ([]domain.User, int64, []string, error) {
	// 1. Hitung nilai offset untuk pagination database
	offset := (page - 1) * limit

	// 2. Ambil data dengan menyediakan 3 penampung (users, total, err) melalui One-Trip Query
	users, total, err := u.userRepo.GetAllUsers(ctx, search, role, offset, limit)
	if err != nil {
		return nil, 0, nil, err
	}

	var roleOptions []string

	// 2. 🌟 AMBIL DATA MASTER PERAN RIIL DARI DATABASE
	// Menggunakan limit besar (misal 100) tanpa filter pencarian agar semua peran terindeks masuk ke dropdown filter
	roles, _, err := u.roleRepo.GetAllRolesMaster(ctx, "", 0, 100)
	if err == nil {
		for _, r := range roles {
			roleOptions = append(roleOptions, r.Name)
		}
	} else {
		// Fallback cadangan yang aman jika database s_roles benar-benar kosong murni
		roleOptions = []string{}
	}

	// Kembalikan 4 nilai sesuai kontrak interface UserUsecase Anda
	return users, total, roleOptions, nil
}

// GetUsersCursor memproses pagination berbasis kursor dan menyertakan opsi role untuk UI filter
func (u *userUsecase) GetUsersCursor(ctx context.Context, params domain.UserCursorParams) (*domain.UserCursorResult, error) {
	result, err := u.userRepo.GetUsersCursor(ctx, params)
	if err != nil {
		return nil, err
	}

	// Ambil daftar peran riil dari database untuk dropdown opsi filter
	roles, _, err := u.roleRepo.GetAllRolesMaster(ctx, "", 0, 100)
	if err == nil {
		for _, r := range roles {
			result.RoleOptions = append(result.RoleOptions, r.Name)
		}
	} else {
		result.RoleOptions = []string{}
	}

	return result, nil
}

func (u *userUsecase) DeleteUser(ctx context.Context, id int64) error {
	return u.userRepo.DeleteUser(ctx, id)
}

// UpdateUser memperbarui data user, termasuk role-nya.
//
// Dua lapis proteksi:
//
//  1. Role tujuan harus berada di bawah wewenang actor (lihat resolveRoleID).
//  2. User yang sedang diedit juga harus berada di bawah wewenang actor.
//     Tanpa ini, actor berkuasa rendah masih bisa mengunci akun admin dengan
//     mengganti email-nya, meskipun tidak bisa menaikkan role-nya.
func (u *userUsecase) UpdateUser(ctx context.Context, actor domain.Actor, id int64, name string, email string, roleName string) error {
	// 1. Cari role_id berdasarkan nama role dari frontend
	roleID, err := u.resolveRoleID(ctx, actor, roleName)
	if err != nil {
		return err
	}

	// 2. Tolak jika actor mencoba mengedit user dengan role yang lebih tinggi.
	target, err := u.userRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("pengguna tidak ditemukan: %w", err)
	}

	if target.RoleID != roleID {
		permissionActor, err := u.permissionActor(ctx, actor)
		if err != nil {
			return err
		}
		permissionTarget, err := u.userRepo.GetPermissionsByRoleID(ctx, target.RoleID)
		if err != nil {
			return fmt.Errorf("gagal memuat permission role pengguna: %w", err)
		}
		if !domain.Subset(permissionTarget, permissionActor) {
			return domain.NewAuthorizationError(
				"tidak boleh mengubah pengguna dengan role %q karena hak aksesnya melebihi hak akses Anda",
				target.RoleName,
			)
		}
	}

	// 3. Eksekusi update ke database menggunakan params sqlc
	return u.userRepo.UpdateUser(ctx, domain.UpdateUserParams{
		ID:     id,
		Name:   name,
		Email:  email,
		RoleID: roleID,
		// Ikutkan role yang baru saja diperiksa. Repository memakai nilai ini
		// sebagai compare-and-swap agar perubahan role oleh admin lain tidak
		// lolos di antara pemeriksaan wewenang dan penulisan.
		ExpectedRoleID: target.RoleID,
	})
}
