package handler

import (
	"encoding/json"
	"go_backend/internal/domain"
	"net/http"
)

type DashboardHandler struct {
	DashboardUsecase domain.DashboardUsecase
}

func NewDashboardHandler(u domain.DashboardUsecase) *DashboardHandler {
	return &DashboardHandler{DashboardUsecase: u}
}

// HandleDashboard menyajikan agregat read-only untuk halaman dashboard.
// Dijaga permission dashboard:read supaya role yang tidak berhak (mis. kasir
// POS) tidak bisa membaca angka ringkasan ERP lewat API.
func (h *DashboardHandler) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "method tidak diizinkan"})
		return
	}

	data, err := h.DashboardUsecase.GetOverview(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": "success",
		"data":   data,
	})
}
