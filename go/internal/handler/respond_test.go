package handler

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"go_backend/internal/domain"
)

// TestStatusForError memastikan seluruh kategori error domain terpetakan ke
// status HTTP yang benar lewat SATU fungsi, bukan lima cara berbeda
// seperti sebelumnya.
func TestStatusForError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"authorization", domain.NewAuthorizationError("tidak berwewenang"), http.StatusForbidden},
		{"business rule", domain.NewBusinessRuleError("pesanan ditolak: stok kurang"), http.StatusUnprocessableEntity},
		{"not found typed", domain.NewNotFoundError("dokumen po tidak ditemukan"), http.StatusNotFound},
		{"duplicate", domain.NewDuplicateError("kode member sudah dipakai"), http.StatusConflict},
		{"konflik transisi status", &domain.TransitionStatusConflictError{From: "APPROVED", To: "APPROVED", Action: "disetujui"}, http.StatusUnprocessableEntity},
		{"jurnal tidak dapat diposting", &domain.JournalNotPostableError{AccountID: 1, Code: "1101"}, http.StatusUnprocessableEntity},
		{"sentinel sale not found", domain.ErrSaleNotFound, http.StatusNotFound},
		{"sentinel period not found", domain.ErrPeriodNotFound, http.StatusNotFound},
		{"sentinel sale sudah void", domain.ErrSaleAlreadyVoid, http.StatusConflict},
		{"sentinel snapshot penjualan berubah", domain.ErrSaleSnapshotChanged, http.StatusConflict},
		{"sentinel daftar grn berubah", domain.ErrGRNReceiptChanged, http.StatusConflict},
		{"sentinel nomor po bentrok", domain.ErrNomorPOBentrok, http.StatusConflict},
		{"sentinel stok tidak cukup", domain.ErrSaleStokTidakCukup, http.StatusUnprocessableEntity},
		{"sentinel grn ditolak", domain.ErrGRNOverReceipt, http.StatusUnprocessableEntity},
		{"sentinel setting ppn", domain.ErrSettingPPNRateInvalid, http.StatusUnprocessableEntity},
		{"error infrastruktur", errors.New("koneksi database terputus"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := StatusForError(tc.err); got != tc.want {
				t.Fatalf("StatusForError(%v) = %d, want %d", tc.err, got, tc.want)
			}
		})
	}
}

// TestStatusForErrorMenerjangErrorBungkus memastikan errors.As/errors.Is tetap
// bekerja walau error dibungkus fmt.Errorf di lapisan usecase/repository.
func TestStatusForErrorMenerjangErrorBungkus(t *testing.T) {
	wrappedRule := fmt.Errorf("lapisan lain: %w", domain.NewBusinessRuleError("koreksi ditolak: sudah diterima"))
	if got := StatusForError(wrappedRule); got != http.StatusUnprocessableEntity {
		t.Fatalf("business rule terbungkus harus 422, dapat %d", got)
	}
	wrappedDup := fmt.Errorf("lapisan lain: %w", domain.NewDuplicateError("kode member sudah dipakai"))
	if got := StatusForError(wrappedDup); got != http.StatusConflict {
		t.Fatalf("duplicate terbungkus harus 409, dapat %d", got)
	}
}
