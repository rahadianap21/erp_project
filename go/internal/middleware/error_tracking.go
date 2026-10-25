package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"go_backend/internal/domain"
	"go_backend/internal/observability"
)

// ErrorCapture adalah bagian dari usecase pelacakan error yang dipakai
// middleware. Interface kecil ini membuat middleware mudah diuji.
type ErrorCapture interface {
	Capture(ctx context.Context, event domain.ErrorEvent) error
}

// TrackErrors membungkus handler terluar: mencatat respons 5xx dan panic,
// memberi setiap request X-Request-ID, serta tidak pernah membaca body,
// header Authorization, atau query string.
//
// Error 4xx sengaja tidak dilacak karena itu penolakan yang memang diharapkan
// (validasi, izin, data tidak ditemukan) dan akan menenggelamkan error asli.
func TrackErrors(capture ErrorCapture, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := newRequestID()
		w.Header().Set("X-Request-ID", requestID)

		tracked := (&trackingResponseWriter{ResponseWriter: w, requestID: requestID}).
			withTrackingState(capture, r.URL.Path, r.Method)

		defer func() {
			if recovered := recover(); recovered != nil {
				message := fmt.Sprintf("panic: %v", recovered)
				status := tracked.status
				if status < 500 {
					status = http.StatusInternalServerError
				}
				if tracked.status == 0 {
					tracked.Header().Set("Content-Type", "application/json")
					tracked.WriteHeader(http.StatusInternalServerError)
					_, _ = tracked.Write([]byte(`{"message":"terjadi kesalahan pada server"}`))
				}
				tracked.panicked = true
				tracked.record(status, message, string(debug.Stack()))
				logTrackedError(r, requestID, status, message)
			} else if tracked.status >= 500 {
				message := ""
				stack := ""
				if tracked.err != nil {
					message = tracked.err.Error()
					stack = tracked.errStack
				}
				if message == "" {
					message = fmt.Sprintf("HTTP %d pada %s %s", tracked.status, r.Method, r.URL.Path)
				}
				tracked.record(tracked.status, message, stack)
				logTrackedError(r, requestID, tracked.status, message)
			}
		}()

		next.ServeHTTP(tracked, r)
	})
}

func (t *trackingResponseWriter) record(status int, message, stack string) {
	if t.capture == nil || t.recorded {
		return
	}
	t.recorded = true

	event := domain.ErrorEvent{
		Source:     domain.ErrorSourceBackend,
		Level:      "error",
		Message:    message,
		Stack:      stack,
		Route:      t.route,
		Method:     t.method,
		StatusCode: status,
		RequestID:  t.requestID,
	}
	if t.panicked {
		event.Level = "panic"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := t.capture.Capture(ctx, event); err != nil {
		log.Printf("error tracking: gagal menyimpan error backend (request_id=%s): %s",
			t.requestID, observability.Redact(err.Error()))
	}
}

func logTrackedError(r *http.Request, requestID string, status int, message string) {
	log.Printf("http %d: %s %s request_id=%s error=%q",
		status, r.Method, observability.CleanRoute(r.URL.Path), requestID, observability.Redact(message))
}

func newRequestID() string {
	var buffer [16]byte
	if _, err := rand.Read(buffer[:]); err != nil {
		return "unavailable"
	}
	return hex.EncodeToString(buffer[:])
}

// trackingResponseWriter menyimpan status dan error yang dilaporkan WriteError
// tanpa mengubah perilaku ResponseWriter lain.
type trackingResponseWriter struct {
	http.ResponseWriter
	requestID string
	status    int
	err       error
	errStack  string
	route     string
	method    string
	panicked  bool
	recorded  bool
	capture   ErrorCapture
}

func (w *trackingResponseWriter) WriteHeader(status int) {
	if w.status == 0 {
		w.status = status
	}
	w.ResponseWriter.WriteHeader(status)
}

func (w *trackingResponseWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(data)
}

// RecordError dipanggil oleh handler.WriteError. Status tetap ditentukan
// pemetaan error; middleware hanya menyimpan detailnya untuk kasus 5xx.
func (w *trackingResponseWriter) RecordError(err error) {
	if w.err == nil && err != nil {
		w.err = err
		w.errStack = string(debug.Stack())
	}
}

func (w *trackingResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *trackingResponseWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// withTrackingState mengisi capture/route/method setelah wrapper dibuat.
func (w *trackingResponseWriter) withTrackingState(capture ErrorCapture, route, method string) *trackingResponseWriter {
	w.capture = capture
	w.route = strings.TrimSpace(route)
	w.method = method
	return w
}
