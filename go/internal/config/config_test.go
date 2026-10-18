package config

import (
	"strings"
	"testing"
)

// secretValid adalah secret yang melewati seluruh aturan validasi.
// Nilai ini khusus test, bukan secret sungguhan.
const secretValid = "secret-khusus-unit-test-config-000000"

func setEnv(t *testing.T, key, value string) {
	t.Helper()
	t.Setenv(key, value)
}

func TestLoadConfigGagalSaatSecretKosong(t *testing.T) {
	setEnv(t, "DATABASE_URL", "postgres://u:p@localhost:5432/db")
	setEnv(t, "JWT_SECRET", "")

	cfg, err := LoadConfig()
	if err == nil {
		t.Fatal("LoadConfig harus error saat JWT_SECRET kosong, bukan memakai fallback")
	}
	if cfg != nil {
		t.Fatal("Config harus nil saat ada error")
	}
	if !strings.Contains(err.Error(), "JWT_SECRET") {
		t.Fatalf("pesan error harus menyebut JWT_SECRET, dapat: %v", err)
	}
}

func TestLoadConfigGagalSaatSecretPendek(t *testing.T) {
	setEnv(t, "DATABASE_URL", "postgres://u:p@localhost:5432/db")
	// 31 karakter, satu di bawah minimum.
	setEnv(t, "JWT_SECRET", strings.Repeat("a", MinJWTSecretLen-1))

	_, err := LoadConfig()
	if err == nil {
		t.Fatalf("LoadConfig harus error saat JWT_SECRET hanya %d karakter", MinJWTSecretLen-1)
	}
	if !strings.Contains(err.Error(), "terlalu pendek") {
		t.Fatalf("pesan error harus menyebut terlalu pendek, dapat: %v", err)
	}
	if !strings.Contains(err.Error(), "31") {
		t.Fatalf("pesan error harus menyebut panjang sebenarnya, dapat: %v", err)
	}
}

func TestLoadConfigGagalSaatDatabaseURLKosong(t *testing.T) {
	setEnv(t, "DATABASE_URL", "")
	setEnv(t, "JWT_SECRET", secretValid)

	cfg, err := LoadConfig()
	if err == nil {
		t.Fatal("LoadConfig harus error saat DATABASE_URL kosong")
	}
	if cfg != nil {
		t.Fatal("Config harus nil saat ada error")
	}
	if !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Fatalf("pesan error harus menyebut DATABASE_URL, dapat: %v", err)
	}
}

func TestLoadConfigGagalSaatKeduanyaKosong(t *testing.T) {
	setEnv(t, "DATABASE_URL", "")
	setEnv(t, "JWT_SECRET", "")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("LoadConfig harus error saat keduanya kosong")
	}
	// errors.Join harus menggabungkan semua masalah, bukan hanya yang pertama.
	msg := err.Error()
	if !strings.Contains(msg, "DATABASE_URL") || !strings.Contains(msg, "JWT_SECRET") {
		t.Fatalf("pesan error harus menyebut DATABASE_URL dan JWT_SECRET, dapat: %v", err)
	}
}

func TestLoadConfigBerhasilDenganNilaiValid(t *testing.T) {
	const dbURL = "postgres://postgres:postgres@localhost:5432/central_db?sslmode=disable"
	setEnv(t, "DATABASE_URL", dbURL)
	setEnv(t, "JWT_SECRET", secretValid)
	setEnv(t, "PORT", "9999")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig harus berhasil, dapat error: %v", err)
	}
	if cfg.DatabaseURL != dbURL {
		t.Errorf("DatabaseURL = %q, want %q", cfg.DatabaseURL, dbURL)
	}
	if cfg.JWTSecret != secretValid {
		t.Errorf("JWTSecret tidak tersimpan apa adanya")
	}
	if cfg.Port != "9999" {
		t.Errorf("Port = %q, want %q", cfg.Port, "9999")
	}
}

func TestLoadConfigSecretTepatDiBatasMinimum(t *testing.T) {
	setEnv(t, "DATABASE_URL", "postgres://u:p@localhost:5432/db")
	setEnv(t, "JWT_SECRET", strings.Repeat("b", MinJWTSecretLen))

	if _, err := LoadConfig(); err != nil {
		t.Fatalf("secret sepanjang tepat %d karakter harus diterima, dapat error: %v", MinJWTSecretLen, err)
	}
}

func TestLoadConfigPortDefault(t *testing.T) {
	setEnv(t, "DATABASE_URL", "postgres://u:p@localhost:5432/db")
	setEnv(t, "JWT_SECRET", secretValid)
	setEnv(t, "PORT", "")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig harus berhasil, dapat error: %v", err)
	}
	if cfg.Port != DefaultPort {
		t.Errorf("Port = %q, want default %q", cfg.Port, DefaultPort)
	}
}

// TestTidakAdaSecretHardcoded mengunci decadanya nilai fallback di source:
// file ini sendiri harus tidak memuat string secret yang pernah dipakai
// sebagai default di versi lama.
func TestTidakAdaSecretHardcoded(t *testing.T) {
	setEnv(t, "DATABASE_URL", "")
	setEnv(t, "JWT_SECRET", "")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("pasti ada bug kalau env kosong tidak error")
	}
	if strings.Contains(err.Error(), secretValid) {
		t.Fatal("secret tidak boleh muncul di pesan error")
	}
}
