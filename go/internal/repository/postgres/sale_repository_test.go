package postgres

import (
	"context"
	"errors"
	"testing"

	"go_backend/internal/domain"
)

// TestCreateSaleMenolakQtyTidakPositif menguji lapisan pertahanan repository.
// Domain sudah menolak qty <= 0, tetapi pemanggil lain bisa melewatinya; tanpa
// pemeriksaan ini qty negatif akan menambah stok lewat current_stock - (-qty).
func TestCreateSaleMenolakQtyTidakPositif(t *testing.T) {
	repo := NewSQLSaleRepository(nil)
	for _, qty := range []int{0, -1, -5} {
		_, err := repo.CreateSale(context.Background(), &domain.Sale{
			Details: []domain.SaleDetail{{ProductID: 1, Qty: qty}},
		}, nil)
		if !errors.Is(err, domain.ErrSaleLineInvalid) {
			t.Fatalf("qty %d harus ErrSaleLineInvalid, dapat %v", qty, err)
		}
	}
}

func TestCreateSaleMenolakTanpaBaris(t *testing.T) {
	repo := NewSQLSaleRepository(nil)
	if _, err := repo.CreateSale(context.Background(), &domain.Sale{}, nil); !errors.Is(err, domain.ErrSaleItemsRequired) {
		t.Fatalf("tanpa baris harus ErrSaleItemsRequired, dapat %v", err)
	}
}
