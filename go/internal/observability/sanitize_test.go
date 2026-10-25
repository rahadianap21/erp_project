package observability

import (
	"strings"
	"testing"
)

func TestRedactMenghapusDataSensitif(t *testing.T) {
	input := `{"password":"rahasia123","email":"budi@example.com","phone":"081234567890"} Authorization: Bearer abc.def.ghi`
	output := Redact(input)

	for _, forbidden := range []string{"rahasia123", "budi@example.com", "081234567890", "abc.def.ghi"} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("Redact masih memuat %q pada %q", forbidden, output)
		}
	}
	if !strings.Contains(output, "[redacted]") {
		t.Fatalf("Redact harus mengganti nilai sensitif, dapat %q", output)
	}
}

func TestRedactEmailDanTeleponTanpaKunci(t *testing.T) {
	output := Redact("hubungi budi@example.com atau 081234567890")
	if strings.Contains(output, "budi@example.com") || strings.Contains(output, "081234567890") {
		t.Fatalf("email/telepon masih tersisa: %q", output)
	}
}

func TestRedactMenghapusNIKDanTokenKueri(t *testing.T) {
	input := "pelanggan 3273010101990001 token=abc123&email=budi@example.com"
	output := Redact(input)

	if strings.Contains(output, "3273010101990001") || strings.Contains(output, "abc123") {
		t.Fatalf("data sensitif masih tersisa: %q", output)
	}
}

func TestCleanRouteMembuangQueryDanFragment(t *testing.T) {
	got := CleanRoute("/members/123?email=budi@example.com#detail")
	if got != "/members/123" {
		t.Fatalf("CleanRoute = %q, want /members/123", got)
	}
}

func TestFingerprintStabilSaatIDBerubah(t *testing.T) {
	first := Fingerprint(
		"backend",
		"sale 123 tidak ditemukan",
		"handler.GetSale() /app/sale.go:10",
		"/api/sales/123",
	)
	second := Fingerprint(
		"backend",
		"sale 456 tidak ditemukan",
		"handler.GetSale() /app/sale.go:20",
		"/api/sales/456",
	)

	if first != second {
		t.Fatalf("fingerprint harus sama untuk error yang sama, dapat %q vs %q", first, second)
	}
}

func TestFingerprintBerbedaUntukSumberBerbeda(t *testing.T) {
	backend := Fingerprint("backend", "gagal", "frame", "/api/x")
	frontend := Fingerprint("frontend", "gagal", "frame", "/api/x")

	if backend == frontend {
		t.Fatal("fingerprint backend dan frontend tidak boleh sama")
	}
}

func TestTruncateTidakMemotongUTF8(t *testing.T) {
	got := Truncate("halo dunia", 4)
	if got != "halo" {
		t.Fatalf("Truncate = %q, want halo", got)
	}
}
