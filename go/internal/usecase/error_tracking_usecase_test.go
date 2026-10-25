package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go_backend/internal/domain"
)

type fakeErrorStore struct {
	result   domain.ErrorRecordResult
	events   []domain.ErrorEvent
	claim    bool
	claimErr error
	released []string
	failWith error
}

func (f *fakeErrorStore) Record(_ context.Context, event domain.ErrorEvent) (domain.ErrorRecordResult, error) {
	f.events = append(f.events, event)
	if f.failWith != nil {
		return domain.ErrorRecordResult{}, f.failWith
	}
	return f.result, nil
}

func (f *fakeErrorStore) ClaimAlert(
	_ context.Context,
	_ string,
	_, _ time.Time,
) (bool, error) {
	return f.claim, f.claimErr
}

func (f *fakeErrorStore) ReleaseAlert(_ context.Context, fingerprint string, _ time.Time) error {
	f.released = append(f.released, fingerprint)
	return nil
}

type fakeErrorAlerter struct {
	alerts   []domain.ErrorAlert
	failWith error
}

func (f *fakeErrorAlerter) Send(_ context.Context, alert domain.ErrorAlert) error {
	if f.failWith != nil {
		return f.failWith
	}
	f.alerts = append(f.alerts, alert)
	return nil
}

func newTestTracker(store domain.ErrorStore, alerter domain.ErrorAlerter, now time.Time) *ErrorTrackingUsecase {
	tracker := NewErrorTrackingUsecase(store, alerter, 24*time.Hour)
	tracker.now = func() time.Time { return now }
	return tracker
}

func TestCaptureErrorBaruMengirimAlertDanMerahasiakanData(t *testing.T) {
	now := time.Date(2026, 10, 25, 10, 0, 0, 0, time.UTC)
	store := &fakeErrorStore{
		claim: true,
		result: domain.ErrorRecordResult{
			IsNew: true, Occurrences: 1, FirstSeen: now, LastSeen: now,
		},
	}
	alerter := &fakeErrorAlerter{}
	tracker := newTestTracker(store, alerter, now)

	err := tracker.capture(context.Background(), domain.ErrorEvent{
		Source:  "backend",
		Message: "login gagal password=rahasia123 email=budi@example.com",
		Route:   "/api/login?email=budi@example.com",
	}, false)
	if err != nil {
		t.Fatalf("capture error: %v", err)
	}

	if len(store.events) != 1 {
		t.Fatalf("event tersimpan = %d, want 1", len(store.events))
	}
	saved := store.events[0]
	for _, forbidden := range []string{"rahasia123", "budi@example.com", "?email="} {
		if strings.Contains(saved.Message, forbidden) || strings.Contains(saved.Route, forbidden) {
			t.Fatalf("data sensitif tersimpan: message=%q route=%q", saved.Message, saved.Route)
		}
	}
	if saved.Fingerprint == "" {
		t.Fatal("fingerprint wajib diisi")
	}

	if len(alerter.alerts) != 1 {
		t.Fatalf("alert terkirim = %d, want 1", len(alerter.alerts))
	}
	if len(store.released) != 0 {
		t.Fatalf("alert sukses tidak boleh melepas klaim, released = %d", len(store.released))
	}
}

func TestCaptureErrorLamaDalamCooldownTidakMengirimAlert(t *testing.T) {
	now := time.Date(2026, 10, 25, 10, 0, 0, 0, time.UTC)
	store := &fakeErrorStore{claim: false, result: domain.ErrorRecordResult{
		IsNew: false, Occurrences: 5, FirstSeen: now.Add(-48 * time.Hour),
		LastSeen: now,
	}}
	alerter := &fakeErrorAlerter{}
	tracker := newTestTracker(store, alerter, now)

	if err := tracker.capture(context.Background(), domain.ErrorEvent{
		Source: "backend", Message: "gagal", Route: "/api/x",
	}, false); err != nil {
		t.Fatalf("capture error: %v", err)
	}

	if len(alerter.alerts) != 0 {
		t.Fatalf("alert tidak boleh dikirim dalam cooldown, dapat %d", len(alerter.alerts))
	}
}

func TestCaptureErrorLamaSetelahCooldownMengirimAlert(t *testing.T) {
	now := time.Date(2026, 10, 25, 10, 0, 0, 0, time.UTC)
	store := &fakeErrorStore{claim: true, result: domain.ErrorRecordResult{
		IsNew: false, Occurrences: 9, FirstSeen: now.Add(-96 * time.Hour),
		LastSeen: now,
	}}
	alerter := &fakeErrorAlerter{}
	tracker := newTestTracker(store, alerter, now)

	if err := tracker.capture(context.Background(), domain.ErrorEvent{
		Source: "backend", Message: "gagal", Route: "/api/x",
	}, false); err != nil {
		t.Fatalf("capture error: %v", err)
	}

	if len(alerter.alerts) != 1 {
		t.Fatalf("alert setelah cooldown = %d, want 1", len(alerter.alerts))
	}
}

func TestCaptureMeneruskanErrorPenyimpanan(t *testing.T) {
	store := &fakeErrorStore{failWith: errors.New("db down")}
	tracker := newTestTracker(store, nil, time.Now())

	err := tracker.Capture(context.Background(), domain.ErrorEvent{
		Source: "backend", Message: "gagal",
	})
	if err == nil {
		t.Fatal("error penyimpanan harus diteruskan ke pemanggil")
	}
}

func TestCaptureMelepasKlaimSaatAlertGagal(t *testing.T) {
	now := time.Date(2026, 10, 25, 10, 0, 0, 0, time.UTC)
	store := &fakeErrorStore{claim: true, result: domain.ErrorRecordResult{
		IsNew: true, Occurrences: 1, FirstSeen: now, LastSeen: now,
	}}
	alerter := &fakeErrorAlerter{failWith: errors.New("webhook down")}
	tracker := newTestTracker(store, alerter, now)

	if err := tracker.capture(context.Background(), domain.ErrorEvent{
		Source: "backend", Message: "gagal", Route: "/api/x",
	}, false); err != nil {
		t.Fatalf("capture error: %v", err)
	}

	if len(store.released) != 1 {
		t.Fatalf("klaim harus dilepas agar bisa dicoba lagi, released = %d", len(store.released))
	}
}

func TestCaptureTetapSuksesSaatKlaimAlertGagal(t *testing.T) {
	now := time.Date(2026, 10, 25, 10, 0, 0, 0, time.UTC)
	store := &fakeErrorStore{
		claimErr: errors.New("db down"),
		result:   domain.ErrorRecordResult{IsNew: true, Occurrences: 1},
	}
	tracker := newTestTracker(store, &fakeErrorAlerter{}, now)

	if err := tracker.capture(context.Background(), domain.ErrorEvent{
		Source: "backend", Message: "gagal",
	}, false); err != nil {
		t.Fatalf("kegagalan klaim alert tidak boleh menggagalkan penyimpanan error: %v", err)
	}
}

func TestPrepareMengisiDefault(t *testing.T) {
	now := time.Date(2026, 10, 25, 10, 0, 0, 0, time.UTC)
	tracker := newTestTracker(&fakeErrorStore{}, nil, now)

	event := tracker.prepare(domain.ErrorEvent{})
	if event.Source != domain.ErrorSourceBackend {
		t.Fatalf("source default = %q, want backend", event.Source)
	}
	if event.Level != "error" {
		t.Fatalf("level default = %q, want error", event.Level)
	}
	if event.Message == "" {
		t.Fatal("message tidak boleh kosong")
	}
	if event.Fingerprint == "" {
		t.Fatal("fingerprint tidak boleh kosong")
	}
}
