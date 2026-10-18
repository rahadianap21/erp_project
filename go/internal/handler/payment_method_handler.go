package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"go_backend/internal/domain"
	"go_backend/internal/middleware"
)

type PaymentMethodHandler struct {
	uc domain.PaymentMethodUsecase
}

func NewPaymentMethodHandler(uc domain.PaymentMethodUsecase) *PaymentMethodHandler {
	return &PaymentMethodHandler{uc: uc}
}

// PaymentMethodsCRUDHandler melayani master metode pembayaran.
func (h *PaymentMethodHandler) PaymentMethodsCRUDHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		protected := middleware.RequirePermission("payment-methods:read", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if idStr := r.URL.Query().Get("id"); idStr != "" {
				id, err := strconv.ParseInt(idStr, 10, 64)
				if err != nil || id <= 0 {
					w.WriteHeader(http.StatusBadRequest)
					_ = json.NewEncoder(w).Encode(map[string]string{"message": "id metode pembayaran tidak valid"})
					return
				}
				pm, err := h.uc.GetByID(r.Context(), id)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					_ = json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
					return
				}
				if pm == nil {
					w.WriteHeader(http.StatusNotFound)
					_ = json.NewEncoder(w).Encode(map[string]string{"message": "metode pembayaran tidak ditemukan"})
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": pm})
				return
			}
			items, err := h.uc.ListAll(r.Context())
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": items})
		}))
		protected(w, r)
		return

	case http.MethodPost:
		protected := middleware.RequirePermission("payment-methods:create", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var in domain.PaymentMethodInput
			if !decodeJSONBody(w, r, &in) {
				return
			}
			uid := middleware.CurrentUserID(r)
			pm, err := h.uc.Create(r.Context(), in, uid)
			if err != nil {
				WriteError(w, err)
				return
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"message": "metode pembayaran berhasil dibuat", "data": pm})
		}))
		protected(w, r)
		return

	case http.MethodPut:
		protected := middleware.RequirePermission("payment-methods:update", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
			if err != nil || id <= 0 {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"message": "id metode pembayaran tidak valid"})
				return
			}
			var in domain.PaymentMethodInput
			if !decodeJSONBody(w, r, &in) {
				return
			}
			uid := middleware.CurrentUserID(r)
			if err := h.uc.Update(r.Context(), id, in, uid); err != nil {
				WriteError(w, err)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "metode pembayaran berhasil diperbarui"})
		}))
		protected(w, r)
		return

	case http.MethodDelete:
		protected := middleware.RequirePermission("payment-methods:delete", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
			if err != nil || id <= 0 {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"message": "id metode pembayaran tidak valid"})
				return
			}
			uid := middleware.CurrentUserID(r)
			if err := h.uc.Delete(r.Context(), id, uid); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "metode pembayaran berhasil dihapus"})
		}))
		protected(w, r)
		return

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
}
