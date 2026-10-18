package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"go_backend/internal/domain"
	"go_backend/internal/middleware"
)

type SaleHandler struct {
	uc    domain.SaleUsecase
	posUC domain.POSUsecase
}

func NewSaleHandler(uc domain.SaleUsecase, posUC domain.POSUsecase) *SaleHandler {
	return &SaleHandler{uc: uc, posUC: posUC}
}

// HandleSales melayani daftar/detail (GET) dan pembuatan (POST) penjualan.
func (h *SaleHandler) HandleSales(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodPost:
		protected := middleware.RequirePermission("sales:create", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var in domain.CreateSaleInput
			if !decodeJSONBody(w, r, &in) {
				return
			}
			uid := middleware.CurrentUserID(r)
			if uid <= 0 {
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{"message": "tidak terautentikasi"})
				return
			}
			in.CreatedBy = uid
			sale, err := h.uc.CreateSale(r.Context(), in)
			if err != nil {
				WriteError(w, err)
				return
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "penjualan berhasil dibuat", "data": sale})
		}))
		protected(w, r)
		return

	case http.MethodGet:
		protected := middleware.RequirePermission("sales:read", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			q := r.URL.Query()
			if idStr := q.Get("id"); idStr != "" {
				id, err := strconv.ParseInt(idStr, 10, 64)
				if err != nil || id <= 0 {
					w.WriteHeader(http.StatusBadRequest)
					_ = json.NewEncoder(w).Encode(map[string]string{"message": "id penjualan tidak valid"})
					return
				}
				sale, err := h.uc.GetByID(r.Context(), id)
				if err != nil {
					WriteError(w, err)
					return
				}
				if sale == nil {
					w.WriteHeader(http.StatusNotFound)
					_ = json.NewEncoder(w).Encode(map[string]string{"message": "penjualan tidak ditemukan"})
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"data": sale})
				return
			}
			limit, _ := strconv.Atoi(q.Get("limit"))
			cursor, _ := strconv.ParseInt(q.Get("cursor_id"), 10, 64)
			items, total, err := h.uc.GetAll(r.Context(), domain.SaleListParam{
				Search:       q.Get("search"),
				StatusFilter: q.Get("status"),
				CursorID:     cursor,
				Limit:        limit,
				Direction:    q.Get("direction"),
			})
			if err != nil {
				WriteError(w, err)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"data": items, "total": total})
		}))
		protected(w, r)
		return

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
}

// HandleSaleVoid membatalkan penjualan POSTED (POST /api/sales/void?id=).
func (h *SaleHandler) HandleSaleVoid(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	protected := middleware.RequirePermission("sales:void", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
		if err != nil || id <= 0 {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "id penjualan tidak valid"})
			return
		}
		uid := middleware.CurrentUserID(r)
		if uid <= 0 {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "tidak terautentikasi"})
			return
		}
		var body struct {
			Reason string `json:"reason"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if err := h.uc.VoidSale(r.Context(), id, uid, body.Reason); err != nil {
			WriteError(w, err)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "penjualan berhasil dibatalkan"})
	}))
	protected(w, r)
}

// HandlePOSBootstrap menyuplai data awal layar kasir (metode bayar, kategori, tarif PPN).
func (h *SaleHandler) HandlePOSBootstrap(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	protected := middleware.RequirePermission("pos:read", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := h.posUC.Bootstrap(r.Context())
		if err != nil {
			WriteError(w, err)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
	}))
	protected(w, r)
}

// HandlePOSProducts melayani pencarian produk untuk grid kasir.
func (h *SaleHandler) HandlePOSProducts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	protected := middleware.RequirePermission("pos:read", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		limit, _ := strconv.Atoi(q.Get("limit"))
		category, _ := strconv.Atoi(q.Get("category_id"))
		items, err := h.posUC.SearchProducts(r.Context(), q.Get("search"), category, limit)
		if err != nil {
			WriteError(w, err)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": items})
	}))
	protected(w, r)
}

// HandlePOSMembers melayani pencarian member dari layar kasir.
func (h *SaleHandler) HandlePOSMembers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	protected := middleware.RequirePermission("pos:read", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		limit, _ := strconv.Atoi(q.Get("limit"))
		items, err := h.posUC.SearchMembers(r.Context(), q.Get("search"), limit)
		if err != nil {
			WriteError(w, err)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": items})
	}))
	protected(w, r)
}
