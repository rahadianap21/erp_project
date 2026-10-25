package alert

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go_backend/internal/domain"
)

func TestNewWebhookAlerterKosongMengembalikanNil(t *testing.T) {
	if NewWebhookAlerter("  ") != nil {
		t.Fatal("URL kosong harus menonaktifkan alerter, bukan menandai alert terkirim")
	}
}

func TestWebhookMengirimRingkasanTanpaDataSensitif(t *testing.T) {
	var received map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &received)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	alerter := NewWebhookAlerter(server.URL)
	if alerter == nil {
		t.Fatal("alerter tidak boleh nil untuk URL valid")
	}

	now := time.Now()
	err := alerter.Send(context.Background(), domain.ErrorAlert{
		Fingerprint: "abc",
		Source:      "backend",
		Level:       "error",
		Message:     "login gagal password=rahasia123 email=budi@example.com",
		Route:       "/api/login?email=budi@example.com",
		Occurrences: 1,
		FirstSeen:   now,
		LastSeen:    now,
	})
	if err != nil {
		t.Fatalf("Send error: %v", err)
	}

	encoded, _ := json.Marshal(received)
	for _, forbidden := range []string{"rahasia123", "budi@example.com", "?email="} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("payload webhook memuat data sensitif %q: %s", forbidden, encoded)
		}
	}
	if received["text"] == nil || received["content"] == nil {
		t.Fatalf("payload harus memuat text dan content: %s", encoded)
	}
}
