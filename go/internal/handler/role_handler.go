package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"go_backend/internal/domain"
	"go_backend/internal/middleware"
)

type RoleHandler struct {
	RoleUsecase domain.RoleUsecase
}

// NewRoleHandler menginisialisasi handler khusus untuk modul peran
func NewRoleHandler(u domain.RoleUsecase) *RoleHandler {
	return &RoleHandler{RoleUsecase: u}
}

// RolesCRUDHandler memilah request HTTP Method secara terpusat untuk modul master peran
func (h *RoleHandler) RolesCRUDHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 🌟 1. JALUR READ DATA PERAN (METHOD GET)
	if r.Method == http.MethodGet {
		middleware.RequirePermission("roles:read", func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Tangkap parameter query dari URL
			search := r.URL.Query().Get("search")
			limit := 25
			if lStr := r.URL.Query().Get("limit"); lStr != "" {
				limit, _ = strconv.Atoi(lStr)
				if limit < 1 {
					limit = 25
				}
			}

			cursorStr := r.URL.Query().Get("cursor")
			pageStr := r.URL.Query().Get("page")
			page, _ := strconv.Atoi(pageStr)

			if cursorStr != "" || page <= 1 {
				direction := r.URL.Query().Get("direction")
				if direction != "prev" {
					direction = "next"
				}

				var cursorID int32 = 0
				if cursorStr != "" {
					c64, _ := strconv.ParseInt(cursorStr, 10, 32)
					cursorID = int32(c64)
				}

				result, err := h.RoleUsecase.GetRolesCursor(ctx, domain.RoleCursorParams{
					Search:    search,
					CursorID:  cursorID,
					Direction: direction,
					Limit:     limit,
				})
				if err != nil {
					w.WriteHeader(http.StatusInternalServerError)
					_ = json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
					return
				}

				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"status":      "success",
					"data":        result.Roles,
					"total":       result.Total,
					"next_cursor": result.NextCursor,
					"prev_cursor": result.PrevCursor,
					"has_next":    result.HasNext,
					"has_prev":    result.HasPrev,
				})
				return
			}

			// Fallback legacy offset pagination jika page > 1
			roles, total, err := h.RoleUsecase.GetAllRolesMaster(ctx, search, page, limit)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
				return
			}

			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "success",
				"data":   roles,
				"total":  total,
			})
		})(w, r)
		return
	}

	// 🌟 2. JALUR CREATE DATA PERAN (METHOD POST)
	if r.Method == http.MethodPost {
		middleware.RequirePermission("roles:create", func(w http.ResponseWriter, r *http.Request) {
			var req struct {
				Name        string   `json:"name"`
				Permissions []string `json:"permissions"`
			}

			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"message": "Payload request tidak valid"})
				return
			}

			err := h.RoleUsecase.CreateRoleWithPermissions(r.Context(), req.Name, req.Permissions)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
				return
			}

			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Peran baru berhasil dibuat"})
		})(w, r)
		return
	}

	// Untuk rute PUT dan DELETE, validasi parameter query ID (?id=...) wajib dipenuhi terlebih dahulu
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Parameter ID peran wajib disertakan"})
		return
	}

	idStrConverted, _ := strconv.Atoi(idStr)
	roleID := int32(idStrConverted)

	// 🌟 3. JALUR UPDATE DATA PERAN (METHOD PUT)
	if r.Method == http.MethodPut {
		middleware.RequirePermission("roles:update", func(w http.ResponseWriter, r *http.Request) {
			var req struct {
				Name        string   `json:"name"`
				Permissions []string `json:"permissions"`
			}

			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"message": "Payload JSON tidak valid"})
				return
			}

			err := h.RoleUsecase.UpdateRoleWithPermissions(r.Context(), roleID, req.Name, req.Permissions)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
				return
			}

			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Peran berhasil diperbarui"})
		})(w, r)
		return
	}

	// 🌟 4. JALUR DELETE DATA PERAN (METHOD DELETE)
	if r.Method == http.MethodDelete {
		middleware.RequirePermission("roles:delete", func(w http.ResponseWriter, r *http.Request) {
			err := h.RoleUsecase.DeleteRoleWithPermissions(r.Context(), roleID)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
				return
			}

			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Peran berhasil dihapus"})
		})(w, r)
		return
	}

	w.WriteHeader(http.StatusMethodNotAllowed)
}

// GetAllMasterPermissions menangani request HTTP GET untuk menyuplai daftar centang di frontend
func (h *RoleHandler) GetAllMasterPermissions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Metode HTTP tidak diizinkan"})
		return
	}

	middleware.RequirePermission("roles:read", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		permissions, err := h.RoleUsecase.GetAllMasterPermissions(ctx)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "Gagal memuat list permissions: " + err.Error()})
			return
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "success",
			"data":   permissions,
		})
	})(w, r)
}
