package postgres

import (
	"context"
	"go_backend/internal/domain"
	"go_backend/internal/repository/postgres/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

type sqlUserRepository struct {
	q *db.Queries
}

// UserRepositoryWithResolver menggabungkan kontrak repository user dengan
// kontrak resolver permission supaya satu objek bisa dipakai untuk dua keperluan
// tanpa perlu membuat instance terpisah.
type UserRepositoryWithResolver interface {
	domain.UserRepository
	domain.PermissionResolver
}

func NewSQLUserRepository(pool *pgxpool.Pool) UserRepositoryWithResolver {
	return &sqlUserRepository{q: db.New(pool)}
}

func (r *sqlUserRepository) Create(ctx context.Context, user *domain.User) error {
	insertedUser, err := r.q.CreateUser(ctx, db.CreateUserParams{
		Name:     user.Name,
		Email:    user.Email,
		Password: user.Password,
		RoleID:   user.RoleID,
	})
	if err != nil {
		// Pemeriksaan GetByEmail di usecase tidak atomik. Saat dua pendaftaran
		// dengan email sama berjalan bersamaan, unique index parsial di
		// database-lah yang menang; terjemahkan pelanggarannya agar pemanggil
		// menerima 409, bukan 500.
		return mapDuplicate(err, "email sudah terdaftar")
	}

	user.ID = insertedUser.ID
	return nil
}

func (r *sqlUserRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	row, err := r.q.GetUserByEmailWithRole(ctx, email)
	if err != nil {
		return nil, err
	}

	return &domain.User{
		ID:       row.ID,
		Name:     row.Name,
		Email:    row.Email,
		Password: row.Password,
		RoleID:   row.RoleID,
		RoleName: row.RoleName,
		RoleApp:  row.RoleApp,
	}, nil
}

// GetByID dipakai untuk memeriksa role milik user yang sedang diedit, supaya
// eskalasi horizontal (actor berkuasa rendah mengedit akun berkuasa tinggi)
// bisa ditolak sebelum data tersimpan.
func (r *sqlUserRepository) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	row, err := r.q.GetUserByIDWithRole(ctx, id)
	if err != nil {
		return nil, err
	}

	return &domain.User{
		ID:       row.ID,
		Name:     row.Name,
		Email:    row.Email,
		Password: row.Password,
		RoleID:   row.RoleID,
		RoleName: row.RoleName,
		RoleApp:  row.RoleApp,
		IsActive: row.IsActive,
	}, nil
}

// OtorisasiAktif mengembalikan role_id dan permission yang benar-benar berlaku
// untuk user pada saat ini, dibaca dari database.
//
// Dipakai middleware otorisasi supaya pencabutan permission, perpindahan role,
// dan penonaktifan akun langsung berlaku pada request berikutnya. Kalau semua itu
// dibaca dari klaim JWT, perubahannya baru terasa setelah token kedaluwarsa —
// token berumur 24 jam.
//
// Status aktif ikut diperiksa di sini. GetByID sengaja tidak memfilter
// is_active supaya admin tetap bisa mengelola akun yang sedang dinonaktifkan;
// pemeriksaannya dipindah ke sini karena di middleware-lah yang memang butuh
// menolak, bukan di tempat mencari target.
func (r *sqlUserRepository) OtorisasiAktif(ctx context.Context, userID int64) (*domain.OtorisasiUser, error) {
	if userID <= 0 {
		return nil, domain.ErrAkunTidakBerhak
	}

	user, err := r.GetByID(ctx, userID)
	if err != nil {
		// Akun tidak ada, sudah dihapus, atau database sedang bermasalah. Untuk
		// middleware semuanya berarti "tidak berhak"; kegagalan database sendiri
		// sudahMAYun dibiarkan apa adanya oleh pemanggil lewat error lain.
		return nil, domain.ErrAkunTidakBerhak
	}
	if !user.IsActive {
		return nil, domain.ErrAkunTidakBerhak
	}

	permissions, err := r.GetPermissionsByRoleID(ctx, user.RoleID)
	if err != nil {
		return nil, err
	}

	return &domain.OtorisasiUser{
		UserID:      user.ID,
		RoleID:      user.RoleID,
		App:         user.RoleApp,
		Permissions: permissions,
	}, nil
}

func (r *sqlUserRepository) GetPermissionsByRoleID(ctx context.Context, roleID int32) ([]string, error) {
	permissions, err := r.q.GetPermissionsByRoleID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	return permissions, nil
}

// GetAllUsers menarik data pengguna murni terpaginasi sekaligus total baris datanya (One-Trip)
func (r *sqlUserRepository) GetAllUsers(ctx context.Context, search, role string, offset, limit int) ([]domain.User, int64, error) {
	rows, err := r.q.GetAllUsersPaginated(ctx, db.GetAllUsersPaginatedParams{
		SearchQuery: search,
		RoleFilter:  role,
		ReqLimit:    int32(limit),
		ReqOffset:   int32(offset),
	})
	if err != nil {
		return nil, 0, err
	}

	var users []domain.User
	var totalCount int64 = 0

	if len(rows) > 0 {
		totalCount = rows[0].TotalCount
	}

	for _, row := range rows {
		users = append(users, domain.User{
			ID:       row.ID,
			Name:     row.Name,
			Email:    row.Email,
			RoleID:   row.RoleID,
			RoleName: row.RoleName,
		})
	}

	return users, totalCount, nil
}

// GetUsersCursor mengambil data pengguna menggunakan teknik cursor-based pagination
func (r *sqlUserRepository) GetUsersCursor(ctx context.Context, params domain.UserCursorParams) (*domain.UserCursorResult, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = 25
	}
	fetchLimit := int32(limit + 1)

	var users []domain.User
	var hasMore bool

	if params.Direction == "prev" && params.CursorID > 0 {
		rows, err := r.q.GetUsersCursorPrev(ctx, db.GetUsersCursorPrevParams{
			SearchQuery: params.Search,
			RoleFilter:  params.Role,
			CursorID:    params.CursorID,
			ReqLimit:    fetchLimit,
		})
		if err != nil {
			return nil, err
		}

		if len(rows) > limit {
			hasMore = true
			rows = rows[:limit]
		}

		// Balik (reverse) hasil karena query ASC
		for i := len(rows) - 1; i >= 0; i-- {
			users = append(users, domain.User{
				ID:       rows[i].ID,
				Name:     rows[i].Name,
				Email:    rows[i].Email,
				RoleID:   rows[i].RoleID,
				RoleName: rows[i].RoleName,
			})
		}
	} else {
		rows, err := r.q.GetUsersCursorNext(ctx, db.GetUsersCursorNextParams{
			SearchQuery: params.Search,
			RoleFilter:  params.Role,
			CursorID:    params.CursorID,
			ReqLimit:    fetchLimit,
		})
		if err != nil {
			return nil, err
		}

		if len(rows) > limit {
			hasMore = true
			rows = rows[:limit]
		}

		for _, row := range rows {
			users = append(users, domain.User{
				ID:       row.ID,
				Name:     row.Name,
				Email:    row.Email,
				RoleID:   row.RoleID,
				RoleName: row.RoleName,
			})
		}
	}

	totalCount, _ := r.q.CountUsers(ctx, db.CountUsersParams{
		SearchQuery: params.Search,
		RoleFilter:  params.Role,
	})

	result := &domain.UserCursorResult{
		Users: users,
		Total: totalCount,
	}

	if len(users) > 0 {
		firstID := users[0].ID
		lastID := users[len(users)-1].ID

		if params.Direction == "prev" {
			result.HasPrev = hasMore
			result.HasNext = true
			if result.HasPrev {
				result.PrevCursor = &firstID
			}
			result.NextCursor = &lastID
		} else {
			result.HasNext = hasMore
			result.HasPrev = params.CursorID > 0
			if result.HasNext {
				result.NextCursor = &lastID
			}
			if result.HasPrev {
				result.PrevCursor = &firstID
			}
		}
	}

	return result, nil
}

// 🌟 PERBAIKAN: Method GetAllUniqueRoles yang memicu eror telah dihapus total dari file ini

func (r *sqlUserRepository) DeleteUser(ctx context.Context, id int64) error {
	return r.q.DeleteUser(ctx, id)
}

func (r *sqlUserRepository) GetRoleByName(ctx context.Context, name string) (domain.Role, error) {
	row, err := r.q.GetRoleByName(ctx, name)
	if err != nil {
		return domain.Role{}, err
	}

	return domain.Role{
		ID:   row.ID,
		Name: row.Name,
	}, nil
}

func (r *sqlUserRepository) UpdateUser(ctx context.Context, arg domain.UpdateUserParams) error {
	rows, err := r.q.UpdateUser(ctx, db.UpdateUserParams{
		ID:     arg.ID,
		Name:   arg.Name,
		Email:  arg.Email,
		RoleID: arg.RoleID,
		// Role target dibaca usecase sebelum wewenang diperiksa. Syarat ini
		// membuat UPDATE gagal bila role berubah di antara baca dan tulis.
		RoleID_2: arg.ExpectedRoleID,
	})
	if err != nil {
		// Dua admin bisa mengganti email ke nilai yang sama secara bersamaan.
		// Unique index tetap menjaga integritas; di sini hanya pesannya yang
		// diterjemahkan supaya tidak bocor sebagai 500.
		return mapDuplicate(err, "email sudah terdaftar")
	}
	if rows == 0 {
		return domain.ErrUserChanged
	}
	return nil
}
