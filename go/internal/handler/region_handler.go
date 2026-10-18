package handler

import (
	"encoding/json"
	"go_backend/internal/domain"
	"net/http"
)

type RegionHandler struct {
	RegionUsecase domain.RegionUsecase
}

func NewRegionHandler(u domain.RegionUsecase) *RegionHandler {
	return &RegionHandler{RegionUsecase: u}
}

func (h *RegionHandler) HandleRegions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()
	provinceID := r.URL.Query().Get("province_id")
	regencyID := r.URL.Query().Get("regency_id")
	districtID := r.URL.Query().Get("district_id")

	// 🌟 CASCADING FILTER LOGIC BASED ON QUERY PARAMS
	if districtID != "" {
		data, err := h.RegionUsecase.GetVillages(ctx, districtID)
		h.respond(w, data, err)
		return
	}
	if regencyID != "" {
		data, err := h.RegionUsecase.GetDistricts(ctx, regencyID)
		h.respond(w, data, err)
		return
	}
	if provinceID != "" {
		data, err := h.RegionUsecase.GetRegencies(ctx, provinceID)
		h.respond(w, data, err)
		return
	}

	// Default: Jika tidak ada query param, kembalikan daftar semua Provinsi
	data, err := h.RegionUsecase.GetProvinces(ctx)
	h.respond(w, data, err)
}

func (h *RegionHandler) respond(w http.ResponseWriter, data interface{}, err error) {
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "data": data})
}
