// Package observability berisi helper yang dipakai jalur pelacakan error:
// pembersihan data sensitif dan pembentukan fingerprint.
//
// Aturan utamanya: error tracking tidak boleh mengirim password, token, atau
// data pelanggan. Lapisan handler tidak pernah membaca request body/header,
// tetapi pesan error dari database/usecase bisa saja mengandung nilai mentah.
// Karena itu semua teks dibersihkan lagi di sini sebagai pertahanan terakhir.
package observability

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

var (
	// authHeader menangkap seluruh nilai header Authorization, termasuk token
	// di belakang kata "Bearer", supaya aturan key-value tidak menyisakan token.
	authHeader = regexp.MustCompile(`(?i)\bauthorization\s*[:=]\s*(?:bearer\s+)?[A-Za-z0-9._~+/=-]+`)

	// bearerToken menangkap "Bearer <token>" di luar header Authorization.
	bearerToken = regexp.MustCompile(`(?i)\bbearer\s+[A-Za-z0-9._~+/=-]+`)

	// sensitiveKV menangkap pasangan kunci-nilai yang lazim membawa rahasia
	// atau data pribadi, baik bentuk JSON ("password":"...") maupun query/form
	// (password=...). Nilai diganti [redacted].
	sensitiveKV = regexp.MustCompile(`(?i)("?(?:password|passwd|pwd|secret|token|authorization|api[_-]?key|apikey|access[_-]?token|refresh[_-]?token|cookie|email|e-mail|phone|telepon|no[_-]?hp|alamat|address|nama|name|username|full[_-]?name|nik|npwp|no[_-]?rekening|rekening|account[_-]?number|card[_-]?number|customer|member)"?\s*[:=]\s*)("[^"]*"|'[^']*'|[^\s,;}&]+)`)

	emailAddress = regexp.MustCompile(`(?i)\b[A-Z0-9._%+\-]+@[A-Z0-9.\-]+\.[A-Z]{2,}\b`)
	phoneNumber  = regexp.MustCompile(`(\+?62|0)8[0-9]{7,12}`)
	longDigits   = regexp.MustCompile(`\b[0-9]{13,19}\b`)

	uuidValue  = regexp.MustCompile(`(?i)\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`)
	longHex    = regexp.MustCompile(`(?i)\b[0-9a-f]{16,}\b`)
	numberPart = regexp.MustCompile(`\b[0-9]+\b`)
	spaceRun   = regexp.MustCompile(`\s+`)
)

const (
	maxMessageRunes = 2000
	maxStackRunes   = 16000
	maxRouteRunes   = 512
	maxFrameRunes   = 1000
)

// Redact menghapus password, token, email, nomor telepon, nomor kartu/NIK, dan
// nilai dari kunci sensitif lain. Fungsi ini idempoten dan aman dipanggil
// berkali-kali (frontend lalu backend).
func Redact(value string) string {
	if value == "" {
		return ""
	}

	value = authHeader.ReplaceAllString(value, "authorization: [redacted]")
	value = bearerToken.ReplaceAllString(value, "Bearer [redacted]")
	value = sensitiveKV.ReplaceAllString(value, "${1}[redacted]")
	value = emailAddress.ReplaceAllString(value, "[email]")
	value = phoneNumber.ReplaceAllString(value, "[phone]")
	value = longDigits.ReplaceAllString(value, "[number]")
	return value
}

// Truncate memotong teks berdasarkan jumlah rune (bukan byte) supaya karakter
// UTF-8 tidak terbelah.
func Truncate(value string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max])
}

// CleanMessage membersihkan dan membatasi pesan error.
func CleanMessage(value string) string {
	return Truncate(strings.TrimSpace(Redact(value)), maxMessageRunes)
}

// CleanStack membersihkan dan membatasi stack trace.
func CleanStack(value string) string {
	return Truncate(strings.TrimSpace(Redact(value)), maxStackRunes)
}

// CleanRoute hanya mempertahankan path (tanpa query string) yang dibersihkan.
func CleanRoute(value string) string {
	if index := strings.IndexAny(value, "?#"); index >= 0 {
		value = value[:index]
	}
	return Truncate(strings.TrimSpace(Redact(value)), maxRouteRunes)
}

// Fingerprint membentuk pengenal stabil untuk sekelompok error. Nilai yang
// berubah antar kejadian (angka, UUID, hex panjang) dinormalisasi agar error
// yang sama tidak dianggap baru setiap kali ID-nya berubah.
func Fingerprint(source, message, stack, route string) string {
	normalizedMessage := normalize(source + " " + message)
	frame := firstStackFrame(stack)
	normalizedStack := normalize(frame)
	normalizedRoute := normalize(route)

	sum := sha256.Sum256([]byte(
		source + "\n" + normalizedMessage + "\n" + normalizedStack + "\n" + normalizedRoute,
	))
	return hex.EncodeToString(sum[:])
}

func normalize(value string) string {
	value = Redact(value)
	value = uuidValue.ReplaceAllString(value, "<id>")
	value = longHex.ReplaceAllString(value, "<hex>")
	value = numberPart.ReplaceAllString(value, "<n>")
	value = strings.ToLower(value)
	return Truncate(strings.TrimSpace(spaceRun.ReplaceAllString(value, " ")), maxFrameRunes)
}

// firstStackFrame mengambil frame teratas yang bermakna. Baris kosong dan baris
// pembuka ("Error: ...", "goroutine ...") dilewati.
func firstStackFrame(stack string) string {
	for _, line := range strings.Split(stack, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "error:") || strings.HasPrefix(lower, "goroutine ") {
			continue
		}
		return line
	}
	return ""
}
