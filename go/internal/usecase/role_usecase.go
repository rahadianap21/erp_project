package usecase

import (
	"context"
	"errors"
	"go_backend/internal/domain"
)

type roleUsecase struct {
	roleRepo domain.RoleRepository
}

// NewRoleUsecase menginisialisasi usecase khusus untuk manajemen data master peran
func NewRoleUsecase(repo domain.RoleRepository) domain.RoleUsecase {
	return &roleUsecase{roleRepo: repo}
}

// GetAllRolesMaster memproses pengambilan data peran murni terpaginasi dari repositori
func (u *roleUsecase) GetAllRolesMaster(ctx context.Context, search string, page, limit int) ([]domain.Role, int64, error) {
	// Hitung nilai offset untuk keperluan pagination database PostgreSQL
	offset := (page - 1) * limit

	// 1 KALI TEMBAK: Langsung mendapatkan data list, array permissions, dan total count sekaligus!
	roles, total, err := u.roleRepo.GetAllRolesMaster(ctx, search, offset, limit)
	if err != nil {
		return nil, 0, errors.New("gagal memuat data master peran dari database")
	}

	return roles, total, nil
}

// GetRolesCursor memproses pengambilan data peran dengan cursor pagination
func (u *roleUsecase) GetRolesCursor(ctx context.Context, params domain.RoleCursorParams) (*domain.RoleCursorResult, error) {
	result, err := u.roleRepo.GetRolesCursor(ctx, params)
	if err != nil {
		return nil, errors.New("gagal memuat data master peran dengan cursor: " + err.Error())
	}
	return result, nil
}

// GetAllMasterPermissions menarik seluruh daftar opsi hak akses yang terdaftar di sistem
func (u *roleUsecase) GetAllMasterPermissions(ctx context.Context) ([]string, error) {
	permissions, err := u.roleRepo.GetAllMasterPermissions(ctx)
	if err != nil {
		return nil, errors.New("gagal memuat daftar pilihan hak akses")
	}
	return permissions, nil
}

// CreateRoleWithPermissions memproses pembuatan peran baru sekaligus menyematkan hak akses granularnya
func (u *roleUsecase) CreateRoleWithPermissions(ctx context.Context, name string, permissions []string) error {
	if name == "" {
		return errors.New("nama peran tidak boleh kosong")
	}

	// Teruskan data ke layer repositori untuk dieksekusi dalam 1 kali transaksi (Tx) DB aman
	err := u.roleRepo.CreateRoleWithPermissions(ctx, name, permissions)
	if err != nil {
		return errors.New("gagal menyimpan peran baru: " + err.Error())
	}
	return nil
}

func (u *roleUsecase) UpdateRoleWithPermissions(ctx context.Context, id int32, name string, permissions []string) error {
	if name == "" {
		return errors.New("nama peran tidak boleh kosong")
	}

	err := u.roleRepo.UpdateRoleWithPermissions(ctx, id, name, permissions)
	if err != nil {
		return errors.New("gagal memperbarui data peran: " + err.Error())
	}
	return nil
}

func (u *roleUsecase) DeleteRoleWithPermissions(ctx context.Context, id int32) error {
	err := u.roleRepo.DeleteRoleWithPermissions(ctx, id)
	if err != nil {
		return errors.New("gagal menghapus peran: " + err.Error())
	}
	return nil
}
