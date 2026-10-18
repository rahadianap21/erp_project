package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"go_backend/internal/domain"
)

type MemberHandler struct {
	uc domain.MemberUsecase
}

func NewMemberHandler(uc domain.MemberUsecase) *MemberHandler {
	return &MemberHandler{uc: uc}
}

func writeMemberError(w http.ResponseWriter, err error) {
	if isUniqueViolation(err) {
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "kode member sudah dipakai"})
		return
	}
	w.WriteHeader(http.StatusUnprocessableEntity)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
}

// MembersCRUDHandler melayani master member.
func (h *MemberHandler) MembersCRUDHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		protected := RequirePermission("members:read", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			q := r.URL.Query()
			if idStr := q.Get("id"); idStr != "" {
				id, err := strconv.ParseInt(idStr, 10, 64)
				if err != nil || id <= 0 {
					w.WriteHeader(http.StatusBadRequest)
					_ = json.NewEncoder(w).Encode(map[string]string{"message": "id member tidak valid"})
					return
				}
				m, err := h.uc.GetByID(r.Context(), id)
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					_ = json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
					return
				}
				if m == nil {
					w.WriteHeader(http.StatusNotFound)
					_ = json.NewEncoder(w).Encode(map[string]string{"message": "member tidak ditemukan"})
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": m})
				return
			}
			limit, _ := strconv.Atoi(q.Get("limit"))
			cursor, _ := strconv.ParseInt(q.Get("cursor_id"), 10, 64)
			items, total, err := h.uc.GetAll(r.Context(), q.Get("search"), cursor, limit, q.Get("direction"))
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": items, "total": total})
		}))
		protected(w, r)
		return

	case http.MethodPost:
		protected := RequirePermission("members:create", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var in domain.MemberInput
			if !decodeJSONBody(w, r, &in) {
				return
			}
			uid := CurrentUserID(r)
			m, err := h.uc.Create(r.Context(), in, uid)
			if err != nil {
				writeMemberError(w, err)
				return
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"message": "member berhasil dibuat", "data": m})
		}))
		protected(w, r)
		return

	case http.MethodPut:
		protected := RequirePermission("members:update", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
			if err != nil || id <= 0 {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"message": "id member tidak valid"})
				return
			}
			var in domain.MemberInput
			if !decodeJSONBody(w, r, &in) {
				return
			}
			uid := CurrentUserID(r)
			if err := h.uc.Update(r.Context(), id, in, uid); err != nil {
				writeMemberError(w, err)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "member berhasil diperbarui"})
		}))
		protected(w, r)
		return

	case http.MethodDelete:
		protected := RequirePermission("members:delete", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
			if err != nil || id <= 0 {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"message": "id member tidak valid"})
				return
			}
			uid := CurrentUserID(r)
			if err := h.uc.Delete(r.Context(), id, uid); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "member berhasil dihapus"})
		}))
		protected(w, r)
		return

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
}

// MembersPointsHandler mengembalikan riwayat buku besar poin seorang member.
// GET /api/members/points?id=<member_id>&limit=<n>
func (h *MemberHandler) MembersPointsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	protected := RequirePermission("members:read", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		id, err := strconv.ParseInt(q.Get("id"), 10, 64)
		if err != nil || id <= 0 {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "id member tidak valid"})
			return
		}
		limit, _ := strconv.Atoi(q.Get("limit"))
		items, err := h.uc.ListPointEntries(r.Context(), id, limit)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": items})
	}))
	protected(w, r)
}
