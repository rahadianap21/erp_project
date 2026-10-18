package postgres

import (
	"context"
	"encoding/json"
	"go_backend/internal/domain"
	"go_backend/internal/repository/postgres/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

type sqlRoleRepository struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

func NewSQLRoleRepository(pool *pgxpool.Pool) domain.RoleRepository {
	return &sqlRoleRepository{pool: pool, q: db.New(pool)}
}

func (r *sqlRoleRepository) GetAllRolesMaster(ctx context.Context, search string, offset, limit int) ([]domain.Role, int64, error) {
	rows, err := r.q.GetAllRolesMaster(ctx, db.GetAllRolesMasterParams{
		SearchQuery: search,
		ReqLimit:    int32(limit),
		ReqOffset:   int32(offset),
	})
	if err != nil {
		return nil, 0, err
	}

	var roles []domain.Role
	var totalCount int64 = 0

	// Jika database mengembalikan data, ambil angka total_count dari baris pertama
	if len(rows) > 0 {
		totalCount = rows[0].TotalCount
	}

	for _, row := range rows {
		// Konversi data jsonb dari PostgreSQL menjadi slice string []string di Go
		var perms []string
		if len(row.PermissionsJson) > 0 {
			_ = json.Unmarshal(row.PermissionsJson, &perms)
		}

		roles = append(roles, domain.Role{
			ID:          row.ID,
			Name:        row.Name,
			Permissions: perms, // 🌟 Permissions langsung terisi otomatis!
		})
	}

	return roles, totalCount, nil
}

func (r *sqlRoleRepository) GetRolesCursor(ctx context.Context, params domain.RoleCursorParams) (*domain.RoleCursorResult, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = 25
	}
	fetchLimit := int32(limit + 1)

	var roles []domain.Role
	var hasMore bool

	if params.Direction == "prev" && params.CursorID > 0 {
		rows, err := r.q.GetRolesCursorPrev(ctx, db.GetRolesCursorPrevParams{
			SearchQuery: params.Search,
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

		for i := len(rows) - 1; i >= 0; i-- {
			var perms []string
			if len(rows[i].PermissionsJson) > 0 {
				_ = json.Unmarshal(rows[i].PermissionsJson, &perms)
			}
			roles = append(roles, domain.Role{
				ID:          rows[i].ID,
				Name:        rows[i].Name,
				Permissions: perms,
			})
		}
	} else {
		rows, err := r.q.GetRolesCursorNext(ctx, db.GetRolesCursorNextParams{
			SearchQuery: params.Search,
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
			var perms []string
			if len(row.PermissionsJson) > 0 {
				_ = json.Unmarshal(row.PermissionsJson, &perms)
			}
			roles = append(roles, domain.Role{
				ID:          row.ID,
				Name:        row.Name,
				Permissions: perms,
			})
		}
	}

	totalCount, _ := r.q.CountRoles(ctx, params.Search)

	result := &domain.RoleCursorResult{
		Roles: roles,
		Total: totalCount,
	}

	if len(roles) > 0 {
		firstID := roles[0].ID
		lastID := roles[len(roles)-1].ID

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

func (r *sqlRoleRepository) GetAllMasterPermissions(ctx context.Context) ([]string, error) {
	return r.q.GetAllMasterPermissions(ctx)
}

// CreateRoleWithPermissions membuat role lalu mengaitkan seluruh permission-nya
// dalam SATU transaksi.
//
// Tanpa transaksi, role sudah committed sebelum permission pertama masuk, sehingga
// kegagalan di tengah menyisakan role dengan permission separuh — user yang memakai
// role itu bisa terkunci di luar sistem atau, lebih buruk, terlihat punya akses
// parsial yang tidak pernah disetujui.
func (r *sqlRoleRepository) CreateRoleWithPermissions(ctx context.Context, name string, permissions []string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := r.q.WithTx(tx)

	role, err := qtx.CreateRoleWithReturn(ctx, name)
	if err != nil {
		return err
	}

	for _, perm := range permissions {
		err = qtx.AssignPermissionToRole(ctx, db.AssignPermissionToRoleParams{
			RoleID: role.ID,
			Name:   perm,
		})
		if err != nil {
			// Kembalikan error; defer Rollback membuat role yang sudah
			// tercipta di dalam transaksi ini hilangtotal.
			return err
		}
	}

	return tx.Commit(ctx)
}

// UpdateRoleWithPermissions mengganti nama dan SELURUH daftar permission role
// dalam satu transaksi.
//
// Ini pernah menjadi transaksi terburuk di sistem: permission lama dihapus dan
// di-commit duluan, baru permission baru di-insert. Kalau insert gagal, role
// kehilangan semua permission dan semua user yang memakainya langsung terkunci
// keluar dari aplikasi.
func (r *sqlRoleRepository) UpdateRoleWithPermissions(ctx context.Context, id int32, name string, permissions []string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := r.q.WithTx(tx)

	// 1. Perbarui nama peran.
	err = qtx.UpdateRoleName(ctx, db.UpdateRoleNameParams{
		Name: name,
		ID:   id,
	})
	if err != nil {
		return err
	}

	// 2. Bersihkan seluruh permissions lama milik peran ini.
	err = qtx.DeleteAllPermissionsFromRole(ctx, id)
	if err != nil {
		return err
	}

	// 3. Suntikkan kembali permissions baru hasil centangan form user.
	for _, perm := range permissions {
		err = qtx.AssignPermissionToRole(ctx, db.AssignPermissionToRoleParams{
			RoleID: id,
			Name:   perm,
		})
		if err != nil {
			// Rollback mengembalikan permission lama persis seperti semula.
			return err
		}
	}

	return tx.Commit(ctx)
}

// DeleteRoleWithPermissions menghapus relasi permission lalu role-nya dalam satu
// transaksi, supaya tidak pernah ada role tanpa permission atau role yatim.
func (r *sqlRoleRepository) DeleteRoleWithPermissions(ctx context.Context, id int32) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	qtx := r.q.WithTx(tx)

	// 1. Hapus semua relasi permissions terlebih dahulu agar tidak memicu error foreign key constraint.
	err = qtx.DeleteAllPermissionsFromRole(ctx, id)
	if err != nil {
		return err
	}

	// 2. Hapus data induk peran dari tabel s_roles.
	err = qtx.DeleteRoleOnly(ctx, id)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
