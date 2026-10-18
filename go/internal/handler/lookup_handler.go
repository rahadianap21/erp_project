package handler

import (
	"encoding/json"
	"go_backend/internal/domain"
	"go_backend/internal/middleware"
	"net/http"
	"strconv"
)

type LookupHandler struct {
	LookupUsecase domain.LookupUsecase
}

func NewLookupHandler(u domain.LookupUsecase) *LookupHandler {
	return &LookupHandler{LookupUsecase: u}
}

// HandleLookupSuppliers melayani pembacaan daftar supplier untuk form PO.
// Sengaja hanya dijaga 'po:read', bukan 'suppliers:read', supaya operator
// purchasing tetap bisa mengisi dropdown tanpa hak akses modul master.
func (h *LookupHandler) HandleLookupSuppliers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodGet {
		middleware.RequirePermission("po:read", func(w http.ResponseWriter, r *http.Request) {
			data, err := h.LookupUsecase.GetSuppliersForPO(r.Context())
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
		})(w, r)
		return
	}

	w.WriteHeader(http.StatusMethodNotAllowed)
}

// HandleLookupProductsFormPO menarik daftar barang, opsional disaring per
// supplier lewat query ?supplier_id=. Nilai 0 atau absen berarti seluruh katalog.
func (h *LookupHandler) HandleLookupProducts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodGet {
		middleware.RequirePermission("po:read", func(w http.ResponseWriter, r *http.Request) {
			supplierID := int64(0)
			if s := r.URL.Query().Get("supplier_id"); s != "" && s != "undefined" {
				if parsed, err := strconv.ParseInt(s, 10, 64); err == nil && parsed > 0 {
					supplierID = parsed
				}
			}

			data, err := h.LookupUsecase.GetProductsForPO(r.Context(), supplierID)
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
		})(w, r)
		return
	}

	w.WriteHeader(http.StatusMethodNotAllowed)
}
