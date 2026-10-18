package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"go_backend/internal/domain"
)

// stubGRNUsecase merekam input CreateGRN supaya test bisa memeriksa hasil decode.
type stubGRNUsecase struct {
	domain.GoodsReceiptNoteUsecase
	masuk domain.CreateGRNInput
}

func (s *stubGRNUsecase) CreateGRN(_ context.Context, in domain.CreateGRNInput) (*domain.GoodsReceiptNote, error) {
	s.masuk = in
	return &domain.GoodsReceiptNote{ID: 1, GRNNumber: "GRN/TEST/0001"}, nil
}

// TestCreateGRNMenerimaTanggalDateOnly adalah regression untuk bug "body tidak
// valid": frontend mengirim received_date "2026-10-04" dari <input type="date">,
// sementara backend dulu memakai *time.Time yang hanya menerima RFC3339.
func TestCreateGRNMenerimaTanggalDateOnly(t *testing.T) {
	uc := &stubGRNUsecase{}
	body := `{
		"purchase_order_id": 7,
		"received_date": "2026-10-04",
		"notes": "diterima gudang",
		"items": [{"purchase_order_detail_id": 3, "qty_received": 12}]
	}`

	r := httptest.NewRequest(http.MethodPost, "/api/goods-receipt-notes", strings.NewReader(body))
	r = r.WithContext(context.WithValue(r.Context(), authKey, requestAuth{
		UserID:      9,
		Permissions: []string{"grn:create"},
	}))
	// CurrentUserID membaca klaim JWT di UserContextKey, bukan requestAuth.
	r = r.WithContext(context.WithValue(r.Context(), UserContextKey, jwt.MapClaims{"user_id": float64(9)}))
	w := httptest.NewRecorder()
	NewGoodsReceiptNoteHandler(uc).HandleGRNs(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("create GRN harus 201, dapat %d: %s", w.Code, w.Body.String())
	}
	if uc.masuk.ReceivedDate == nil {
		t.Fatal("received_date harus ter-decode, bukan nil")
	}
	if got := uc.masuk.ReceivedDate.Format("2006-01-02"); got != "2026-10-04" {
		t.Fatalf("tanggal harus 2026-10-04, dapat %s", got)
	}
	if uc.masuk.PurchaseOrderID != 7 || len(uc.masuk.Items) != 1 {
		t.Fatalf("isi input tidak ter-decode benar: %+v", uc.masuk)
	}
}
