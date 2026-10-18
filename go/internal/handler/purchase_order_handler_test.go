package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go_backend/internal/domain"
	"go_backend/internal/middleware"
)

// stubPOUsecase hanya mengimplementasikan ApprovePO dan CancelPO; dua metode itu
// saja yang disentuh oleh test di bawah.
type stubPOUsecase struct {
	domain.PurchaseOrderUsecase

	approveErr error
	cancelErr  error
}

func (s *stubPOUsecase) ApprovePO(ctx context.Context, id int64, userID int64) error {
	return s.approveErr
}

func (s *stubPOUsecase) CancelPO(ctx context.Context, id int64, userID int64) error {
	return s.cancelErr
}

func wrapErr(err error) error {
	return fmt.Errorf("lapisan lain: %w", err)
}

// callPO mengirim request yang sudah membawa requestAuth, persis seperti yang
// diisi JWTMiddlewareWithResolver setelah membaca permission aktual dari
// database.
func callPO(t *testing.T, permission string, handler func(http.ResponseWriter, *http.Request)) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(http.MethodPut, "/api/purchase-orders/approve?id=77", nil)
	r = middleware.WithRequestAuth(r, middleware.RequestAuth{
		UserID:      9,
		Permissions: []string{permission},
	})
	w := httptest.NewRecorder()
	handler(w, r)
	return w
}

// TestKonflikStatusApproveHarus422 menjaga agar penolakan balapan status keluar
// sebagai 422, bukan 500. Penolakan ini memang sudah benar terjadi di usecase,
// jadi status 500 akan salah dan membingungkan.
func TestKonflikStatusApproveHarus422(t *testing.T) {
	uc := &stubPOUsecase{approveErr: &domain.TransitionStatusConflictError{
		From: "APPROVED", To: "APPROVED", Action: "disetujui",
	}}

	w := callPO(t, "po:approve", NewPurchaseOrderHandler(uc).HandleApprovePO)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("konflik status harus 422, dapat %d: %s", w.Code, w.Body.String())
	}
}

// TestKonflikStatusCancelHarus422 adalah regression untuk bug nyata: pesan
// konflik pembatalan diawali "dokumen tidak dapat dibatalkan..." yang tidak ada
// di daftar awalan kalimat, sehingga sebelumnya ikut terpetakan ke 500.
func TestKonflikStatusCancelHarus422(t *testing.T) {
	uc := &stubPOUsecase{cancelErr: &domain.TransitionStatusConflictError{
		From: "APPROVED", To: "REJECTED", Action: "dibatalkan",
	}}

	w := callPO(t, "po:reject", NewPurchaseOrderHandler(uc).HandleCancelPO)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("konflik status saat pembatalan harus 422, dapat %d: %s", w.Code, w.Body.String())
	}

	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("respons bukan JSON valid: %v", err)
	}
	if !strings.Contains(body["message"], "dibatalkan") {
		t.Fatalf("pesan harus menyebut aksi pembatalan, dapat %q", body["message"])
	}
}

// TestErrorPOInternalTetap500 memastikan pengenalan konflik lewat tipe error
// tidak membuat kegagalan server ikut tersamar jadi 422.
func TestErrorPOInternalTetap500(t *testing.T) {
	uc := &stubPOUsecase{approveErr: errors.New("koneksi database terputus")}

	w := callPO(t, "po:approve", NewPurchaseOrderHandler(uc).HandleApprovePO)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("error internal harus tetap 500, dapat %d", w.Code)
	}
}

// TestKonflikStatusDibungkusTetapTerdeteksi memastikan errors.As tetap bekerja
// walau error dibungkus dengan fmt.Errorf, karena lapisan lain bisa membungkus
// error sebelum sampai ke handler.
func TestKonflikStatusDibungkusTetapTerdeteksi(t *testing.T) {
	wrapped := wrapErr(&domain.TransitionStatusConflictError{
		From: "APPROVED", To: "REJECTED", Action: "dibatalkan",
	})
	uc2 := &stubPOUsecase{cancelErr: wrapped}

	w := callPO(t, "po:reject", NewPurchaseOrderHandler(uc2).HandleCancelPO)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("error konflik terbungkus harus tetap 422, dapat %d: %s", w.Code, w.Body.String())
	}
}
