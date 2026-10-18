package handler

import (
	"encoding/json"
	"go_backend/internal/domain"
	"go_backend/internal/middleware"
	"net/http"
	"strings"
)

type UnitHandler struct {
	UnitUsecase domain.UnitUsecase
}

func NewUnitHandler(u domain.UnitUsecase) *UnitHandler {
	return &UnitHandler{UnitUsecase: u}
}

func (h *UnitHandler) HandleGetUnits(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	// 🌟 1. JALUR BACA DATA (GET)
	if r.Method == http.MethodGet {
		middleware.RequirePermission("units:read", func(w http.ResponseWriter, r *http.Request) {
			data, err := h.UnitUsecase.GetAllUnits(r.Context())
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
				return
			}

			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "success",
				"data":   data,
			})
		})(w, r)
		return
	}

	// 🌟 2. JALUR BUAT DATA BARU (POST) - MENYELAMATKAN DARI EROR 405
	if r.Method == http.MethodPost {
		middleware.RequirePermission("units:create", func(w http.ResponseWriter, r *http.Request) {
			userID := middleware.CurrentUserID(r)

			var req struct {
				Kode        string `json:"kode"`
				Name        string `json:"name"`
				Description string `json:"description"`
			}

			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"message": "Format payload JSON tidak valid"})
				return
			}

			// Merakit objek entitas domain unit
			newUnit := &domain.Unit{
				Kode:        req.Kode,
				Name:        req.Name,
				Description: req.Description,
				CreatedBy:   &userID,
				UpdatedBy:   &userID,
			}

			// ⚡ TEMBAK KE DATABASE POSTGRESQL LEWAT LAYER USECASE
			if err := h.UnitUsecase.CreateUnit(r.Context(), newUnit); err != nil {
				// 🌟 INTERSEPSI EROR DUPLIKAT (Hukum Defensif Kode 23505)
				// Memeriksa apakah eror disebabkan karena kode satuan sudah terdaftar di PostgreSQL
				if strings.Contains(err.Error(), "23505") || strings.Contains(err.Error(), "unique_constraint") || strings.Contains(err.Error(), "duplicate key") {
					w.WriteHeader(http.StatusConflict) // Gunakan HTTP Status 409 Conflict untuk data kembar
					_ = json.NewEncoder(w).Encode(map[string]string{
						"status":  "error",
						"message": "Kode satuan '" + req.Kode + "' sudah terdaftar di sistem! Silakan gunakan kode singkat yang lain.",
					})
					return
				}

				// Eror umum lainnya (misal koneksi DB terputus)
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
				return
			}

			// Mengembalikan respons sukses dengan ID asli dari sequence database serial
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "success",
				"message": "Master satuan baru berhasil didaftarkan kilat ke basis data",
				"data": map[string]interface{}{
					"id":          newUnit.ID, // ✨ Mengembalikan ID fisik ril hasil INSERT PostgreSQL
					"kode":        newUnit.Kode,
					"name":        newUnit.Name,
					"description": newUnit.Description,
				},
			})
		})(w, r)
		return
	}
	// Jika ada method selain GET dan POST (misal PUT/DELETE), barulah kunci 405
	w.WriteHeader(http.StatusMethodNotAllowed)
}
