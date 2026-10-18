package usecase

import (
	"context"
	"go_backend/internal/domain"
)

type lookupUsecase struct {
	lookupRepo domain.LookupRepository
}

func NewLookupUsecase(repo domain.LookupRepository) domain.LookupUsecase {
	return &lookupUsecase{lookupRepo: repo}
}

func (u *lookupUsecase) GetSuppliersForPO(ctx context.Context) ([]domain.SupplierLookup, error) {
	return u.lookupRepo.GetSuppliersForPO(ctx)
}

// GetProductsForPO meneruskan supplierID apa adanya ke query. Nilai 0 berarti
// tarik seluruh katalog barang, dipakai form saat belum ada supplier terpilih.
func (u *lookupUsecase) GetProductsForPO(ctx context.Context, supplierID int64) ([]domain.ProductLookup, error) {
	if supplierID < 0 {
		supplierID = 0
	}
	return u.lookupRepo.GetProductsForPO(ctx, supplierID)
}
