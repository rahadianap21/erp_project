package usecase

import (
	"context"
	"errors"
	"go_backend/internal/domain"
)

type supplierUsecase struct {
	supplierRepo domain.SupplierRepository
}

// NewSupplierUsecase menginisialisasi usecase untuk data master supplier
func NewSupplierUsecase(repo domain.SupplierRepository) domain.SupplierUsecase {
	return &supplierUsecase{supplierRepo: repo}
}

func (u *supplierUsecase) CreateSupplier(ctx context.Context, supplier *domain.Supplier) error {
	if supplier.Name == "" {
		return errors.New("nama supplier tidak boleh kosong")
	}
	if supplier.Email == "" {
		return errors.New("alamat email supplier tidak boleh kosong")
	}
	if supplier.VillageID == "" {
		return errors.New("wilayah kelurahan tempat supplier wajib ditentukan")
	}

	return u.supplierRepo.Create(ctx, supplier)
}

func (u *supplierUsecase) GetAllSuppliers(ctx context.Context, search, province string, cursorID int64, limit int, direction string) ([]domain.Supplier, int64, error) {
	if limit <= 0 {
		limit = 10 // Fallback limit standar jika kosong
	}
	return u.supplierRepo.GetSuppliersCursor(ctx, search, province, cursorID, limit, direction)
}

func (u *supplierUsecase) GetSupplierByID(ctx context.Context, id int64) (*domain.Supplier, error) {
	if id <= 0 {
		return nil, errors.New("id supplier tidak valid")
	}
	return u.supplierRepo.GetByID(ctx, id)
}

func (u *supplierUsecase) UpdateSupplier(ctx context.Context, supplier *domain.Supplier) error {
	if supplier.ID <= 0 {
		return errors.New("id supplier yang akan diperbarui tidak valid")
	}
	if supplier.Name == "" {
		return errors.New("nama supplier tidak boleh kosong")
	}

	return u.supplierRepo.Update(ctx, supplier)
}

func (u *supplierUsecase) DeleteSupplier(ctx context.Context, id, deletedBy int64) error {
	if id <= 0 {
		return errors.New("id supplier yang akan dihapus tidak valid")
	}
	return u.supplierRepo.Delete(ctx, id, deletedBy)
}
