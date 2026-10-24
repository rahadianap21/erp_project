package usecase

import (
	"context"
	"testing"

	"go_backend/internal/domain"
)

// saleRepoRetryPalsu mengembalikan snapshot harga yang berubah pada percobaan
// pertama dan berhasil pada percobaan kedua.
type saleRepoRetryPalsu struct {
	domain.SaleRepository
	getCalls    int
	createCalls int
}

func (r *saleRepoRetryPalsu) GetProductsForSale(_ context.Context, ids []int64) (map[int64]domain.SaleProductInfo, error) {
	r.getCalls++
	price := 20000.0
	if r.getCalls > 1 {
		price = 25000
	}
	out := make(map[int64]domain.SaleProductInfo, len(ids))
	for _, id := range ids {
		out[id] = domain.SaleProductInfo{
			ID:           id,
			SKU:          "SKU-UJI",
			Name:         "Produk Uji",
			SellingPrice: price,
			AvgCost:      10000,
			CurrentStock: 10,
			DiscountType: domain.SaleDiscountNone,
			IsTaxable:    false,
		}
	}
	return out, nil
}

func (r *saleRepoRetryPalsu) CreateSale(_ context.Context, sale *domain.Sale, _ *domain.SaleJournalPlan) (*domain.Sale, error) {
	r.createCalls++
	if r.createCalls == 1 {
		return nil, domain.ErrSaleSnapshotChanged
	}
	sale.ID = 1
	return sale, nil
}

type paymentMethodRepoPalsu struct {
	domain.PaymentMethodRepository
}

func (r *paymentMethodRepoPalsu) GetByID(_ context.Context, id int64) (*domain.PaymentMethod, error) {
	accountID := int64(99)
	return &domain.PaymentMethod{
		ID:        id,
		Name:      "Tunai",
		Channel:   domain.PaymentChannelCash,
		AccountID: &accountID,
	}, nil
}

func (r *paymentMethodRepoPalsu) GetByIDs(_ context.Context, ids []int64) ([]domain.PaymentMethod, error) {
	accountID := int64(99)
	methods := make([]domain.PaymentMethod, 0, len(ids))
	for _, id := range ids {
		methods = append(methods, domain.PaymentMethod{
			ID:        id,
			Name:      "Tunai",
			Channel:   domain.PaymentChannelCash,
			AccountID: &accountID,
		})
	}
	return methods, nil
}

func TestCreateSaleMenghitungUlangSaatSnapshotBerubah(t *testing.T) {
	repo := &saleRepoRetryPalsu{}
	uc := NewSaleUsecase(repo, nil, &paymentMethodRepoPalsu{}, nil, nil, nil)

	sale, err := uc.CreateSale(context.Background(), domain.CreateSaleInput{
		Items: []domain.SaleLineInput{{ProductID: 1, Qty: 1}},
		Payments: []domain.SalePaymentInput{
			{PaymentMethodID: 1, Amount: 25000},
		},
		CreatedBy: 7,
	})
	if err != nil {
		t.Fatalf("CreateSale gagal: %v", err)
	}
	if repo.getCalls != 2 || repo.createCalls != 2 {
		t.Fatalf("harus menghitung ulang sekali, dapat get=%d create=%d", repo.getCalls, repo.createCalls)
	}
	if sale.Details[0].UnitPrice != 25000 || sale.TotalAmount != 25000 {
		t.Fatalf("harus memakai harga snapshot terbaru 25000, dapat harga %.2f total %.2f", sale.Details[0].UnitPrice, sale.TotalAmount)
	}
}
