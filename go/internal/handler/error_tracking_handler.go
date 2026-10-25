package handler

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"go_backend/internal/domain"
	"go_backend/internal/observability"
)

const maxErrorPayloadBytes = 64 << 10

// ErrorCapture adalah kontrak minimal yang dibutuhkan endpoint ingest. Dibuat
// sebagai interface supaya handler bisa diuji tanpa database.
type ErrorCapture interface {
	Capture(ctx context.Context, event domain.ErrorEvent) error
}

// ErrorTrackingHandler menerima laporan error dari frontend (melalui route
// handler Next.js). Endpoint ini sengaja tidak memakai JWT: pengguna yang
// belum login pun bisa mengalami error. Akses dilindungi kunci ingest dan
// rate limit di router.
type ErrorTrackingHandler struct {
	capture   ErrorCapture
	ingestKey string
}

func NewErrorTrackingHandler(capture ErrorCapture, ingestKey string) *ErrorTrackingHandler {
	return &ErrorTrackingHandler{capture: capture, ingestKey: strings.TrimSpace(ingestKey)}
}

type errorEventRequest struct {
	Name           string `json:"name"`
	Level          string `json:"level"`
	Message        string `json:"message"`
	Stack          string `json:"stack"`
	ComponentStack string `json:"componentStack"`
	Route          string `json:"route"`
	Method         string `json:"method"`
	StatusCode     int    `json:"statusCode"`
	RequestID      string `json:"requestId"`
	OccurredAt     string `json:"occurredAt"`
}

// HandleIngest melayani POST /api/errors.
func (h *ErrorTrackingHandler) HandleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		respondJSON(w, http.StatusMethodNotAllowed, map[string]string{"message": "method tidak diizinkan"})
		return
	}

	if h.ingestKey != "" {
		provided := r.Header.Get("X-Error-Ingest-Key")
		if subtle.ConstantTimeCompare([]byte(provided), []byte(h.ingestKey)) != 1 {
			respondJSON(w, http.StatusUnauthorized, map[string]string{"message": "kunci ingest tidak valid"})
			return
		}
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxErrorPayloadBytes)
	var payload errorEventRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"message": "payload error tidak valid"})
		return
	}

	message := strings.TrimSpace(payload.Message)
	if name := strings.TrimSpace(payload.Name); name != "" {
		if message == "" {
			message = name
		} else {
			message = name + ": " + message
		}
	}
	if message == "" && strings.TrimSpace(payload.Stack) == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"message": "pesan atau stack wajib diisi"})
		return
	}

	route := observability.CleanRoute(payload.Route)
	if route == "" {
		route = observability.CleanRoute(r.URL.Path)
	}

	var occurredAt time.Time
	if parsed, err := time.Parse(time.RFC3339Nano, payload.OccurredAt); err == nil {
		occurredAt = parsed
	}

	event := domain.ErrorEvent{
		Source:         domain.ErrorSourceFrontend,
		Level:          payload.Level,
		Message:        message,
		Stack:          payload.Stack,
		ComponentStack: payload.ComponentStack,
		Route:          route,
		Method:         payload.Method,
		StatusCode:     payload.StatusCode,
		RequestID:      payload.RequestID,
		OccurredAt:     occurredAt,
	}

	if err := h.capture.Capture(r.Context(), event); err != nil {
		log.Printf("error tracking: gagal menyimpan laporan frontend: %s",
			observability.Redact(err.Error()))
		respondJSON(w, http.StatusServiceUnavailable, map[string]string{"message": "gagal menyimpan laporan error"})
		return
	}

	// 202: laporan diterima; frontend tidak perlu menunggu/menampilkan apa pun.
	respondJSON(w, http.StatusAccepted, map[string]string{"status": "accepted"})
}
