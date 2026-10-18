package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"go_backend/internal/domain"
	"go_backend/internal/middleware"
)

type SupplierHandler struct {
	SupplierUsecase domain.SupplierUsecase
}

// NewSupplierHandler menginisialisasi handler untuk modul supplier
func NewSupplierHandler(u domain.SupplierUsecase) *SupplierHandler {
	return &SupplierHandler{SupplierUsecase: u}
}

// SuppliersCRUDHandler mengorkestrasi HTTP Method terpusat (RESTful murni terintegrasi RBAC & Finansial)
func (h *SupplierHandler) SuppliersCRUDHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 🌟 1. JALUR READ DATA SUPPLIER (METHOD GET - CURSOR PAGINATION & BANK LEFT JOIN)
	if r.Method == http.MethodGet {
		middleware.RequirePermission("suppliers:read", func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			search := r.URL.Query().Get("search")
			province := r.URL.Query().Get("province_id")
			if province == "" {
				province = "ALL"
			}

			// Amankan parsing cursor_id agar default ke 0 jika kosong atau tidak valid
			var cursorID int64 = 0
			if cStr := r.URL.Query().Get("cursor_id"); cStr != "" && cStr != "undefined" {
				id, err := strconv.ParseInt(cStr, 10, 64)
				if err == nil {
					cursorID = id
				}
			}

			limit := 10
			if lStr := r.URL.Query().Get("limit"); lStr != "" {
				l, err := strconv.Atoi(lStr)
				if err == nil && l > 0 {
					limit = l
				}
			}

			direction := r.URL.Query().Get("direction")
			if direction == "" {
				direction = "next"
			}

			// Tarik data gabungan supplier + regional + bank dari level bisnis usecase
			suppliers, total, err := h.SupplierUsecase.GetAllSuppliers(ctx, search, province, cursorID, limit, direction)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"message": "Gagal kueri data: " + err.Error()})
				return
			}

			// Tentukan nilai penanda halaman kursor Next & Prev secara sah memakai indeks array
			var nextCursor int64 = 0
			var prevCursor int64 = 0
			if len(suppliers) > 0 {
				nextCursor = suppliers[len(suppliers)-1].ID // ID paling bawah untuk Next
				prevCursor = suppliers[0].ID                // ID paling atas untuk Prev
			}

			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "success",
				"data":   suppliers,
				"total":  total,
				"pagination": map[string]any{
					"next_cursor": nextCursor,
					"prev_cursor": prevCursor,
					"has_more":    len(suppliers) >= limit,
				},
			})
		})(w, r)
		return
	}

	// 🌟 2. JALUR CREATE DATA SUPPLIER (METHOD POST)
	if r.Method == http.MethodPost {
		middleware.RequirePermission("suppliers:create", func(w http.ResponseWriter, r *http.Request) {
			// Ambil user_id aktif dari context JWT untuk kebutuhan Audit Trail database
			userID := middleware.CurrentUserID(r)

			var req struct {
				Name          string `json:"name"`
				PicName       string `json:"pic_name"`
				Phone         string `json:"phone"`
				Email         string `json:"email"`
				Address       string `json:"address"`
				VillageID     string `json:"village_id"`
				BankID        *int32 `json:"bank_id"`        // 🌟 Tangkap Properti Keuangan Baru
				AccountNumber string `json:"account_number"` // 🌟 Tangkap Properti Keuangan Baru
				AccountName   string `json:"account_name"`   // 🌟 Tangkap Properti Keuangan Baru
			}

			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"message": "Payload request JSON tidak valid"})
				return
			}

			newSupplier := &domain.Supplier{
				Name:          req.Name,
				PicName:       req.PicName,
				Phone:         req.Phone,
				Email:         req.Email,
				Address:       req.Address,
				VillageID:     req.VillageID,
				BankID:        req.BankID,        // Oper data perbankan ke level usecase
				AccountNumber: req.AccountNumber, // Oper data perbankan ke level usecase
				AccountName:   req.AccountName,   // Oper data perbankan ke level usecase
				CreatedBy:     &userID,
				UpdatedBy:     &userID,
			}

			err := h.SupplierUsecase.CreateSupplier(r.Context(), newSupplier)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
				return
			}

			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status":  "success",
				"message": "Mitra supplier baru berhasil didaftarkan bersama rekening finansial",
				"data":    newSupplier,
			})
		})(w, r)
		return
	}

	// Ambil penanda parameter query string ID untuk rute PUT dan DELETE (?id=...)
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Parameter ID supplier wajib dicantumkan"})
		return
	}
	idConverted, _ := strconv.ParseInt(idStr, 10, 64)

	// 🌟 3. JALUR UPDATE DATA SUPPLIER (METHOD PUT)
	if r.Method == http.MethodPut {
		middleware.RequirePermission("suppliers:update", func(w http.ResponseWriter, r *http.Request) {
			userID := middleware.CurrentUserID(r)

			var req struct {
				Name          string `json:"name"`
				PicName       string `json:"pic_name"`
				Phone         string `json:"phone"`
				Email         string `json:"email"`
				Address       string `json:"address"`
				VillageID     string `json:"village_id"`
				BankID        *int32 `json:"bank_id"`        // 🌟 Tangkap Pembaruan Keuangan
				AccountNumber string `json:"account_number"` // 🌟 Tangkap Pembaruan Keuangan
				AccountName   string `json:"account_name"`   // 🌟 Tangkap Pembaruan Keuangan
				IsActive      bool   `json:"is_active"`
			}

			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			updatedSupplier := &domain.Supplier{
				ID:            idConverted,
				Name:          req.Name,
				PicName:       req.PicName,
				Phone:         req.Phone,
				Email:         req.Email,
				Address:       req.Address,
				VillageID:     req.VillageID,
				BankID:        req.BankID,        // Oper pembaruan ke level usecase
				AccountNumber: req.AccountNumber, // Oper pembaruan ke level usecase
				AccountName:   req.AccountName,
				IsActive:      req.IsActive,
				UpdatedBy:     &userID,
			}

			err := h.SupplierUsecase.UpdateSupplier(r.Context(), updatedSupplier)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
				return
			}

			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Profil data supplier dan informasi bank berhasil diperbarui"})
		})(w, r)
		return
	}

	// 🌟 4. JALUR DELETE DATA SUPPLIER (METHOD DELETE)
	if r.Method == http.MethodDelete {
		middleware.RequirePermission("suppliers:delete", func(w http.ResponseWriter, r *http.Request) {
			userID := middleware.CurrentUserID(r)

			err := h.SupplierUsecase.DeleteSupplier(r.Context(), idConverted, userID)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
				return
			}

			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Data supplier berhasil dihapus"})
		})(w, r)
		return
	}

	w.WriteHeader(http.StatusMethodNotAllowed)
}
