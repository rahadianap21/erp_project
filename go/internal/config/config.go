package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// MinJWTSecretLen adalah panjang minimum secret JWT.
//
// Secret pendek atau yang mudah ditebak, dan HS256 bersifat simetris:
// siapa pun yang bisa membaca source atau menebak secret bisa membuat token
// palsu dengan role_id admin dan lolos RequireRole. Karena itu secret kosong
// atau pendek harus menggagalkan startup, bukan diam-diam memakai fallback.
const MinJWTSecretLen = 32

// DefaultPort hanya dipakai kalau PORT tidak di-set. Ini bukan rahasia,
// jadi fallback di sini aman.
const DefaultPort = "8090"

// DefaultCORSOrigin dipakai kalau CORS_ORIGIN tidak di-set. Mengarah ke dev
// server Next.js lokal, bukan nilai rahasia.
const DefaultCORSOrigin = "http://localhost:3000"

type Config struct {
	Port        string
	DatabaseURL string
	JWTSecret   string
	CORSOrigin  string
	// TrustedProxies adalah daftar IP/CIDR reverse proxy yang boleh mengisi
	// X-Forwarded-For. Bila kosong, header tersebut diabaikan sepenuhnya.
	TrustedProxies []string
}

// LoadConfig membaca konfigurasi dari environment dan mengembalikan error
// bila ada nilai wajib yang kosong atau tidak aman.
//
// Tidak ada fallback untuk JWT_SECRET maupun DATABASE_URL. Dulu keduanya
// memakai nilai hardcoded, yang membuat service tetap berjalan dengan secret
// yang tertulis di source code.
func LoadConfig() (*Config, error) {
	var problems []error

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		problems = append(problems, errors.New("DATABASE_URL wajib diisi"))
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	switch {
	case jwtSecret == "":
		problems = append(problems, errors.New("JWT_SECRET wajib diisi"))
	case len(jwtSecret) < MinJWTSecretLen:
		problems = append(problems, fmt.Errorf(
			"JWT_SECRET terlalu pendek: %d karakter, minimal %d karakter",
			len(jwtSecret), MinJWTSecretLen,
		))
	}

	if len(problems) > 0 {
		return nil, fmt.Errorf("konfigurasi tidak valid: %w", errors.Join(problems...))
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = DefaultPort
	}

	corsOrigin := os.Getenv("CORS_ORIGIN")
	if corsOrigin == "" {
		corsOrigin = DefaultCORSOrigin
	}

	return &Config{
		Port:           port,
		DatabaseURL:    databaseURL,
		JWTSecret:      jwtSecret,
		CORSOrigin:     corsOrigin,
		TrustedProxies: splitCSV(os.Getenv("TRUSTED_PROXIES")),
	}, nil
}

// splitCSV memecah daftar dipisah koma dan membuang nilai kosong.
func splitCSV(raw string) []string {
	if raw == "" {
		return nil
	}
	var result []string
	for _, item := range strings.Split(raw, ",") {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
