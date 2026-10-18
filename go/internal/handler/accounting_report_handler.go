package handler

import (
	"net/http"
	"time"

	"go_backend/internal/domain"
	"go_backend/internal/middleware"
)

// parseTanggalLaporan membaca parameter tanggal (format YYYY-MM-DD). Nilai
// kosong maupun format yang tidak sah dikembalikan sebagai
// ErrRentangTanggalInvalid supaya tampil sebagai pesan bisnis (422), bukan
// menghasilkan error server 500.
func parseTanggalLaporan(r *http.Request, key string) (time.Time, error) {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return time.Time{}, domain.ErrRentangTanggalInvalid
	}
	t, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return time.Time{}, domain.ErrRentangTanggalInvalid
	}
	return t, nil
}

// withReportRead membatasi endpoint laporan pada GET dan permission
// accounting:read. Bungkus respons {status, data} mengikuti endpoint lain
// di modul akuntansi.
func withReportRead(body func(w http.ResponseWriter, r *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"message": "Method tidak diizinkan"})
			return
		}
		middleware.RequirePermission("accounting:read", func(w http.ResponseWriter, r *http.Request) {
			body(w, r)
		})(w, r)
	}
}

// HandleBalanceSheet menyajikan Neraca pada satu tanggal (parameter ?as_of=).
func (h *AccountingHandler) HandleBalanceSheet(w http.ResponseWriter, r *http.Request) {
	withReportRead(func(w http.ResponseWriter, r *http.Request) {
		asOf, err := parseTanggalLaporan(r, "as_of")
		if err != nil {
			respondError(w, err)
			return
		}
		report, err := h.ReportUsecase.BalanceSheet(r.Context(), asOf)
		if err != nil {
			respondError(w, err)
			return
		}
		respondJSON(w, http.StatusOK, map[string]interface{}{"status": "success", "data": report})
	})(w, r)
}

// HandleIncomeStatement menyajikan Laba Rugi pada rentang tanggal
// (?from=&to=).
func (h *AccountingHandler) HandleIncomeStatement(w http.ResponseWriter, r *http.Request) {
	withReportRead(func(w http.ResponseWriter, r *http.Request) {
		from, err := parseTanggalLaporan(r, "from")
		if err != nil {
			respondError(w, err)
			return
		}
		to, err := parseTanggalLaporan(r, "to")
		if err != nil {
			respondError(w, err)
			return
		}
		report, err := h.ReportUsecase.IncomeStatement(r.Context(), from, to)
		if err != nil {
			respondError(w, err)
			return
		}
		respondJSON(w, http.StatusOK, map[string]interface{}{"status": "success", "data": report})
	})(w, r)
}

// HandleTrialBalance menyajikan Neraca Saldo pada rentang tanggal
// (?from=&to=).
func (h *AccountingHandler) HandleTrialBalance(w http.ResponseWriter, r *http.Request) {
	withReportRead(func(w http.ResponseWriter, r *http.Request) {
		from, err := parseTanggalLaporan(r, "from")
		if err != nil {
			respondError(w, err)
			return
		}
		to, err := parseTanggalLaporan(r, "to")
		if err != nil {
			respondError(w, err)
			return
		}
		report, err := h.ReportUsecase.TrialBalance(r.Context(), from, to)
		if err != nil {
			respondError(w, err)
			return
		}
		respondJSON(w, http.StatusOK, map[string]interface{}{"status": "success", "data": report})
	})(w, r)
}

// HandleGeneralLedger menyajikan Buku Besar satu akun pada rentang tanggal
// (?account_id=&from=&to=).
func (h *AccountingHandler) HandleGeneralLedger(w http.ResponseWriter, r *http.Request) {
	withReportRead(func(w http.ResponseWriter, r *http.Request) {
		accountID, ok := parseInt64Param(r, "account_id")
		if !ok || accountID <= 0 {
			respondJSON(w, http.StatusBadRequest, map[string]string{"message": "Parameter account_id tidak valid"})
			return
		}
		from, err := parseTanggalLaporan(r, "from")
		if err != nil {
			respondError(w, err)
			return
		}
		to, err := parseTanggalLaporan(r, "to")
		if err != nil {
			respondError(w, err)
			return
		}
		report, err := h.ReportUsecase.GeneralLedger(r.Context(), accountID, from, to)
		if err != nil {
			respondError(w, err)
			return
		}
		respondJSON(w, http.StatusOK, map[string]interface{}{"status": "success", "data": report})
	})(w, r)
}
