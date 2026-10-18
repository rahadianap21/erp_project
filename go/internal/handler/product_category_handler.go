package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"go_backend/internal/domain"
)

type ProductCategoryHandler struct {
	Usecase domain.ProductCategoryUsecase
}

func NewProductCategoryHandler(u domain.ProductCategoryUsecase) *ProductCategoryHandler {
	return &ProductCategoryHandler{Usecase: u}
}

func (h *ProductCategoryHandler) CategoriesCRUDHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 1. JALUR GET DATA (CURSOR PAGINATION BASED)
	if r.Method == http.MethodGet {
		RequirePermission("categories:read", func(w http.ResponseWriter, r *http.Request) {
			search := r.URL.Query().Get("search")

			var cursorID int32 = 0
			if cStr := r.URL.Query().Get("cursor_id"); cStr != "" && cStr != "undefined" {
				id, _ := strconv.ParseInt(cStr, 10, 32)
				cursorID = int32(id)
			}

			limit := 10
			if lStr := r.URL.Query().Get("limit"); lStr != "" {
				l, _ := strconv.Atoi(lStr)
				if l > 0 {
					limit = l
				}
			}
			direction := r.URL.Query().Get("direction")

			data, total, err := h.Usecase.GetAllCategories(r.Context(), search, cursorID, limit, direction)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
				return
			}

			var nextCursor int32 = 0
			var prevCursor int32 = 0
			if len(data) > 0 {
				nextCursor = data[len(data)-1].ID
				prevCursor = data[0].ID
			}

			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "success",
				"data":   data,
				"total":  total,
				"pagination": map[string]interface{}{
					"next_cursor": nextCursor,
					"prev_cursor": prevCursor,
					"has_more":    len(data) >= limit,
				},
			})
		})(w, r)
		return
	}

	// 2. JALUR CREATE DATA (POST)
	if r.Method == http.MethodPost {
		RequirePermission("categories:create", func(w http.ResponseWriter, r *http.Request) {
			userID := CurrentUserID(r)

			var req struct {
				Name        string `json:"name"`
				Slug        string `json:"slug"`
				Description string `json:"description"`
				ParentID    *int32 `json:"parent_id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			newCat := &domain.ProductCategory{
				Name:        req.Name,
				Slug:        req.Slug,
				Description: req.Description,
				ParentID:    req.ParentID,
				CreatedBy:   &userID,
				UpdatedBy:   &userID,
			}

			if err := h.Usecase.CreateCategory(r.Context(), newCat); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
				return
			}

			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "message": "Kategori produk baru berhasil ditambahkan"})
		})(w, r)
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Parameter ID kategori wajib disertakan"})
		return
	}
	idConverted, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil || idConverted <= 0 {
		// Salah ketik / ID kosong adalah kesalahan input, bukan kegagalan server.
		// Sebelumnya error parse diabaikan sehingga ID "abc" menjadi 0 lalu
		// usecase menolaknya sebagai 500.
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "ID kategori tidak valid"})
		return
	}

	// 3. JALUR UPDATE DATA (PUT)
	if r.Method == http.MethodPut {
		RequirePermission("categories:update", func(w http.ResponseWriter, r *http.Request) {
			userID := CurrentUserID(r)

			var req struct {
				Name        string `json:"name"`
				Slug        string `json:"slug"`
				Description string `json:"description"`
				ParentID    *int32 `json:"parent_id"`
				IsActive    bool   `json:"is_active"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			updatedCat := &domain.ProductCategory{
				ID:          int32(idConverted),
				Name:        req.Name,
				Slug:        req.Slug,
				Description: req.Description,
				ParentID:    req.ParentID,
				IsActive:    req.IsActive,
				UpdatedBy:   &userID,
			}

			if err := h.Usecase.UpdateCategory(r.Context(), updatedCat); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
				return
			}

			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Kategori produk berhasil diperbarui"})
		})(w, r)
		return
	}

	// 4. JALUR DELETE DATA (DELETE)
	if r.Method == http.MethodDelete {
		RequirePermission("categories:delete", func(w http.ResponseWriter, r *http.Request) {
			userID := CurrentUserID(r)

			if err := h.Usecase.DeleteCategory(r.Context(), int32(idConverted), userID); err != nil {
				if errors.Is(err, domain.ErrKategoriTidakDitemukan) {
					w.WriteHeader(http.StatusNotFound)
					_ = json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
					return
				}
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
				return
			}

			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Kategori produk berhasil dihapus"})
		})(w, r)
		return
	}

	w.WriteHeader(http.StatusMethodNotAllowed)
}
