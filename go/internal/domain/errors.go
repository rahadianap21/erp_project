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
