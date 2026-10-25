// Package alert mengirim notifikasi error ke webhook generik (Slack, Discord,
// atau endpoint HTTP lain). Payload berisi field "text" (Slack) dan "content"
// (Discord) yang keduanya berisi ringkasan yang sama.
package alert

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go_backend/internal/domain"
	"go_backend/internal/observability"
)

// WebhookAlerter mengirim POST JSON ke URL webhook. URL kosong berarti
// notifikasi webhook dimatikan (error tetap disimpan ke database).
type WebhookAlerter struct {
	url    string
	client *http.Client
}

// NewWebhookAlerter mengembalikan nil bila URL kosong. Dengan begitu usecase
// tahu alert benar-benar dimatikan dan tidak menandai error sebagai "sudah
// dialertkan" hanya karena tidak ada kanal yang dikonfigurasi.
func NewWebhookAlerter(url string) domain.ErrorAlerter {
	url = strings.TrimSpace(url)
	if url == "" {
		return nil
	}
	return &WebhookAlerter{
		url:    url,
		client: &http.Client{Timeout: 8 * time.Second},
	}
}

func (a *WebhookAlerter) Send(ctx context.Context, event domain.ErrorAlert) error {
	if a == nil || a.url == "" {
		return nil
	}

	message := observability.Redact(event.Message)
	summary := formatSummary(event, message)

	payload := map[string]any{
		// Slack membaca "text", Discord membaca "content". Endpoint generik
		// tetap mendapat semua field terstruktur di bawah.
		"text":        summary,
		"content":     summary,
		"source":      event.Source,
		"level":       event.Level,
		"fingerprint": event.Fingerprint,
		"message":     message,
		"route":       observability.CleanRoute(event.Route),
		"status_code": event.StatusCode,
		"occurrences": event.Occurrences,
		"first_seen":  event.FirstSeen,
		"last_seen":   event.LastSeen,
		"stack":       observability.CleanStack(event.Stack),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "erp-error-tracker/1.0")

	resp, err := a.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook membalas HTTP %d", resp.StatusCode)
	}
	return nil
}

func formatSummary(event domain.ErrorAlert, message string) string {
	var builder strings.Builder
	builder.WriteString("[ERP ERROR] ")
	builder.WriteString(event.Source)
	if event.Level != "" && event.Level != "error" {
		builder.WriteString(" / ")
		builder.WriteString(event.Level)
	}
	builder.WriteString("\n")
	if event.Route != "" {
		builder.WriteString("Rute: ")
		builder.WriteString(observability.CleanRoute(event.Route))
		builder.WriteString("\n")
	}
	builder.WriteString("Pesan: ")
	builder.WriteString(message)
	builder.WriteString("\nFingerprint: ")
	builder.WriteString(event.Fingerprint)
	fmt.Fprintf(&builder, "\nKejadian: %d | pertama: %s | terakhir: %s",
		event.Occurrences,
		event.FirstSeen.Format(time.RFC3339),
		event.LastSeen.Format(time.RFC3339),
	)
	return builder.String()
}
