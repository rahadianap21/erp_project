package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go_backend/internal/domain"
	"go_backend/internal/middleware"
)

// stubProductUsecase hanya merekam produk yang diteruskan handler; cukup untuk
// memastikan pemetaan field request tidak diam-diam jatuh ke nilai nol.
type stubProductUsecase struct {
	domain.ProductUsecase
	created *domain.Product
	updated *domain.Product
}

func (s *stubProductUsecase) CreateProduct(_ context.Context, p *domain.Product) error {
	cp := *p
	s.created = &cp
	return nil
}

func (s *stubProductUsecase) UpdateProduct(_ context.Context, p *domain.Product) error {
	cp := *p
	s.updated = &cp
	return nil
}

func callProduct(t *testing.T, method, permission, body string) (*httptest.ResponseRecorder, *stubProductUsecase) {
	t.Helper()
	uc := &stubProductUsecase{}
	r := httptest.NewRequest(method, "/api/products?id=5", strings.NewReader(body))
	r = middleware.WithRequestAuth(r, middleware.RequestAuth{
		UserID:      9,
		Permissions: []string{permission},
	})
	w := httptest.NewRecorder()
	NewProductHandler(uc).ProductsCRUDHandler(w, r)
	return w, uc
}

const productBodyDenganRate12 = `{
	"name": "Beras Premium",
	"category_id": 1,
	"supplier_id": 1,
	"selling_unit_id": 6,
	"purchase_unit_id": 3,
	"conversion_rate": 12,
	"purchase_price": 100,
	"selling_price": 150,
	"min_stock_level": 0,
	"is_active": true
}`

// TestUpdateProductHandlerMeneruskanConversionRate adalah regression untuk bug
// nyata: struct request PUT tidak punya field conversion_rate, sehingga nilainya
// selalu ter-decode 0 lalu melanggar CHECK m_products_conversion_rate_check.
func TestUpdateProductHandlerMeneruskanConversionRate(t *testing.T) {
	w, uc := callProduct(t, http.MethodPut, "products:update", productBodyDenganRate12)

	if w.Code != http.StatusOK {
		t.Fatalf("update harus 200, dapat %d: %s", w.Code, w.Body.String())
	}
	if uc.updated == nil {
		t.Fatal("UpdateProduct tidak dipanggil")
	}
	if uc.updated.ConversionRate != 12 {
		t.Fatalf("conversion_rate harus diteruskan 12, dapat %d", uc.updated.ConversionRate)
	}
}

func TestCreateProductHandlerMeneruskanConversionRate(t *testing.T) {
	w, uc := callProduct(t, http.MethodPost, "products:create", productBodyDenganRate12)

	if w.Code != http.StatusCreated {
		t.Fatalf("create harus 201, dapat %d: %s", w.Code, w.Body.String())
	}
	if uc.created == nil {
		t.Fatal("CreateProduct tidak dipanggil")
	}
	if uc.created.ConversionRate != 12 {
		t.Fatalf("conversion_rate harus diteruskan 12, dapat %d", uc.created.ConversionRate)
	}
}
