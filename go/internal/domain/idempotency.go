package domain

import (
	"context"
	"errors"
)

var (
	// ErrIdempotencyKeyReused dikembalikan ketika satu idempotency key dipakai
	// untuk payload/permintaan yang berbeda. Pemanggil tidak boleh mengeksekusi
	// permintaan baru karena hasil permintaan pertama sudah terikat pada key itu.
	ErrIdempotencyKeyReused = errors.New("idempotency key sudah dipakai untuk permintaan berbeda")
	// ErrIdempotencyInProgress dikembalikan ketika permintaan lain dengan key
	// yang sama masih diproses. Klien boleh mengulang setelah permintaan pertama
	// selesai; mengeksekusi sekarang berisiko membuat dokumen ganda.
	ErrIdempotencyInProgress = errors.New("permintaan dengan idempotency key ini sedang diproses")
)

// IdempotencyRecord adalah hasil klaim sebuah Idempotency-Key.
//
// Record dengan Completed=false berarti pemanggil memenangkan klaim dan wajib
// memanggil Complete (bila berhasil) atau Release (bila gagal) memakai ID.
// Record dengan Completed=true adalah respons yang pernah disimpan dan harus
// diputar ulang tanpa menjalankan handler lagi.
type IdempotencyRecord struct {
	ID                  int64
	RequestHash         string
	Completed           bool
	ResponseStatus      int
	ResponseBody        []byte
	ResponseContentType string
}

// IdempotencyStore menyimpan klaim dan respons untuk header Idempotency-Key.
type IdempotencyStore interface {
	// Claim mengklaim (userID, scope, key) untuk requestHash. Bila key sudah
	// selesai, record Completed dikembalikan. Bila masih diproses, atau hash-nya
	// berbeda, error sentinel di atas dikembalikan.
	Claim(ctx context.Context, userID int64, scope, key, requestHash string) (*IdempotencyRecord, error)
	// Complete menyimpan status/body respons sehingga retry berikutnya diputar
	// ulang tanpa mengeksekusi handler lagi.
	Complete(ctx context.Context, id int64, status int, body []byte, contentType string) error
	// Release menghapus klaim yang belum selesai agar permintaan yang gagal bisa
	// dicoba ulang dengan key yang sama.
	Release(ctx context.Context, id int64) error
}
