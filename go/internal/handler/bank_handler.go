package handler

import (
	"encoding/json"
	"net/http"

	"go_backend/internal/domain"
)

type BankHandler struct {
	BankUsecase domain.BankUsecase
}

// NewBankHandler menginisialisasi handler untuk modul master bank
func NewBankHandler(u domain.BankUsecase) *BankHandler {
	return &BankHandler{BankUsecase: u}
}

// HandleGetBanks melayani request GET untuk menarik seluruh data opsi pilihan bank aktif
func (h *BankHandler) HandleGetBanks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Validasi pembatasan HTTP Method kaku
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Metode HTTP tidak diizinkan"})
		return
	}

	ctx := r.Context()

	// Tarik data melalui layer bisnis usecase
	banks, err := h.BankUsecase.GetAllBanks(ctx)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Gagal memuat data master bank: " + err.Error()})
		return
	}

	// Kirim data dengan format bungkus standar ERP Axion
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": "success",
		"data":   banks,
	})
}
