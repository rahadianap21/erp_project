package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go_backend/internal/domain"
	"go_backend/internal/handler"
	"go_backend/internal/middleware"
)

type fakeCapture struct {
	events []domain.ErrorEvent
}

func (f *fakeCapture) Capture(_ context.Context, event domain.ErrorEvent) error {
	f.events = append(f.events, event)
	return nil
}

func TestTrackErrorsMencatat500DanMenyembunyikanDetail(t *testing.T) {
	capture := &fakeCapture{}
	h := middleware.TrackErrors(capture, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		handler.WriteError(w, errors.New("db gagal password=rahasia123"))
	}))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/secret?email=budi@example.com", nil)
	h.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", recorder.Code)
	}
	if strings.Contains(recorder.Body.String(), "rahasia123") {
		t.Fatalf("respons tidak boleh membocorkan detail error: %q", recorder.Body.String())
	}
	if recorder.Header().Get("X-Request-ID") == "" {
		t.Fatal("X-Request-ID harus diisi")
	}
	if len(capture.events) != 1 {
		t.Fatalf("event tercatat = %d, want 1", len(capture.events))
	}
	event := capture.events[0]
	if event.Source != domain.ErrorSourceBackend {
		t.Fatalf("source = %q, want backend", event.Source)
	}
	if event.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status event = %d, want 500", event.StatusCode)
	}
	if strings.Contains(event.Route, "?") {
		t.Fatalf("route tidak boleh memuat query: %q", event.Route)
	}
}

func TestTrackErrorsTidakMencatat4xx(t *testing.T) {
	capture := &fakeCapture{}
	h := middleware.TrackErrors(capture, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		handler.WriteError(w, domain.NewNotFoundError("tidak ditemukan"))
	}))

	recorder := httptest.NewRecorder()
	h.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/x", nil))

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
	if len(capture.events) != 0 {
		t.Fatalf("4xx tidak boleh dicatat, dapat %d event", len(capture.events))
	}
}

func TestTrackErrorsMencatatPanic(t *testing.T) {
	capture := &fakeCapture{}
	h := middleware.TrackErrors(capture, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic("boom")
	}))

	recorder := httptest.NewRecorder()
	h.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/x", nil))

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", recorder.Code)
	}
	if len(capture.events) != 1 {
		t.Fatalf("event panic = %d, want 1", len(capture.events))
	}
	if capture.events[0].Level != "panic" {
		t.Fatalf("level = %q, want panic", capture.events[0].Level)
	}
	if capture.events[0].Stack == "" {
		t.Fatal("stack panic wajib diisi")
	}
}
