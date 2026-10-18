package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"go_backend/internal/domain"
	"go_backend/internal/middleware"
)

type GoodsReceiptNoteHandler struct {
	uc domain.GoodsReceiptNoteUsecase
}

func NewGoodsReceiptNoteHandler(uc domain.GoodsReceiptNoteUsecase) *GoodsReceiptNoteHandler {
	return &GoodsReceiptNoteHandler{uc: uc}
}

func (h *GoodsReceiptNoteHandler) HandleGRNs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodPost:
		protected := middleware.RequirePermission("grn:create", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var in domain.CreateGRNInput
			if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{"message": "body tidak valid: " + err.Error()})
				return
			}
			uid := middleware.CurrentUserID(r)
			if uid <= 0 {
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{"message": "tidak terautentikasi"})
				return
			}
			in.CreatedBy = uid
			grn, err := h.uc.CreateGRN(r.Context(), in)
			if err != nil {
				WriteError(w, err)
				return
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]any{"message": "grn berhasil dibuat", "data": grn})
		}))
		protected(w, r)
		return

	case http.MethodGet:
		protected := middleware.RequirePermission("grn:read", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			q := r.URL.Query()
			if idStr := q.Get("id"); idStr != "" {
				id, err := strconv.ParseInt(idStr, 10, 64)
				if err != nil || id <= 0 {
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(map[string]string{"message": "id grn tidak valid"})
					return
				}
				g, err := h.uc.GetByID(r.Context(), id)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
					return
				}
				if g == nil {
					w.WriteHeader(http.StatusNotFound)
					json.NewEncoder(w).Encode(map[string]string{"message": "grn tidak ditemukan"})
					return
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]any{"data": g})
				return
			}
			search := strings.TrimSpace(q.Get("search"))
			limit, _ := strconv.Atoi(q.Get("limit"))
			cursor, _ := strconv.ParseInt(q.Get("cursor_id"), 10, 64)
			dir := q.Get("direction")
			if dir != "prev" {
				dir = "next"
			}
			items, total, err := h.uc.GetAll(r.Context(), search, cursor, limit, dir)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
				return
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"data": items, "total": total})
		}))
		protected(w, r)
		return

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
}

func (h *GoodsReceiptNoteHandler) HandleFinishReceiving(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	protected := middleware.RequirePermission("po:receive", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		id, err := strconv.ParseInt(q.Get("id"), 10, 64)
		if err != nil || id <= 0 {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"message": "id po tidak valid"})
			return
		}
		allowPartial := strings.ToLower(q.Get("allow_partial")) == "true" || q.Get("allow_partial") == "1"
		uid := middleware.CurrentUserID(r)
		if uid <= 0 {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"message": "tidak terautentikasi"})
			return
		}
		if err := h.uc.FinishReceiving(r.Context(), id, allowPartial, uid); err != nil {
			WriteError(w, err)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "po berhasil diterima"})
	}))
	protected(w, r)
}
