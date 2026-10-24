package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"go_backend/internal/domain"

	"github.com/go-playground/validator/v10"
)

// File ini memusatkan seluruh helper request/response handler yang sebelumnya
// terduplikasi di banyak file (respondJSON/parseInt64Param di accounting,
// decodeJSONBody di sale, parsePOID di purchase order, dsb.) sekaligus
// menjadi satu-satunya tempat pemetaan error → status HTTP.

// respondJSON menulis payload JSON dengan status yang diberikan.
<<<<<<< HEAD
func respondJSON(w http.ResponseWriter, status int, payload any) {
=======
func respondJSON(w http.ResponseWriter, status int, payload interface{}) {
>>>>>>> e47ff22 (refactor(go): sentralisasi pemetaan error domain ke handler.WriteError)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

// WriteError adalah SATU-SATUNYA jalur pemetaan error domain → respons HTTP
// error di seluruh aplikasi. Semua handler memanggil helper ini; tidak ada
// handler yang menyusun status HTTP dari error secara manual.
//
// Pemetaan:
//   - AuthorizationError → 403 (penolakan wewenang, bukan kegagalan server)
//   - validator.ValidationErrors → 400 (payload tidak valid secara struktural)
//   - NotFoundError + sentinel "tidak ditemukan" → 404
//   - DuplicateError + sentinel bentrok/konflik → 409
//   - BusinessRuleError + sentinel aturan bisnis + konflik transisi status → 422
//   - sisanya → 500 (kegagalan infrastruktur: DB, koneksi, dsb.)
//
// Sebelum helper ini ada, pemetaan yang sama ditulis ulang di lima tempat
// dengan strategi berbeda (pencocokan prefiks string di PO, daftar sentinel
// panjang di akuntansi, dsb.) sehingga pesan error baru yang lupa didaftarkan
// bocor keluar sebagai 500 Internal Server Error.
func WriteError(w http.ResponseWriter, err error) {
	if err == nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"message": "terjadi kesalahan pada server"})
		return
	}
	msg := err.Error()
	if msg == "" {
		msg = "terjadi kesalahan pada server"
	}
	respondJSON(w, StatusForError(err), map[string]string{"message": msg})
}

// StatusForError memetakan error domain ke status HTTP. Diekspos untuk test.
func StatusForError(err error) int {
	if err == nil {
		return http.StatusInternalServerError
	}

	var authz *domain.AuthorizationError
	if errors.As(err, &authz) {
		return http.StatusForbidden
	}

	var verrs validator.ValidationErrors
	if errors.As(err, &verrs) {
		return http.StatusBadRequest
	}

	var notFound *domain.NotFoundError
	if errors.As(err, &notFound) {
		return http.StatusNotFound
	}

	var dup *domain.DuplicateError
	if errors.As(err, &dup) {
		return http.StatusConflict
	}

	var rule *domain.BusinessRuleError
	if errors.As(err, &rule) {
		return http.StatusUnprocessableEntity
	}

	var conflict *domain.TransitionStatusConflictError
	if errors.As(err, &conflict) {
		return http.StatusUnprocessableEntity
	}

	var notPostable *domain.JournalNotPostableError
	if errors.As(err, &notPostable) {
		return http.StatusUnprocessableEntity
	}

	switch {
	case errors.Is(err, domain.ErrSaleNotFound),
		errors.Is(err, domain.ErrJournalNotFound),
		errors.Is(err, domain.ErrPeriodNotFound),
		errors.Is(err, domain.ErrAccountNotFound),
		errors.Is(err, domain.ErrIndukAkunNotFound):
		return http.StatusNotFound

	case errors.Is(err, domain.ErrSaleAlreadyVoid),
		errors.Is(err, domain.ErrSaleNotPosted),
<<<<<<< HEAD
		errors.Is(err, domain.ErrSaleSnapshotChanged),
		errors.Is(err, domain.ErrGRNReceiptChanged),
=======
>>>>>>> e47ff22 (refactor(go): sentralisasi pemetaan error domain ke handler.WriteError)
		errors.Is(err, domain.ErrJournalAlreadyRev),
		errors.Is(err, domain.ErrJournalAlreadyPost),
		errors.Is(err, domain.ErrJournalAlreadyVoid),
		errors.Is(err, domain.ErrAccountDuplicate),
		errors.Is(err, domain.ErrNomorJurnalBentrok),
		errors.Is(err, domain.ErrNomorPOBentrok):
		return http.StatusConflict

	case errors.Is(err, domain.ErrJournalUnbalanced),
		errors.Is(err, domain.ErrJournalLineDouble),
		errors.Is(err, domain.ErrKodeAkunWajib),
		errors.Is(err, domain.ErrNamaAkunWajib),
		errors.Is(err, domain.ErrTipeAkunInvalid),
		errors.Is(err, domain.ErrSaldoNormalMismatch),
		errors.Is(err, domain.ErrJournalLineKosong),
		errors.Is(err, domain.ErrJournalDateInvalid),
		errors.Is(err, domain.ErrJournalPeriodClosed),
		errors.Is(err, domain.ErrVoidPostedJournal),
		errors.Is(err, domain.ErrJournalNotPosted),
		errors.Is(err, domain.ErrJournalNotDraft),
		errors.Is(err, domain.ErrJournalPostBox),
		errors.Is(err, domain.ErrPeriodClosed),
		errors.Is(err, domain.ErrPeriodOpen),
		errors.Is(err, domain.ErrPeriodeInvalid),
		errors.Is(err, domain.ErrPeriodeBerubah),
		errors.Is(err, domain.ErrRentangTanggalInvalid),
		errors.Is(err, domain.ErrAutoPostingSettings),
		errors.Is(err, domain.ErrSaleItemsRequired),
		errors.Is(err, domain.ErrSaleLineInvalid),
		errors.Is(err, domain.ErrSaleProductNotFound),
		errors.Is(err, domain.ErrSaleStokTidakCukup),
		errors.Is(err, domain.ErrSalePaymentInvalid),
		errors.Is(err, domain.ErrSalePaymentExceeds),
		errors.Is(err, domain.ErrSalePaymentKurang),
		errors.Is(err, domain.ErrSaleMetodeGanda),
		errors.Is(err, domain.ErrSaleRedeemTanpaMember),
		errors.Is(err, domain.ErrSaleRedeemMelebihiSaldo),
		errors.Is(err, domain.ErrSaleTaxInvalid),
		errors.Is(err, domain.ErrSaleDiscountInvalid),
		errors.Is(err, domain.ErrGRNBarisTidakValid),
		errors.Is(err, domain.ErrGRNPenerimaanKosong),
		errors.Is(err, domain.ErrGRNPOSalahStatus),
		errors.Is(err, domain.ErrGRNOverReceipt),
		errors.Is(err, domain.ErrGRNPONotFound),
		errors.Is(err, domain.ErrGRNSatuanBermasalah),
		errors.Is(err, domain.ErrGRNBarisPONotFound),
		errors.Is(err, domain.ErrSettingPPNRateInvalid),
		errors.Is(err, domain.ErrSettingPPNModeInvalid):
		return http.StatusUnprocessableEntity
	}

	return http.StatusInternalServerError
}

// parseInt64Param membaca query param berbasis integer. false berarti param
// tidak ada atau bukan integer yang sah — keputusan status HTTP tetap di
// pemanggil karena pesannya spesifik per endpoint.
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

// parsePOID membaca query param ?id= dokumen Purchase Order. Nilai kosong
// maupun non-positif ditolak dengan pesan yang sama seperti sebelumnya.
func parsePOID(r *http.Request) (int64, error) {
	idStr := r.URL.Query().Get("id")
	poID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || poID <= 0 {
<<<<<<< HEAD
		return 0, errors.New("parameter ID dokumen PO tidak valid")
=======
		return 0, errors.New("Parameter ID dokumen PO tidak valid")
>>>>>>> e47ff22 (refactor(go): sentralisasi pemetaan error domain ke handler.WriteError)
	}
	return poID, nil
}

// decodeJSONBody membaca body JSON request. Bila gagal, sudah menulis 400 ke w
// dan mengembalikan false supaya pemanggil cukup return.
<<<<<<< HEAD
func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst any) bool {
=======
func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst interface{}) bool {
>>>>>>> e47ff22 (refactor(go): sentralisasi pemetaan error domain ke handler.WriteError)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"message": "body tidak valid: " + err.Error()})
		return false
	}
	return true
}
