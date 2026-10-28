package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"go_backend/internal/domain"
	"go_backend/internal/middleware"
)

type ProductHandler struct {
	ProductUsecase domain.ProductUsecase
}

// NewProductHandler menginisialisasi HTTP controller gateway untuk modul produk
func NewProductHandler(u domain.ProductUsecase) *ProductHandler {
	return &ProductHandler{ProductUsecase: u}
}

// ProductsCRUDHandler mengorkestrasi seluruh mutasi jalur HTTP barang secara kaku
func (h *ProductHandler) ProductsCRUDHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 🌟 1. JALUR BACA DATA (METHOD GET - CURSOR PAGINATION MULTI-JOIN)
	if r.Method == http.MethodGet {
		middleware.RequirePermission("products:read", func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			search := r.URL.Query().Get("search")

			var catFilter int32 = 0
			if catStr := r.URL.Query().Get("category_id"); catStr != "" {
				c, _ := strconv.Atoi(catStr)
				catFilter = int32(c)
			}

			var cursorID int64 = 0
			if cStr := r.URL.Query().Get("cursor_id"); cStr != "" && cStr != "undefined" {
				id, _ := strconv.ParseInt(cStr, 10, 64)
				cursorID = id
			}

			limit := 10
			if lStr := r.URL.Query().Get("limit"); lStr != "" {
				l, _ := strconv.Atoi(lStr)
				if l > 0 {
					limit = l
				}
			}

			direction := r.URL.Query().Get("direction")

			products, total, err := h.ProductUsecase.GetAllProducts(ctx, search, catFilter, cursorID, limit, direction)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
				return
			}

			var nextCursor int64 = 0
			var prevCursor int64 = 0
			if len(products) > 0 {
				nextCursor = products[len(products)-1].ID
				prevCursor = products[0].ID
			}

			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status": "success",
				"data":   products,
				"total":  total,
				"pagination": map[string]any{
					"next_cursor": nextCursor,
					"prev_cursor": prevCursor,
					"has_more":    len(products) >= limit,
				},
			})
		})(w, r)
		return
	}

	// 🌟 2. JALUR BUAT DATA BARU (METHOD POST)
	if r.Method == http.MethodPost {
		middleware.RequirePermission("products:create", func(w http.ResponseWriter, r *http.Request) {
			userID := middleware.CurrentUserID(r)

			var req struct {
				Sku            string         `json:"sku"`
				Barcode        string         `json:"barcode"`
				Name           string         `json:"name"`
				Description    string         `json:"description"`
				CategoryID     int32          `json:"category_id"`
				SupplierID     int64          `json:"supplier_id"`
				SellingUnitID  int32          `json:"selling_unit_id"`
				PurchaseUnitID int32          `json:"purchase_unit_id"`
				ConversionRate int32          `json:"conversion_rate"`
				PurchasePrice  domain.Money   `json:"purchase_price"`
				SellingPrice   domain.Money   `json:"selling_price"`
				MinStockLevel  int32          `json:"min_stock_level"`
				DiscountType   string         `json:"discount_type"`
				DiscountValue  domain.Decimal `json:"discount_value"`
				IsTaxable      *bool          `json:"is_taxable"`
			}

			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"message": "Format JSON data tidak valid"})
				return
			}

			// Default kena PPN bila field tidak dikirim.
			isTaxable := true
			if req.IsTaxable != nil {
				isTaxable = *req.IsTaxable
			}

			newProduct := &domain.Product{
				Sku:            req.Sku,
				Barcode:        req.Barcode,
				Name:           req.Name,
				Description:    req.Description,
				CategoryID:     req.CategoryID,
				SupplierID:     req.SupplierID,
				SellingUnitID:  req.SellingUnitID,
				PurchaseUnitID: req.PurchaseUnitID,
				ConversionRate: req.ConversionRate,
				PurchasePrice:  req.PurchasePrice,
				SellingPrice:   req.SellingPrice,
				MinStockLevel:  req.MinStockLevel,
				DiscountType:   req.DiscountType,
				DiscountValue:  req.DiscountValue,
				IsTaxable:      isTaxable,
				CreatedBy:      &userID,
				UpdatedBy:      &userID,
			}

			ctx := r.Context()
			if err := h.ProductUsecase.CreateProduct(ctx, newProduct); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
				return
			}

			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"status":  "success",
				"message": "Master produk/barang baru berhasil didaftarkan ke sistem inventaris gudang",
			})
		})(w, r)
		return
	}

	// Validasi Id parameter query string (?id=...) untuk rute PUT & DELETE
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Parameter ID produk wajib dilampirkan"})
		return
	}
	idConverted, _ := strconv.ParseInt(idStr, 10, 64)

	// 🌟 3. JALUR UBAH REKOR DATA (METHOD PUT)
	if r.Method == http.MethodPut {
		middleware.RequirePermission("products:update", func(w http.ResponseWriter, r *http.Request) {
			userID := middleware.CurrentUserID(r)

			var req struct {
				Sku            string         `json:"sku"`
				Barcode        string         `json:"barcode"`
				Name           string         `json:"name"`
				Description    string         `json:"description"`
				CategoryID     int32          `json:"category_id"`
				SupplierID     int64          `json:"supplier_id"`
				SellingUnitID  int32          `json:"selling_unit_id"`
				PurchaseUnitID int32          `json:"purchase_unit_id"`
				ConversionRate int32          `json:"conversion_rate"`
				PurchasePrice  domain.Money   `json:"purchase_price"`
				SellingPrice   domain.Money   `json:"selling_price"`
				MinStockLevel  int32          `json:"min_stock_level"`
				DiscountType   string         `json:"discount_type"`
				DiscountValue  domain.Decimal `json:"discount_value"`
				IsTaxable      *bool          `json:"is_taxable"`
				IsActive       bool           `json:"is_active"`
			}

			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			isTaxable := true
			if req.IsTaxable != nil {
				isTaxable = *req.IsTaxable
			}

			updatedProduct := &domain.Product{
				ID:             idConverted,
				Sku:            req.Sku,
				Barcode:        req.Barcode,
				Name:           req.Name,
				Description:    req.Description,
				CategoryID:     req.CategoryID,
				SupplierID:     req.SupplierID,
				SellingUnitID:  req.SellingUnitID,
				PurchaseUnitID: req.PurchaseUnitID,
				ConversionRate: req.ConversionRate,
				PurchasePrice:  req.PurchasePrice,
				SellingPrice:   req.SellingPrice,
				MinStockLevel:  req.MinStockLevel,
				DiscountType:   req.DiscountType,
				DiscountValue:  req.DiscountValue,
				IsTaxable:      isTaxable,
				IsActive:       req.IsActive,
				UpdatedBy:      &userID,
			}

			if err := h.ProductUsecase.UpdateProduct(r.Context(), updatedProduct); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
				return
			}

			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Spesifikasi data produk berhasil diperbarui"})
		})(w, r)
		return
	}

	// 🌟 4. JALUR PENGHAPUSAN SOFT-DELETE (METHOD DELETE)
	if r.Method == http.MethodDelete {
		middleware.RequirePermission("products:delete", func(w http.ResponseWriter, r *http.Request) {
			userID := middleware.CurrentUserID(r)

			if err := h.ProductUsecase.DeleteProduct(r.Context(), idConverted, userID); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
				return
			}

			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Produk berhasil dicabut dari inventaris master"})
		})(w, r)
		return
	}

	w.WriteHeader(http.StatusMethodNotAllowed)
}
