package handler

import (
	"encoding/json"
	"net/http"

	"go_backend/internal/domain"
	"go_backend/internal/middleware"
)

type SettingsHandler struct {
	uc domain.SettingsUsecase
}

func NewSettingsHandler(uc domain.SettingsUsecase) *SettingsHandler {
	return &SettingsHandler{uc: uc}
}

// HandleSettings melayani GET /api/settings (settings:read) dan
// PUT /api/settings (settings:update).
func (h *SettingsHandler) HandleSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		protected := middleware.RequirePermission("settings:read", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			view, err := h.uc.GetAll(r.Context())
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"data": view})
		}))
		protected(w, r)
		return

	case http.MethodPut:
		protected := middleware.RequirePermission("settings:update", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var in domain.UpdateSettingsInput
			if !decodeJSONBody(w, r, &in) {
				return
			}
			uid := middleware.CurrentUserID(r)
			view, err := h.uc.Update(r.Context(), in, uid)
			if err != nil {
				WriteError(w, err)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"message": "pengaturan berhasil disimpan", "data": view})
		}))
		protected(w, r)
		return

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
