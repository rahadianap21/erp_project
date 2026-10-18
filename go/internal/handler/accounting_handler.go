package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"go_backend/internal/domain"
	"go_backend/internal/middleware"
)

// AccountingHandler mengelompokkan endpoint modul akuntansi: bagan akun,
// periode fiskal, dan jurnal umum. Satu struct dipakai karena ketiga modul
// berbagi pola respons dan pemetaan error yang sama.
type AccountingHandler struct {
	CoaUsecase     domain.ChartOfAccountUsecase
	PeriodUsecase  domain.AccountingPeriodUsecase
	JournalUsecase domain.JournalUsecase
	ReportUsecase  domain.AccountingReportUsecase
}

func NewAccountingHandler(
	coa domain.ChartOfAccountUsecase,
	period domain.AccountingPeriodUsecase,
	journal domain.JournalUsecase,
	report domain.AccountingReportUsecase,
) *AccountingHandler {
	return &AccountingHandler{
		CoaUsecase:     coa,
		PeriodUsecase:  period,
		JournalUsecase: journal,
		ReportUsecase:  report,
	}
}

// statusUntukAkuntansi memetakan error domain akuntansi ke status HTTP.
// Pelanggaran aturan bisnis dikembalikan sebagai 422 agar frontend bisa
// menampilkan pesannya; error lainnya tetap 500.
func statusUntukAkuntansi(err error) int {
	switch {
	case errors.Is(err, domain.ErrJournalUnbalanced),
		errors.Is(err, domain.ErrJournalLineDouble),
		errors.Is(err, domain.ErrAccountDuplicate),
		errors.Is(err, domain.ErrKodeAkunWajib),
		errors.Is(err, domain.ErrNamaAkunWajib),
		errors.Is(err, domain.ErrTipeAkunInvalid),
		errors.Is(err, domain.ErrSaldoNormalMismatch),
		errors.Is(err, domain.ErrIndukAkunNotFound),
		errors.Is(err, domain.ErrAccountNotFound),
		errors.Is(err, domain.ErrJournalLineKosong),
		errors.Is(err, domain.ErrJournalDateInvalid),
		errors.Is(err, domain.ErrJournalPeriodClosed),
		errors.Is(err, domain.ErrVoidPostedJournal),
		errors.Is(err, domain.ErrJournalNotPosted),
		errors.Is(err, domain.ErrJournalAlreadyPost),
		errors.Is(err, domain.ErrJournalAlreadyVoid),
		errors.Is(err, domain.ErrJournalAlreadyRev),
		errors.Is(err, domain.ErrJournalNotDraft),
		errors.Is(err, domain.ErrJournalPostBox),
		errors.Is(err, domain.ErrPeriodNotFound),
		errors.Is(err, domain.ErrPeriodClosed),
		errors.Is(err, domain.ErrPeriodOpen),
		errors.Is(err, domain.ErrPeriodeInvalid),
		errors.Is(err, domain.ErrPeriodeBerubah),
		errors.Is(err, domain.ErrRentangTanggalInvalid):
		return http.StatusUnprocessableEntity
	}
	var jnp *domain.JournalNotPostableError
	if errors.As(err, &jnp) {
		return http.StatusUnprocessableEntity
	}
	return http.StatusInternalServerError
}

func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func respondError(w http.ResponseWriter, err error) {
	respondJSON(w, statusUntukAkuntansi(err), map[string]string{"message": err.Error()})
}

func parseInt64Param(r *http.Request, key string) (int64, bool) {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return 0, false
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

// --- Bagan Akun ---

// AccountsCRUDHandler melayani GET (list/detail), POST, PUT, DELETE bagan akun.
func (h *AccountingHandler) AccountsCRUDHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		middleware.RequirePermission("accounting:read", func(w http.ResponseWriter, r *http.Request) {
			if id, ok := parseInt64Param(r, "id"); ok && id > 0 {
				acc, err := h.CoaUsecase.FindAccountByID(r.Context(), id)
				if err != nil {
					respondError(w, err)
					return
				}
				respondJSON(w, http.StatusOK, map[string]interface{}{"status": "success", "data": acc})
				return
			}
			cursorID, _ := parseInt64Param(r, "cursor_id")
			limitVal, _ := parseInt64Param(r, "limit")
			if limitVal <= 0 {
				limitVal = 20
			}
			list, total, err := h.CoaUsecase.FindAllAccounts(r.Context(), domain.ChartOfAccountParam{
				Search:    r.URL.Query().Get("search"),
				CursorID:  cursorID,
				Limit:     int32(limitVal),
				Direction: r.URL.Query().Get("direction"),
			})
			if err != nil {
				respondError(w, err)
				return
			}
			respondJSON(w, http.StatusOK, map[string]interface{}{"status": "success", "data": list, "total": total})
		})(w, r)

	case http.MethodPost:
		middleware.RequirePermission("accounting:create", func(w http.ResponseWriter, r *http.Request) {
			var req struct {
				Code          string `json:"code"`
				Name          string `json:"name"`
				AccountType   string `json:"account_type"`
				NormalBalance string `json:"normal_balance"`
				ParentID      *int64 `json:"parent_id"`
				IsHeader      bool   `json:"is_header"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				respondJSON(w, http.StatusBadRequest, map[string]string{"message": "Format payload JSON tidak valid"})
				return
			}
			userID := middleware.CurrentUserID(r)
			acc := &domain.ChartOfAccount{
				Code:          req.Code,
				Name:          req.Name,
				AccountType:   req.AccountType,
				NormalBalance: req.NormalBalance,
				ParentID:      req.ParentID,
				IsHeader:      req.IsHeader,
			}
			if err := h.CoaUsecase.CreateAccount(r.Context(), acc, userID); err != nil {
				respondError(w, err)
				return
			}
			respondJSON(w, http.StatusCreated, map[string]interface{}{
				"status": "success", "message": "Akun berhasil dibuat", "data": acc,
			})
		})(w, r)

	case http.MethodPut:
		middleware.RequirePermission("accounting:update", func(w http.ResponseWriter, r *http.Request) {
			id, ok := parseInt64Param(r, "id")
			if !ok || id <= 0 {
				respondJSON(w, http.StatusBadRequest, map[string]string{"message": "Parameter ID akun tidak valid"})
				return
			}
			var req struct {
				Code          string `json:"code"`
				Name          string `json:"name"`
				AccountType   string `json:"account_type"`
				NormalBalance string `json:"normal_balance"`
				ParentID      *int64 `json:"parent_id"`
				IsHeader      bool   `json:"is_header"`
				IsActive      bool   `json:"is_active"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				respondJSON(w, http.StatusBadRequest, map[string]string{"message": "Format payload JSON tidak valid"})
				return
			}
			userID := middleware.CurrentUserID(r)
			acc := &domain.ChartOfAccount{
				ID:            id,
				Code:          req.Code,
				Name:          req.Name,
				AccountType:   req.AccountType,
				NormalBalance: req.NormalBalance,
				ParentID:      req.ParentID,
				IsHeader:      req.IsHeader,
				IsActive:      req.IsActive,
			}
			if err := h.CoaUsecase.UpdateAccount(r.Context(), acc, userID); err != nil {
				respondError(w, err)
				return
			}
			respondJSON(w, http.StatusOK, map[string]interface{}{
				"status": "success", "message": "Akun berhasil dikoreksi",
			})
		})(w, r)

	case http.MethodDelete:
		middleware.RequirePermission("accounting:delete", func(w http.ResponseWriter, r *http.Request) {
			id, ok := parseInt64Param(r, "id")
			if !ok || id <= 0 {
				respondJSON(w, http.StatusBadRequest, map[string]string{"message": "Parameter ID akun tidak valid"})
				return
			}
			userID := middleware.CurrentUserID(r)
			if err := h.CoaUsecase.DeleteAccount(r.Context(), id, userID); err != nil {
				respondError(w, err)
				return
			}
			respondJSON(w, http.StatusOK, map[string]interface{}{
				"status": "success", "message": "Akun berhasil dihapus dari daftar aktif",
			})
		})(w, r)

	default:
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"message": "Method tidak diizinkan"})
	}
}

// --- Periode Fiskal ---

// PeriodsCRUDHandler melayani GET (list/detail) dan POST (buka periode).
func (h *AccountingHandler) PeriodsCRUDHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		middleware.RequirePermission("accounting:read", func(w http.ResponseWriter, r *http.Request) {
			if y, ok := parseInt64Param(r, "year"); ok {
				m, _ := parseInt64Param(r, "month")
				period, err := h.PeriodUsecase.GetPeriod(r.Context(), y, m)
				if err != nil {
					respondError(w, err)
					return
				}
				respondJSON(w, http.StatusOK, map[string]interface{}{"status": "success", "data": period})
				return
			}
			limitVal, _ := parseInt64Param(r, "limit")
			list, err := h.PeriodUsecase.ListPeriods(r.Context(), int(limitVal))
			if err != nil {
				respondError(w, err)
				return
			}
			respondJSON(w, http.StatusOK, map[string]interface{}{"status": "success", "data": list})
		})(w, r)

	case http.MethodPost:
		middleware.RequirePermission("accounting:close-period", func(w http.ResponseWriter, r *http.Request) {
			var req struct {
				Year  int64 `json:"year"`
				Month int64 `json:"month"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				respondJSON(w, http.StatusBadRequest, map[string]string{"message": "Format payload JSON tidak valid"})
				return
			}
			period, err := h.PeriodUsecase.OpenPeriod(r.Context(), req.Year, req.Month, middleware.CurrentUserID(r))
			if err != nil {
				respondError(w, err)
				return
			}
			respondJSON(w, http.StatusCreated, map[string]interface{}{
				"status": "success", "message": "Periode fiskal dibuka", "data": period,
			})
		})(w, r)

	default:
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"message": "Method tidak diizinkan"})
	}
}

func (h *AccountingHandler) HandleClosePeriod(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"message": "Method tidak diizinkan"})
		return
	}
	middleware.RequirePermission("accounting:close-period", func(w http.ResponseWriter, r *http.Request) {
		y, okY := parseInt64Param(r, "year")
		m, okM := parseInt64Param(r, "month")
		if !okY || !okM {
			respondJSON(w, http.StatusBadRequest, map[string]string{"message": "Parameter tahun/bulan tidak valid"})
			return
		}
		if err := h.PeriodUsecase.ClosePeriod(r.Context(), y, m, middleware.CurrentUserID(r)); err != nil {
			respondError(w, err)
			return
		}
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"status": "success", "message": "Periode fiskal ditutup",
		})
	})(w, r)
}

func (h *AccountingHandler) HandleReopenPeriod(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"message": "Method tidak diizinkan"})
		return
	}
	middleware.RequirePermission("accounting:close-period", func(w http.ResponseWriter, r *http.Request) {
		y, okY := parseInt64Param(r, "year")
		m, okM := parseInt64Param(r, "month")
		if !okY || !okM {
			respondJSON(w, http.StatusBadRequest, map[string]string{"message": "Parameter tahun/bulan tidak valid"})
			return
		}
		if err := h.PeriodUsecase.ReopenPeriod(r.Context(), int(y), int(m), middleware.CurrentUserID(r)); err != nil {
			respondError(w, err)
			return
		}
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"status": "success", "message": "Periode fiskal dibuka kembali",
		})
	})(w, r)
}

// --- Jurnal Umum ---

// JournalsCRUDHandler melayani GET (list/detail) dan POST (jurnal manual).
func (h *AccountingHandler) JournalsCRUDHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		middleware.RequirePermission("accounting:read", func(w http.ResponseWriter, r *http.Request) {
			if id, ok := parseInt64Param(r, "id"); ok && id > 0 {
				entry, err := h.JournalUsecase.FindJournalByID(r.Context(), id)
				if err != nil {
					respondError(w, err)
					return
				}
				respondJSON(w, http.StatusOK, map[string]interface{}{"status": "success", "data": entry})
				return
			}
			cursorID, _ := parseInt64Param(r, "cursor_id")
			limitVal, _ := parseInt64Param(r, "limit")
			if limitVal <= 0 {
				limitVal = 10
			}
			list, total, err := h.JournalUsecase.FindAllJournals(r.Context(), domain.JournalEntryParam{
				Search:       r.URL.Query().Get("search"),
				StatusFilter: r.URL.Query().Get("status"),
				CursorID:     cursorID,
				Limit:        int32(limitVal),
				Direction:    r.URL.Query().Get("direction"),
			})
			if err != nil {
				respondError(w, err)
				return
			}
			respondJSON(w, http.StatusOK, map[string]interface{}{"status": "success", "data": list, "total": total})
		})(w, r)

	case http.MethodPost:
		middleware.RequirePermission("accounting:create", func(w http.ResponseWriter, r *http.Request) {
			var req struct {
				EntryDate   string `json:"entry_date"`
				Description string `json:"description"`
				Lines       []struct {
					AccountID   int64   `json:"account_id"`
					Description string  `json:"description"`
					Debit       float64 `json:"debit"`
					Credit      float64 `json:"credit"`
				} `json:"lines"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				respondJSON(w, http.StatusBadRequest, map[string]string{"message": "Format payload JSON tidak valid"})
				return
			}
			entryDate, err := time.Parse("2006-01-02", req.EntryDate)
			if err != nil {
				respondJSON(w, http.StatusUnprocessableEntity, map[string]string{"message": domain.ErrJournalDateInvalid.Error()})
				return
			}
			input := domain.JournalCreateInput{
				EntryDate:   entryDate,
				Description: req.Description,
			}
			for _, l := range req.Lines {
				input.Lines = append(input.Lines, domain.JournalLineInput{
					AccountID:   l.AccountID,
					Description: l.Description,
					Debit:       l.Debit,
					Credit:      l.Credit,
				})
			}
			entry, err := h.JournalUsecase.CreateJournal(r.Context(), input, middleware.CurrentUserID(r))
			if err != nil {
				respondError(w, err)
				return
			}
			respondJSON(w, http.StatusCreated, map[string]interface{}{
				"status": "success", "message": "Jurnal berhasil dibuat (DRAFT)", "data": entry,
			})
		})(w, r)

	default:
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"message": "Method tidak diizinkan"})
	}
}

func (h *AccountingHandler) HandlePostJournal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"message": "Method tidak diizinkan"})
		return
	}
	middleware.RequirePermission("accounting:post", func(w http.ResponseWriter, r *http.Request) {
		id, ok := parseInt64Param(r, "id")
		if !ok || id <= 0 {
			respondJSON(w, http.StatusBadRequest, map[string]string{"message": "Parameter ID jurnal tidak valid"})
			return
		}
		if err := h.JournalUsecase.PostJournal(r.Context(), id, middleware.CurrentUserID(r)); err != nil {
			respondError(w, err)
			return
		}
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"status": "success", "message": "Jurnal berhasil diposting",
		})
	})(w, r)
}

func (h *AccountingHandler) HandleVoidJournal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"message": "Method tidak diizinkan"})
		return
	}
	middleware.RequirePermission("accounting:post", func(w http.ResponseWriter, r *http.Request) {
		id, ok := parseInt64Param(r, "id")
		if !ok || id <= 0 {
			respondJSON(w, http.StatusBadRequest, map[string]string{"message": "Parameter ID jurnal tidak valid"})
			return
		}
		var payload struct {
			Reason string `json:"reason"`
		}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		if err := h.JournalUsecase.VoidJournal(r.Context(), id, middleware.CurrentUserID(r), payload.Reason); err != nil {
			respondError(w, err)
			return
		}
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"status": "success", "message": "Jurnal DRAFT dibatalkan",
		})
	})(w, r)
}

func (h *AccountingHandler) HandleReverseJournal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"message": "Method tidak diizinkan"})
		return
	}
	middleware.RequirePermission("accounting:post", func(w http.ResponseWriter, r *http.Request) {
		id, ok := parseInt64Param(r, "id")
		if !ok || id <= 0 {
			respondJSON(w, http.StatusBadRequest, map[string]string{"message": "Parameter ID jurnal tidak valid"})
			return
		}
		var payload struct {
			Reason string `json:"reason"`
		}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		if err := h.JournalUsecase.ReverseJournal(r.Context(), id, middleware.CurrentUserID(r), payload.Reason); err != nil {
			respondError(w, err)
			return
		}
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"status": "success", "message": "Jurnal pembalik berhasil diterbitkan",
		})
	})(w, r)
}
