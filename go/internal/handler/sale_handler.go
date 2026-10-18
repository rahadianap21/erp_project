package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"go_backend/internal/domain"
	"go_backend/internal/middleware"
)

type SaleHandler struct {
	uc    domain.SaleUsecase
	posUC domain.POSUsecase
}

func NewSaleHandler(uc domain.SaleUsecase, posUC domain.POSUsecase) *SaleHandler {
	return &SaleHandler{uc: uc, posUC: posUC}
}

// statusUntukErrorSale memetakan error domain penjualan menjadi kode HTTP.
func statusUntukErrorSale(err error) int {
	switch {
	case err == nil:
		return http.StatusOK
	case errors.Is(err, domain.ErrSaleNotFound), errors.Is(err, domain.ErrJournalNotFound):
		return http.StatusNotFound
	case errors.Is(err, domain.ErrSaleAlreadyVoid), errors.Is(err, domain.ErrSaleNotPosted),
		errors.Is(err, domain.ErrJournalAlreadyRev):
		return http.StatusConflict
	case errors.Is(err, domain.ErrSaleItemsRequired),
		errors.Is(err, domain.ErrSaleLineInvalid), errors.Is(err, domain.ErrSaleProductNotFound),
		errors.Is(err, domain.ErrSaleStokTidakCukup), errors.Is(err, domain.ErrSalePaymentInvalid),
		errors.Is(err, domain.ErrSalePaymentExceeds), errors.Is(err, domain.ErrSalePaymentKurang),
		errors.Is(err, domain.ErrSaleMetodeGanda), errors.Is(err, domain.ErrSaleRedeemTanpaMember),
		errors.Is(err, domain.ErrSaleRedeemMelebihiSaldo),
		errors.Is(err, domain.ErrSaleTaxInvalid),
		errors.Is(err, domain.ErrSaleDiscountInvalid), errors.Is(err, domain.ErrAutoPostingSettings),
		errors.Is(err, domain.ErrJournalUnbalanced), errors.Is(err, domain.ErrJournalPeriodClosed):
		return http.StatusUnprocessableEntity
	default:
		return http.StatusInternalServerError
	}
}

func writeSaleError(w http.ResponseWriter, err error) {
	status := statusUntukErrorSale(err)
	msg := err.Error()
	if msg == "" {
		msg = "terjadi kesalahan pada transaksi penjualan"
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": msg})
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst interface{}) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "body tidak valid: " + err.Error()})
		return false
	}
	return true
}

// HandleSales melayani daftar/detail (GET) dan pembuatan (POST) penjualan.
func (h *SaleHandler) HandleSales(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodPost:
		protected := middleware.RequirePermission("sales:create", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var in domain.CreateSaleInput
			if !decodeJSONBody(w, r, &in) {
				return
			}
			uid := middleware.CurrentUserID(r)
			if uid <= 0 {
				w.WriteHeader(http.StatusUnauthorized)
				_ = json.NewEncoder(w).Encode(map[string]string{"message": "tidak terautentikasi"})
				return
			}
			in.CreatedBy = uid
			sale, err := h.uc.CreateSale(r.Context(), in)
			if err != nil {
				writeSaleError(w, err)
				return
			}
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"message": "penjualan berhasil dibuat", "data": sale})
		}))
		protected(w, r)
		return

	case http.MethodGet:
		protected := middleware.RequirePermission("sales:read", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			q := r.URL.Query()
			if idStr := q.Get("id"); idStr != "" {
				id, err := strconv.ParseInt(idStr, 10, 64)
				if err != nil || id <= 0 {
					w.WriteHeader(http.StatusBadRequest)
					_ = json.NewEncoder(w).Encode(map[string]string{"message": "id penjualan tidak valid"})
					return
				}
				sale, err := h.uc.GetByID(r.Context(), id)
				if err != nil {
					writeSaleError(w, err)
					return
				}
				if sale == nil {
					w.WriteHeader(http.StatusNotFound)
					_ = json.NewEncoder(w).Encode(map[string]string{"message": "penjualan tidak ditemukan"})
					return
				}
				_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": sale})
				return
			}
			limit, _ := strconv.Atoi(q.Get("limit"))
			cursor, _ := strconv.ParseInt(q.Get("cursor_id"), 10, 64)
			items, total, err := h.uc.GetAll(r.Context(), domain.SaleListParam{
				Search:       q.Get("search"),
				StatusFilter: q.Get("status"),
				CursorID:     cursor,
				Limit:        limit,
				Direction:    q.Get("direction"),
			})
			if err != nil {
				writeSaleError(w, err)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": items, "total": total})
		}))
		protected(w, r)
		return

	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
}

// HandleSaleVoid membatalkan penjualan POSTED (POST /api/sales/void?id=).
func (h *SaleHandler) HandleSaleVoid(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	protected := middleware.RequirePermission("sales:void", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
		if err != nil || id <= 0 {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "id penjualan tidak valid"})
			return
		}
		uid := middleware.CurrentUserID(r)
		if uid <= 0 {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "tidak terautentikasi"})
			return
		}
		var body struct {
			Reason string `json:"reason"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if err := h.uc.VoidSale(r.Context(), id, uid, body.Reason); err != nil {
			writeSaleError(w, err)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "penjualan berhasil dibatalkan"})
	}))
	protected(w, r)
}

// HandlePOSBootstrap menyuplai data awal layar kasir (metode bayar, kategori, tarif PPN).
func (h *SaleHandler) HandlePOSBootstrap(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	protected := middleware.RequirePermission("pos:read", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := h.posUC.Bootstrap(r.Context())
		if err != nil {
			writeSaleError(w, err)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": data})
	}))
	protected(w, r)
}

// HandlePOSProducts melayani pencarian produk untuk grid kasir.
func (h *SaleHandler) HandlePOSProducts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	protected := middleware.RequirePermission("pos:read", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		limit, _ := strconv.Atoi(q.Get("limit"))
		category, _ := strconv.Atoi(q.Get("category_id"))
		items, err := h.posUC.SearchProducts(r.Context(), q.Get("search"), category, limit)
		if err != nil {
			writeSaleError(w, err)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": items})
	}))
	protected(w, r)
}

// HandlePOSMembers melayani pencarian member dari layar kasir.
func (h *SaleHandler) HandlePOSMembers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	protected := middleware.RequirePermission("pos:read", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		limit, _ := strconv.Atoi(q.Get("limit"))
		items, err := h.posUC.SearchMembers(r.Context(), q.Get("search"), limit)
		if err != nil {
			writeSaleError(w, err)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": items})
	}))
	protected(w, r)
}

// isUniqueViolation mendeteksi pelanggaran unique (kode duplikat) dari pesan pgx.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "23505") || strings.Contains(msg, "duplicate key value")
}
