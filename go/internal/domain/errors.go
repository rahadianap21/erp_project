package domain

import "fmt"

// AuthorizationError menandai error yang disebabkan oleh TOKEN klien, bukan
// kegagalan internal server.
//
// Pemisahan ini penting: sebelum tipe ini ada, handler memetakan seluruh error
// dari UpdateUser ke HTTP 500. Akibatnya percobaan privilege escalation yang
// sudah ditolak dengan benar tetap tercatat sebagai "server error" di log dan
// monitoring, sehingga alarm error asli ikut tenggelam di antara percobaan
// attacker. Penolakan wewenang harus keluar sebagai 403.
type AuthorizationError struct {
	Reason string
}

func (e *AuthorizationError) Error() string {
	return e.Reason
}

// NewAuthorizationError membuat penolakan wewenang dengan alasan yang aman untuk
// ditunjukkan ke user.
func NewAuthorizationError(format string, args ...any) *AuthorizationError {
	return &AuthorizationError{Reason: fmt.Sprintf(format, args...)}
}

// BusinessRuleError menandai penolakan permintaan yang VALID secara sintaks
// tetapi melanggar aturan bisnis (stok kurang, periode ditutup, status
// dokumen tidak sesuai, dan sebagainya).
//
// Sebelum tipe ini ada, handler memetakan aturan bisnis lewat pencocokan
// prefiks string ("pesanan ditolak: ...", "penerimaan ditolak: ...") sehingga
// pesan baru yang lupa didaftarkan bocor keluar sebagai 500 Internal Server
// Error. Dengan error bertipe, penolakan aturan bisnis selalu berstatus 422
// Unprocessable Entity tanpa bergantung pada teks pesan.
type BusinessRuleError struct {
	Reason string
}

func (e *BusinessRuleError) Error() string {
	return e.Reason
}

// NewBusinessRuleError membuat penolakan aturan bisnis dengan pesan yang aman
// untuk ditampilkan ke user.
func NewBusinessRuleError(format string, args ...any) *BusinessRuleError {
	return &BusinessRuleError{Reason: fmt.Sprintf(format, args...)}
}

// NotFoundError menandai entitas yang dicari tidak ada. Berbeda dengan
// sentinel per-entitas (ErrSaleNotFound, ErrAccountNotFound, ...), tipe ini
// dipakai saat pencarian terjadi di dalam usecase dengan pesan dinamis
// sehingga handler cukup memetakan satu tipe ke HTTP 404.
type NotFoundError struct {
	Reason string
}

func (e *NotFoundError) Error() string {
	return e.Reason
}

func NewNotFoundError(format string, args ...any) *NotFoundError {
	return &NotFoundError{Reason: fmt.Sprintf(format, args...)}
}

// DuplicateError menandai pelanggaran constraint unique di database.
//
// Sebelum tipe ini ada, handler memeriksa kode SQLSTATE "23505" sebagai string
// di pesan error ("isUniqueViolation"), yang bocor ke lapisan handler dan
// diulang terpisah di setiap handler (member, payment method, unit). Sekarang
// repository mengubah pelanggaran unique menjadi tipe ini, dan satu pemetaan
// di handler memutarnya menjadi HTTP 409 Conflict.
type DuplicateError struct {
	Reason string
}

func (e *DuplicateError) Error() string {
	return e.Reason
}

func NewDuplicateError(format string, args ...any) *DuplicateError {
	return &DuplicateError{Reason: fmt.Sprintf(format, args...)}
}
