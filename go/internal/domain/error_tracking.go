package domain

import (
	"context"
	"time"
)

// Sumber error yang dikenali pelacak.
const (
	ErrorSourceBackend  = "backend"
	ErrorSourceFrontend = "frontend"
)

// ErrorEvent adalah satu kejadian error yang sudah dibersihkan dari data
// sensitif. Tidak ada field untuk request body, query, cookie, atau identitas
// pelanggan — data itu memang tidak boleh dikumpulkan.
type ErrorEvent struct {
	Fingerprint    string
	Source         string
	Level          string
	Message        string
	Stack          string
	ComponentStack string
	Route          string
	Method         string
	StatusCode     int
	RequestID      string
	OccurredAt     time.Time
}

// ErrorRecordResult berisi keadaan grup setelah event disimpan.
type ErrorRecordResult struct {
	IsNew       bool
	Occurrences int64
	FirstSeen   time.Time
	LastSeen    time.Time
}

// ErrorStore menyimpan event dan mengelola kelompok error.
type ErrorStore interface {
	Record(ctx context.Context, event ErrorEvent) (ErrorRecordResult, error)
	// ClaimAlert menandai grup sebagai sudah dialertkan secara atomik dan
	// mengembalikan true hanya untuk satu pemanggil. Pemanggil yang kalah
	// harus melewati pengiriman alert. now adalah waktu klaim, notBefore
	// adalah batas waktu alert sebelumnya agar cooldown tetap dihormati.
	ClaimAlert(ctx context.Context, fingerprint string, now, notBefore time.Time) (bool, error)
	// ReleaseAlert mengembalikan klaim bila pengiriman alert gagal, sehingga
	// error berikutnya masih bisa memicu percobaan ulang.
	ReleaseAlert(ctx context.Context, fingerprint string, claimedAt time.Time) error
}

// ErrorAlert adalah ringkasan yang dikirim ke kanal notifikasi ketika sebuah
// kelompok error baru terdeteksi (atau muncul lagi setelah cooldown).
type ErrorAlert struct {
	Fingerprint string
	Source      string
	Level       string
	Message     string
	Route       string
	Stack       string
	StatusCode  int
	Occurrences int64
	FirstSeen   time.Time
	LastSeen    time.Time
}

// ErrorAlerter mengirim notifikasi error. Pengirim tidak boleh menulis URL
// webhook/rahasia ke log.
type ErrorAlerter interface {
	Send(ctx context.Context, alert ErrorAlert) error
}
