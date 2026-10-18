package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"go_backend/internal/domain"
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
		protected := RequirePermission("settings:read", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			view, err := h.uc.GetAll(r.Context())
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": view})
		}))
		protected(w, r)
		return

	case http.MethodPut:
		protected := RequirePermission("settings:update", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var in domain.UpdateSettingsInput
			if !decodeJSONBody(w, r, &in) {
				return
			}
			uid := CurrentUserID(r)
			view, err := h.uc.Update(r.Context(), in, uid)
			if err != nil {
				status := http.StatusInternalServerError
				if errors.Is(err, domain.ErrSettingPPNRateInvalid) || errors.Is(err, domain.ErrSettingPPNModeInvalid) {
					status = http.StatusUnprocessableEntity
				}
				w.WriteHeader(status)
				_ = json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"message": "pengaturan berhasil disimpan", "data": view})
		}))
		protected(w, r)
		return

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
