package usecase

import (
	"context"
	"testing"

	"go_backend/internal/domain"
)

// productRepoPalsu merekam produk terakhir yang disimpan, cukup untuk menguji
// normalisasi conversion_rate tanpa database.
type productRepoPalsu struct {
	disimpan domain.Product
}

func (r *productRepoPalsu) Create(_ context.Context, p *domain.Product) error {
	r.disimpan = *p
	return nil
}

func (r *productRepoPalsu) GetProductsCursor(context.Context, string, int32, int64, int, string) ([]domain.Product, int64, error) {
	return nil, 0, nil
}

func (r *productRepoPalsu) GetByID(context.Context, int64) (*domain.Product, error) {
	return nil, nil
}

func (r *productRepoPalsu) GetByIDs(context.Context, []int64) ([]domain.Product, error) {
	return nil, nil
}

func (r *productRepoPalsu) Update(_ context.Context, p *domain.Product) error {
	r.disimpan = *p
	return nil
}

func (r *productRepoPalsu) Delete(context.Context, int64, int64) error {
	return nil
}

// TestUpdateProductMemaksaRateMinimalSatu menjaga CHECK constraint
// m_products_conversion_rate_check: klien lama yang mengirim 0 tidak boleh
// menghasilkan error database.
func TestUpdateProductMemaksaRateMinimalSatu(t *testing.T) {
	repo := &productRepoPalsu{}
	uc := NewProductUsecase(repo)

	p := &domain.Product{
		ID: 1, Name: "Beras", SellingUnitID: 6, PurchaseUnitID: 3,
		ConversionRate: 0, PurchasePrice: 10, SellingPrice: 20,
	}
	if err := uc.UpdateProduct(context.Background(), p); err != nil {
		t.Fatalf("update tidak boleh gagal: %v", err)
	}
	if repo.disimpan.ConversionRate != 1 {
		t.Fatalf("conversion_rate 0 harus dinormalkan ke 1, dapat %d", repo.disimpan.ConversionRate)
	}
}

// TestUpdateProductMemaksaRateSatuSaatSatuanSama memastikan aturan "satuan beli
// == satuan jual berarti faktor 1" juga berlaku pada jalur update, bukan cuma
// create.
func TestUpdateProductMemaksaRateSatuSaatSatuanSama(t *testing.T) {
	repo := &productRepoPalsu{}
	uc := NewProductUsecase(repo)

	p := &domain.Product{
		ID: 1, Name: "Beras", SellingUnitID: 6, PurchaseUnitID: 6,
		ConversionRate: 12, PurchasePrice: 10, SellingPrice: 20,
	}
	if err := uc.UpdateProduct(context.Background(), p); err != nil {
		t.Fatalf("update tidak boleh gagal: %v", err)
	}
	if repo.disimpan.ConversionRate != 1 {
		t.Fatalf("satuan sama harus memaksa rate 1, dapat %d", repo.disimpan.ConversionRate)
	}
}

func TestUpdateProductMempertahankanRateValid(t *testing.T) {
	repo := &productRepoPalsu{}
	uc := NewProductUsecase(repo)

	p := &domain.Product{
		ID: 1, Name: "Beras", SellingUnitID: 6, PurchaseUnitID: 3,
		ConversionRate: 12, PurchasePrice: 10, SellingPrice: 20,
	}
	if err := uc.UpdateProduct(context.Background(), p); err != nil {
		t.Fatalf("update tidak boleh gagal: %v", err)
	}
	if repo.disimpan.ConversionRate != 12 {
		t.Fatalf("rate valid harus dipertahankan 12, dapat %d", repo.disimpan.ConversionRate)
	}
}
