package middleware

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"go_backend/internal/domain"
)

const (
	// IdempotencyKeyHeader adalah nama header publik untuk kunci idempotensi.
	IdempotencyKeyHeader = "Idempotency-Key"
	// IdempotencyReplayedHeader menandai respons yang diputar ulang dari
	// penyimpanan, bukan hasil eksekusi handler baru.
	IdempotencyReplayedHeader = "Idempotency-Replayed"

	maxIdempotencyKeyLength = 255
	maxIdempotencyBodyBytes = 1 << 20
)

// Idempotency membungkus endpoint mutasi dengan dukungan Idempotency-Key.
//
// Alurnya:
//  1. Klaim (user, method+path, key) beserta hash body/query SEBELUM handler.
//  2. Bila klaim sudah selesai, putar ulang status/body tanpa eksekusi ulang.
//  3. Bila key yang sama dipakai untuk request berbeda, tolak 409.
//  4. Bila eksekusi menghasilkan 2xx, simpan respons. Selain 2xx, lepaskan
//     klaim supaya pengguna dapat memperbaiki input dan mencoba lagi.
//
// required=true mewajibkan header pada POST/PUT/PATCH/DELETE; GET/HEAD tidak
// pernah membutuhkan idempotency key.
func Idempotency(store domain.IdempotencyStore, required bool, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !isMutatingMethod(r.Method) {
			next(w, r)
			return
		}

		key := strings.TrimSpace(r.Header.Get(IdempotencyKeyHeader))
		if key == "" {
			if !required {
				next(w, r)
				return
			}
			writeIdempotencyError(w, http.StatusPreconditionRequired,
				"header Idempotency-Key wajib diisi untuk permintaan ini")
			return
		}
		if len(key) > maxIdempotencyKeyLength {
			writeIdempotencyError(w, http.StatusBadRequest,
				"header Idempotency-Key terlalu panjang")
			return
		}

		auth, ok := GetRequestAuth(r)
		if !ok || auth.UserID <= 0 {
			writeIdempotencyError(w, http.StatusUnauthorized, "tidak terautentikasi")
			return
		}

		body, err := io.ReadAll(io.LimitReader(r.Body, maxIdempotencyBodyBytes+1))
		if err != nil {
			writeIdempotencyError(w, http.StatusBadRequest, "body permintaan tidak dapat dibaca")
			return
		}
		if len(body) > maxIdempotencyBodyBytes {
			writeIdempotencyError(w, http.StatusRequestEntityTooLarge,
				"body permintaan terlalu besar")
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(body))
		r.ContentLength = int64(len(body))

		scope := r.Method + " " + r.URL.Path
		requestHash := hashIdempotencyRequest(r, body)

		record, err := store.Claim(r.Context(), auth.UserID, scope, key, requestHash)
		if err != nil {
			switch {
			case errors.Is(err, domain.ErrIdempotencyKeyReused):
				writeIdempotencyError(w, http.StatusConflict,
					"idempotency key sudah dipakai untuk permintaan berbeda")
			case errors.Is(err, domain.ErrIdempotencyInProgress):
				writeIdempotencyError(w, http.StatusConflict,
					"permintaan dengan idempotency key ini sedang diproses")
			default:
				writeIdempotencyError(w, http.StatusInternalServerError,
					"gagal memproses idempotency key")
			}
			return
		}

		if record.Completed {
			if record.ResponseContentType != "" {
				w.Header().Set("Content-Type", record.ResponseContentType)
			}
			w.Header().Set(IdempotencyReplayedHeader, "true")
			status := record.ResponseStatus
			if status == 0 {
				status = http.StatusOK
			}
			w.WriteHeader(status)
			_, _ = w.Write(record.ResponseBody)
			return
		}

		rec := newBufferedResponseWriter()
		next(rec, r)

		status := rec.status
		if status == 0 {
			status = http.StatusOK
		}
		contentType := rec.header.Get("Content-Type")
		// Penyimpanan hasil tidak boleh ikut batal saat klien memutus koneksi:
		// transaksi bisnis sudah commit dan retry justru membutuhkan respons ini.
		persistCtx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), 5*time.Second)
		defer cancel()

		if status >= 200 && status < 300 {
			// Bila penyimpanan respons gagal setelah transaksi bisnis sukses,
			// klaim dibiarkan IN_PROGRESS. Retry akan ditolak 409 sampai klaim
			// basi, sehingga tidak ada peluang dokumen tercatat dua kali.
			if err := store.Complete(persistCtx, record.ID, status, rec.body.Bytes(), contentType); err != nil {
				log.Printf("idempotency: gagal menyimpan respons key %q: %v", key, err)
			}
		} else if err := store.Release(persistCtx, record.ID); err != nil {
			log.Printf("idempotency: gagal melepas klaim key %q: %v", key, err)
		}

		for name, values := range rec.header {
			for _, value := range values {
				w.Header().Add(name, value)
			}
		}
		w.WriteHeader(status)
		_, _ = w.Write(rec.body.Bytes())
	}
}

func isMutatingMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func hashIdempotencyRequest(r *http.Request, body []byte) string {
	h := sha256.New()
	_, _ = h.Write([]byte(r.Method))
	_, _ = h.Write([]byte{'\n'})
	_, _ = h.Write([]byte(r.URL.EscapedPath()))
	_, _ = h.Write([]byte{'\n'})
	_, _ = h.Write([]byte(r.URL.RawQuery))
	_, _ = h.Write([]byte{'\n'})
	_, _ = h.Write(body)
	return hex.EncodeToString(h.Sum(nil))
}

func writeIdempotencyError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"message": message})
}

// bufferedResponseWriter menahan respons handler sampai idempotency middleware
// memutuskan untuk menyimpannya, memutar ulang, atau melepaskan klaim.
type bufferedResponseWriter struct {
	header http.Header
	body   bytes.Buffer
	status int
}

func newBufferedResponseWriter() *bufferedResponseWriter {
	return &bufferedResponseWriter{header: make(http.Header)}
}

func (w *bufferedResponseWriter) Header() http.Header {
	return w.header
}

func (w *bufferedResponseWriter) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
}

func (w *bufferedResponseWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.body.Write(p)
}
