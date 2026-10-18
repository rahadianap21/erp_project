package handler

import (
	"encoding/json"
	"go_backend/internal/domain"
	"go_backend/internal/middleware"
	"net/http"
	"strconv"
)

type PurchaseOrderHandler struct {
	PoUsecase domain.PurchaseOrderUsecase
}

func NewPurchaseOrderHandler(u domain.PurchaseOrderUsecase) *PurchaseOrderHandler {
	return &PurchaseOrderHandler{PoUsecase: u}
}

// HandlePurchaseOrders melayani registrasi dokumen (POST) dan pembacaan list (GET)
func (h *PurchaseOrderHandler) HandlePurchaseOrders(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// 🌟 1. JALUR AMAN CREATION DATA (POST)
	if r.Method == http.MethodPost {
		middleware.RequirePermission("po:create", func(w http.ResponseWriter, r *http.Request) {
			userID := middleware.CurrentUserID(r)

			// Parsing Payload request perluasan parameter pajak dari Next.js klien
			var req struct {
				SupplierID int64  `json:"supplier_id"`
				TaxType    string `json:"tax_type"` // 🌟 Bidang Pajak Baru
				Notes      string `json:"notes"`
				Items      []struct {
					ProductID  int64 `json:"product_id"`
					QtyOrdered int32 `json:"qty_ordered"`
				} `json:"items"`
			}

			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"message": "Format payload JSON transaksional tidak valid"})
				return
			}

			var detailItems []domain.PurchaseOrderDetail
			for _, item := range req.Items {
				detailItems = append(detailItems, domain.PurchaseOrderDetail{
					ProductID:  item.ProductID,
					QtyOrdered: item.QtyOrdered,
				})
			}

			newPO := &domain.PurchaseOrder{
				SupplierID: req.SupplierID,
				TaxType:    req.TaxType, // ✨ Teruskan tipe pajak ke tingkat usecase
				Notes:      req.Notes,
				CreatedBy:  &userID,
				Items:      detailItems,
			}

			if err := h.PoUsecase.CreatePO(r.Context(), newPO); err != nil {
				WriteError(w, err)
				return
			}

			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "success",
				"message": "Dokumen Purchase Order berhasil diterbitkan otomatis",
				"data": map[string]interface{}{
					"id":        newPO.ID,
					"po_number": newPO.PoNumber,
					"dpp":       newPO.DppAmount,
					"tax":       newPO.TaxAmount,
					"amount":    newPO.TotalAmount,
					"status":    newPO.Status,
				},
			})
		})(w, r)
		return
	}

	// 🌟 2. JALUR DAFTAR DATA DENGAN CURSOR PAGINATION (GET)
	if r.Method == http.MethodGet {
		middleware.RequirePermission("po:read", func(w http.ResponseWriter, r *http.Request) {
			q := r.URL.Query()

			// 1. ✨ SEKRUP PENYELAMAT: Periksa apakah frontend meminta detail item tunggal via ?id=
			idStr := q.Get("id")
			if idStr != "" {
				poID, err := strconv.ParseInt(idStr, 10, 64)
				if err != nil || poID <= 0 {
					w.WriteHeader(http.StatusBadRequest)
					_ = json.NewEncoder(w).Encode(map[string]string{"message": "Parameter ID dokumen PO tidak valid"})
					return
				}

				// Tarik data utuh header + berserta baris detail itemnya sekaligus dari database Go
				singlePO, err := h.PoUsecase.FindPOByID(r.Context(), poID)
				if err != nil {
					WriteError(w, err)
					return
				}

				w.WriteHeader(http.StatusOK)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"status": "success",
					"data":   singlePO, // Mengembalikan objek PurchaseOrder lengkap berserta .Items
				})
				return // 🔒 Potong eksekusi agar tidak turun ke area list pagination di bawah!
			}

			// 2. Jalur Pembacaan List Global (Cursor Pagination bawaan Anda - Biarkan Utuh)
			cursorID, _ := strconv.ParseInt(q.Get("cursor_id"), 10, 64)
			limitVal, _ := strconv.ParseInt(q.Get("limit"), 10, 32)
			if limitVal <= 0 {
				limitVal = 10
			}

			param := domain.PurchaseOrderParam{
				Search:       q.Get("search"),
				StatusFilter: q.Get("status"),
				CursorID:     cursorID,
				Limit:        int32(limitVal),
				Direction:    q.Get("direction"),
			}

			list, total, err := h.PoUsecase.FindAllPO(r.Context(), param)
			if err != nil {
				WriteError(w, err)
				return
			}

			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "success",
				"data":   list,
				"total":  total,
			})
		})(w, r)
		return
	}

	// 🌟 3. JALUR KOREKSI DOKUMEN (PUT) - dipanggil dari form edit Purchase Order
	if r.Method == http.MethodPut {
		idStr := r.URL.Query().Get("id")
		poID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || poID <= 0 {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "Parameter ID dokumen PO tidak valid"})
			return
		}

		middleware.RequirePermission("po:update", func(w http.ResponseWriter, r *http.Request) {
			userID := middleware.CurrentUserID(r)

			// Sengaja TIDAK ada field harga di payload. UnitPrice & PurchaseUnitID
			// diturunkan usecase dari master produk supaya harga PO tidak bisa
			// dimanipulasi dari sisi klien.
			var req struct {
				TaxType string `json:"tax_type"`
				Notes   string `json:"notes"`
				Items   []struct {
					ProductID  int64 `json:"product_id"`
					QtyOrdered int32 `json:"qty_ordered"`
				} `json:"items"`
			}

			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]string{"message": "Format payload JSON transaksional tidak valid"})
				return
			}

			input := domain.PurchaseOrderUpdateInput{
				TaxType: req.TaxType,
				Notes:   req.Notes,
			}
			for _, item := range req.Items {
				input.Items = append(input.Items, domain.PurchaseOrderItemInput{
					ProductID:  item.ProductID,
					QtyOrdered: item.QtyOrdered,
				})
			}

			if err := h.PoUsecase.UpdatePO(r.Context(), poID, userID, input); err != nil {
				WriteError(w, err)
				return
			}

			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "success",
				"message": "Dokumen Purchase Order berhasil dikoreksi",
			})
		})(w, r)
		return
	}

	// 🌟 4. JALUR PENCABUTAN DOKUMEN (DELETE) - soft delete, isi kolom deleted_at
	if r.Method == http.MethodDelete {
		idStr := r.URL.Query().Get("id")
		poID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || poID <= 0 {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "Parameter ID dokumen PO tidak valid"})
			return
		}

		middleware.RequirePermission("po:delete", func(w http.ResponseWriter, r *http.Request) {
			userID := middleware.CurrentUserID(r)

			if err := h.PoUsecase.DeletePO(r.Context(), poID, userID); err != nil {
				WriteError(w, err)
				return
			}

			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "success",
				"message": "Dokumen Purchase Order berhasil dicabut dari daftar aktif",
			})
		})(w, r)
		return
	}

	w.WriteHeader(http.StatusMethodNotAllowed)
}

// HandleApprovePO melayani pengesahan dokumen (PUT /api/purchase-orders/approve?id=)
func (h *PurchaseOrderHandler) HandleApprovePO(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	middleware.RequirePermission("po:approve", func(w http.ResponseWriter, r *http.Request) {
		poID, err := parsePOID(r)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
			return
		}

		userID := middleware.CurrentUserID(r)
		if err := h.PoUsecase.ApprovePO(r.Context(), poID, userID); err != nil {
			WriteError(w, err)
			return
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "success",
			"message": "Dokumen Purchase Order berhasil disetujui dan Dismbuskan ke vendor",
		})
	})(w, r)
}

// HandleCancelPO melayani pembatalan dokumen (PUT /api/purchase-orders/cancel?id=)
//
// Catatan permission: endpoint ini digate 'po:reject', bukan 'po:cancel'.
// Permission 'po:cancel' belum terdaftar di s_permissions sehingga tidak ada role
// yang memilikinya. Frontend membaca 'po:cancel', jadi tombol batal di UI TIDAK
// AKAN muncul sampai permission tersebut dibuat.
func (h *PurchaseOrderHandler) HandleCancelPO(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	middleware.RequirePermission("po:reject", func(w http.ResponseWriter, r *http.Request) {
		poID, err := parsePOID(r)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
			return
		}

		userID := middleware.CurrentUserID(r)
		if err := h.PoUsecase.CancelPO(r.Context(), poID, userID); err != nil {
			WriteError(w, err)
			return
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "success",
			"message": "Dokumen Purchase Order berhasil dibatalkan",
		})
	})(w, r)
}

// HandleRejectPO melayani penolakan dokumen PO (PUT /api/purchase-orders/reject?id=)
func (h *PurchaseOrderHandler) HandleRejectPO(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	middleware.RequirePermission("po:reject", func(w http.ResponseWriter, r *http.Request) {
		poID, err := parsePOID(r)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
			return
		}

		var payload struct {
			Reason string `json:"reason"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "body tidak valid"})
			return
		}

		userID := middleware.CurrentUserID(r)
		if err := h.PoUsecase.RejectPO(r.Context(), poID, userID, payload.Reason); err != nil {
			WriteError(w, err)
			return
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "success",
			"message": "Dokumen Purchase Order berhasil ditolak",
		})
	})(w, r)
}
