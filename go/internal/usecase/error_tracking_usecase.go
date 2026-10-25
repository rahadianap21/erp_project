package usecase

import (
	"context"
	"log"
	"strings"
	"time"

	"go_backend/internal/domain"
	"go_backend/internal/observability"
)

const (
	defaultErrorAlertCooldown = 24 * time.Hour
	maxComponentStackRunes    = 4000
)

// ErrorTrackingUsecase membersihkan event, menyimpannya, lalu mengirim alert
// hanya saat kelompok error baru muncul atau masih berulang setelah cooldown.
// Ini mencegah satu bug yang dipicu ribuan kali membanjiri webhook.
type ErrorTrackingUsecase struct {
	store    domain.ErrorStore
	alerter  domain.ErrorAlerter
	cooldown time.Duration
	now      func() time.Time
}

func NewErrorTrackingUsecase(
	store domain.ErrorStore,
	alerter domain.ErrorAlerter,
	cooldown time.Duration,
) *ErrorTrackingUsecase {
	if cooldown <= 0 {
		cooldown = defaultErrorAlertCooldown
	}
	return &ErrorTrackingUsecase{
		store:    store,
		alerter:  alerter,
		cooldown: cooldown,
		now:      time.Now,
	}
}

// Capture menyimpan error tanpa menahan response HTTP lebih lama dari yang
// diperlukan: penyimpanan dilakukan sinkron agar error tidak hilang, sedangkan
// pengiriman webhook berjalan di latar belakang.
func (u *ErrorTrackingUsecase) Capture(ctx context.Context, event domain.ErrorEvent) error {
	return u.capture(ctx, event, true)
}

func (u *ErrorTrackingUsecase) capture(ctx context.Context, event domain.ErrorEvent, asyncAlert bool) error {
	event = u.prepare(event)

	result, err := u.store.Record(ctx, event)
	if err != nil {
		return err
	}

	if result.IsNew {
		log.Printf("error tracking: error baru [%s] %s", event.Source, event.Message)
	}

	if u.alerter == nil {
		return nil
	}

	// Klaim alert dalam satu UPDATE atomik. Jika beberapa request mencatat
	// error yang sama bersamaan, hanya pemenang klaim yang mengirim webhook.
	claimedAt := u.now()
	claimed, err := u.store.ClaimAlert(ctx, event.Fingerprint, claimedAt, claimedAt.Add(-u.cooldown))
	if err != nil {
		log.Printf("error tracking: gagal mengklaim alert: %v", err)
		return nil
	}
	if !claimed {
		return nil
	}

	if asyncAlert {
		go u.sendAlert(event, result, claimedAt)
	} else {
		u.sendAlert(event, result, claimedAt)
	}
	return nil
}

func (u *ErrorTrackingUsecase) sendAlert(
	event domain.ErrorEvent,
	result domain.ErrorRecordResult,
	claimedAt time.Time,
) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	alert := domain.ErrorAlert{
		Fingerprint: event.Fingerprint,
		Source:      event.Source,
		Level:       event.Level,
		Message:     event.Message,
		Route:       event.Route,
		Stack:       event.Stack,
		StatusCode:  event.StatusCode,
		Occurrences: result.Occurrences,
		FirstSeen:   result.FirstSeen,
		LastSeen:    result.LastSeen,
	}

	if err := u.alerter.Send(ctx, alert); err != nil {
		// Jangan sertakan URL webhook di log.
		log.Printf("error tracking: gagal mengirim alert webhook: %v", err)
		if err := u.store.ReleaseAlert(ctx, event.Fingerprint, claimedAt); err != nil {
			log.Printf("error tracking: gagal melepas klaim alert: %v", err)
		}
	}
}

func (u *ErrorTrackingUsecase) prepare(event domain.ErrorEvent) domain.ErrorEvent {
	event.Source = strings.ToLower(strings.TrimSpace(event.Source))
	if event.Source != domain.ErrorSourceFrontend {
		event.Source = domain.ErrorSourceBackend
	}

	event.Level = cleanLevel(event.Level)

	event.Message = observability.CleanMessage(event.Message)
	event.Stack = observability.CleanStack(event.Stack)
	event.ComponentStack = observability.Truncate(
		observability.CleanStack(event.ComponentStack), maxComponentStackRunes,
	)
	event.Route = observability.CleanRoute(event.Route)
	event.Method = cleanMethod(event.Method)
	event.RequestID = observability.Truncate(observability.Redact(event.RequestID), 64)

	if event.Message == "" {
		event.Message = "error tanpa pesan"
	}
	if event.StatusCode < 0 || event.StatusCode > 599 {
		event.StatusCode = 0
	}

	now := u.now()
	if event.OccurredAt.IsZero() ||
		event.OccurredAt.After(now.Add(5*time.Minute)) ||
		event.OccurredAt.Before(now.Add(-30*24*time.Hour)) {
		event.OccurredAt = now
	}

	event.Fingerprint = observability.Fingerprint(
		event.Source, event.Message, event.Stack, event.Route,
	)
	return event
}

func cleanLevel(level string) string {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "panic", "fatal":
		return "panic"
	case "warning", "warn":
		return "warning"
	case "server":
		return "server"
	default:
		return "error"
	}
}

func cleanMethod(method string) string {
	method = strings.ToUpper(strings.TrimSpace(method))
	if len(method) > 10 {
		return ""
	}
	for _, r := range method {
		if r < 'A' || r > 'Z' {
			return ""
		}
	}
	return method
}
