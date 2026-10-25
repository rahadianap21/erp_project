package handler

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go_backend/internal/domain"
)

type fakeErrorCapture struct {
	events []domain.ErrorEvent
	err    error
}

func (f *fakeErrorCapture) Capture(_ context.Context, event domain.ErrorEvent) error {
	f.events = append(f.events, event)
	return f.err
}

func TestHandleIngestMenolakTanpaKunci(t *testing.T) {
	capture := &fakeErrorCapture{}
	h := NewErrorTrackingHandler(capture, "kunci-rahasia")

	request := httptest.NewRequest(http.MethodPost, "/api/errors",
		strings.NewReader(`{"message":"boom"}`))
	recorder := httptest.NewRecorder()
	h.HandleIngest(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", recorder.Code)
	}
	if len(capture.events) != 0 {
		t.Fatal("event tidak boleh disimpan tanpa kunci")
	}
}

func TestHandleIngestMenyimpanLaporanFrontend(t *testing.T) {
	capture := &fakeErrorCapture{}
	h := NewErrorTrackingHandler(capture, "kunci-rahasia")

	request := httptest.NewRequest(http.MethodPost, "/api/errors",
		strings.NewReader(`{"name":"TypeError","message":"gagal memuat","stack":"at x","route":"/dashboard?email=budi@example.com"}`))
	request.Header.Set("X-Error-Ingest-Key", "kunci-rahasia")
	recorder := httptest.NewRecorder()
	h.HandleIngest(recorder, request)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202; body=%s", recorder.Code, recorder.Body.String())
	}
	if len(capture.events) != 1 {
		t.Fatalf("event = %d, want 1", len(capture.events))
	}
	event := capture.events[0]
	if event.Source != domain.ErrorSourceFrontend {
		t.Fatalf("source = %q, want frontend", event.Source)
	}
	if event.Route != "/dashboard" {
		t.Fatalf("route = %q, want /dashboard", event.Route)
	}
	if !strings.Contains(event.Message, "TypeError") {
		t.Fatalf("nama error harus ikut pada pesan, dapat %q", event.Message)
	}
}

func TestHandleIngestMenolakTanpaPesanDanStack(t *testing.T) {
	capture := &fakeErrorCapture{}
	h := NewErrorTrackingHandler(capture, "")

	request := httptest.NewRequest(http.MethodPost, "/api/errors", strings.NewReader(`{}`))
	recorder := httptest.NewRecorder()
	h.HandleIngest(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
}

type recordingErrorWriter struct {
	header   http.Header
	status   int
	body     bytes.Buffer
	recorded error
}

func (w *recordingErrorWriter) Header() http.Header {
	if w.header == nil {
		w.header = http.Header{}
	}
	return w.header
}

func (w *recordingErrorWriter) WriteHeader(status int) {
	w.status = status
}

func (w *recordingErrorWriter) Write(data []byte) (int, error) {
	return w.body.Write(data)
}

func (w *recordingErrorWriter) RecordError(err error) {
	w.recorded = err
}

func TestWriteError500MenyembunyikanDetailDanMencatat(t *testing.T) {
	writer := &recordingErrorWriter{}
	internal := errors.New("db gagal password=rahasia123")

	WriteError(writer, internal)

	if writer.status != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", writer.status)
	}
	if strings.Contains(writer.body.String(), "rahasia123") {
		t.Fatalf("respons membocorkan detail internal: %q", writer.body.String())
	}
	if !strings.Contains(writer.body.String(), "terjadi kesalahan pada server") {
		t.Fatalf("respons harus berisi pesan generik, dapat %q", writer.body.String())
	}
	if !errors.Is(writer.recorded, internal) {
		t.Fatal("error internal harus diteruskan ke pelacak")
	}
}

func TestWriteError4xxTetapMenampilkanPesanAman(t *testing.T) {
	writer := &recordingErrorWriter{}
	WriteError(writer, domain.NewBusinessRuleError("stok tidak cukup"))

	if writer.status != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", writer.status)
	}
	if !strings.Contains(writer.body.String(), "stok tidak cukup") {
		t.Fatalf("pesan aturan bisnis harus tetap tampil, dapat %q", writer.body.String())
	}
	if writer.recorded != nil {
		t.Fatal("4xx tidak perlu dicatat sebagai error sistem")
	}
}
