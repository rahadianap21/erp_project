package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"go_backend/internal/domain"
	"go_backend/internal/middleware"
)

type VoucherHandler struct {
	uc domain.VoucherUsecase
}

func NewVoucherHandler(uc domain.VoucherUsecase) *VoucherHandler {
	return &VoucherHandler{uc: uc}
}

// VouchersCRUDHandler melayani master voucher.
func (h *VoucherHandler) VouchersCRUDHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		protected := middleware.RequirePermission("vouchers:read", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			q := r.URL.Query()
			if idStr := q.Get("id"); idStr != "" {
				id, err := strconv.ParseInt(idStr, 10, 64)
				if err != nil || id <= 0 {
					w.WriteHeader(http.StatusBadRequest)
					_ = json.NewEncoder(w).Encode(map[string]string{"message": "id voucher tidak valid"})
					return
				}
				v, err := h.uc.GetByID(r.Context(), id)
				if err != nil {
					WriteError(w, err)
					return
				}
				if v == nil {
					w.WriteHeader(http.StatusNotFound)
					_ = json.NewEncoder(w).Encode(map[string]string{"message": "voucher tidak ditemukan"})
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"data": v})
				return
			}
			limit, _ := strconv.Atoi(q.Get("limit"))
			cursor, _ := strconv.ParseInt(q.Get("cursor_id"), 10, 64)
			items, total, err := h.uc.GetAll(r.Context(), q.Get("search"), cursor, limit, q.Get("direction"))
			if err != nil {
				WriteError(w, err)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"data": items, "total": total})
		}))
		protected(w, r)
		return

	case http.MethodPost:
		protected := middleware.RequirePermission("vouchers:create", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var in domain.VoucherInput
			if !decodeJSONBody(w, r, &in) {
				return
			}
			uid := middleware.CurrentUserID(r)
			v, err := h.uc.Create(r.Context(), in, uid)
			if err != nil {
				WriteError(w, err)
				return
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "voucher berhasil dibuat", "data": v})
		}))
		protected(w, r)
		return

	case http.MethodPut:
		protected := middleware.RequirePermission("vouchers:update", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
			if err != nil || id <= 0 {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"message": "id voucher tidak valid"})
				return
			}
			var in domain.VoucherInput
			if !decodeJSONBody(w, r, &in) {
				return
			}
			uid := middleware.CurrentUserID(r)
			if err := h.uc.Update(r.Context(), id, in, uid); err != nil {
				WriteError(w, err)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "voucher berhasil diperbarui"})
		}))
		protected(w, r)
		return

	case http.MethodDelete:
		protected := middleware.RequirePermission("vouchers:delete", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
			if err != nil || id <= 0 {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"message": "id voucher tidak valid"})
				return
			}
			uid := middleware.CurrentUserID(r)
			if err := h.uc.Delete(r.Context(), id, uid); err != nil {
				WriteError(w, err)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "voucher berhasil dihapus"})
		}))
		protected(w, r)
		return

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
}

// HandlePOSLookupVoucher mencari voucher dari isi barcode untuk layar kasir.
// GET /api/pos/vouchers?code=<kode>
func (h *VoucherHandler) HandlePOSLookupVoucher(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	protected := middleware.RequirePermission("pos:read", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		v, err := h.uc.GetByCode(r.Context(), code)
		if err != nil {
			WriteError(w, err)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": v})
	}))
	protected(w, r)
}
