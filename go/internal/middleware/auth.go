// internal/middleware/auth.go
package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"go_backend/internal/config"
	"go_backend/internal/domain"
	"net/http"
	"strings"

	"errors"
	"github.com/golang-jwt/jwt/v5"
)

// Define custom type untuk key context agar menghindari bentrok (best practice Go)
type contextKey string

const UserContextKey contextKey = "user_claims"

// =========================================================================
// SOLUSI UTAMA CORS: Menangani Preflight (OPTIONS) dan Menyematkan Header
// =========================================================================
func EnableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Setel asal domain frontend Next.js Anda
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:3000")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		// Jika browser mengirim request OPTIONS (Preflight), langsung jawab dengan 204 No Content
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Teruskan ke middleware / handler berikutnya (seperti JWTMiddleware)
		next(w, r)
	}
}

// JWTMiddleware menghadang request untuk validasi token
func JWTMiddleware(cfg *config.Config, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Batasi total pembacaan body hanya maksimal 1MB
		r.Body = http.MaxBytesReader(w, r.Body, 1048576)

		// 1. Ambil header Authorization
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Akses ditolak: token tidak ditemukan"})
			return
		}

		// 2. Format harus berupa: "Bearer <token_string>"
		bearerToken := strings.Split(authHeader, " ")
		if len(bearerToken) != 2 || strings.ToLower(bearerToken[0]) != "bearer" {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Format token tidak valid (wajib Bearer)"})
			return
		}

		tokenString := bearerToken[1]

		// 3. Parse dan verifikasi tanda tangan token menggunakan Secret Key kita
		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			// Pastikan metode algoritma yang digunakan adalah HS256
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("metode enkripsi tidak sesuai: %v", t.Header["alg"])
			}
			return []byte(cfg.JWTSecret), nil
		})

		// 4. Cek apakah token valid atau sudah expired
		if err != nil || !token.Valid {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Token tidak valid atau telah kedaluwarsa"})
			return
		}

		// 5. Ekstrak data klaim (payload) dari JWT
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Gagal membaca data otorisasi"})
			return
		}

		// 6. Simpan claims ke dalam Context dan teruskan request ke handler tujuan
		ctx := context.WithValue(r.Context(), UserContextKey, claims)
		next(w, r.WithContext(ctx))
	}
}

// =========================================================================
// RequireRole membatasi akses hanya untuk daftar Role ID tertentu yang diizinkan.
//
// Role ID dibaca dari requestAuth (hasil resolve dari database), bukan dari klaim
// JWT, supaya perpindahan role langsung berlaku tanpa perlu logout/login ulang.
// =========================================================================
func RequireRole(allowedRoles []int32, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1048576)

		auth, ok := GetRequestAuth(r)
		if !ok {
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "Akses ditolak: gagal memvalidasi identitas"})
			return
		}

		isAllowed := false
		for _, role := range allowedRoles {
			if auth.RoleID == role {
				isAllowed = true
				break
			}
		}

		if !isAllowed {
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "Akses ditolak: Anda tidak memiliki hak akses untuk area ini"})
			return
		}

		next(w, r)
	}
}

// =========================================================================
// RequirePermission memeriksa izin akses berdasarkan permission aktual dari
// database yang sudah disimpan oleh JWTMiddlewareWithResolver.
//
// Tidak lagi membaca klaim JWT karena token bisa berumur 24 jam — pencabutan
// permission baru harus langsung berlaku pada request berikutnya.
// =========================================================================
func RequirePermission(requiredPermission string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		auth, ok := GetRequestAuth(r)
		if !ok {
			w.WriteHeader(http.StatusForbidden)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "Akses ditolak, sesi otorisasi tidak ditemukan"})
			return
		}

		for _, p := range auth.Permissions {
			if p == requiredPermission {
				next(w, r)
				return
			}
		}

		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Anda tidak memiliki hak akses: " + requiredPermission})
	}
}

// =========================================================================
// CurrentActor mengekstrak identitas pelaku dari JWT untuk keperluan
// pemeriksaan wewenang.
//
// Hanya UserID yang diambil, bukan role_id atau permission. Token JWT berlaku
// 24 jam, sehingga permission di dalamnya bisa sudah dicabut; layer usecase
// akan membaca ulang role dan permission aktual dari database.
func CurrentActor(r *http.Request) domain.Actor {
	return domain.Actor{UserID: CurrentUserID(r)}
}

// CurrentUserID mengekstrak user id dari JWT yang sudah dilewati JWTMiddleware.
//
// PENTING: jangan pernah menulis r.Context().Value("user_id").(int64) secara
// langsung. Middleware hanya menyimpan claims pada key UserContextKey
// ("user_claims"), sehingga Value("user_id") selalu nil dan hasilnya selalu 0.
// Nilai 0 itu berbahaya karena kolom *_by adalah foreign key ke s_users(id),
// dan s_users tidak punya id=0 — jadi penulisan 0 memicu pelanggaran FK dan
// seluruh endpoint DELETE membalas HTTP 500 tanpa menghapus apa pun.
//
// Nilai user_id juga datang dalam dua bentuk tipe: saat token baru diterbitkan
// nilainya int64, tetapi setelah melewati serialisasi JSON di dalam JWT claim
// nilainya kembali menjadi float64. Keduanya harus diterima.
func CurrentUserID(r *http.Request) int64 {
	if r == nil {
		return 0
	}

	claims, ok := r.Context().Value(UserContextKey).(jwt.MapClaims)
	if !ok {
		return 0
	}

	switch v := claims["user_id"].(type) {
	case float64:
		return int64(v)
	case float32:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case json.Number:
		if parsed, err := v.Int64(); err == nil {
			return parsed
		}
		return 0
	case string:
		var parsed int64
		if _, err := fmt.Sscanf(v, "%d", &parsed); err == nil {
			return parsed
		}
		return 0
	default:
		return 0
	}
}

// requestAuth menyimpan identitas yang sudah divalidasi berdasarkan data
// aktual di database.
//
// Permission dan role_id tidak diambil dari klaim JWT, melainkan dibaca ulang
// dari database ketika setiap request autentikasi melewati JWTMiddlewareWithResolver.
// Ini memastikan pencabutan permission atau penonaktifan akun langsung berlaku
// pada request berikutnya (token JWT bisa berlaku 24 jam).
type RequestAuth struct {
	UserID      int64
	RoleID      int32
	App         string
	Permissions []string
}

type authContextKey string

const authKey authContextKey = "request_auth"

// getRequestAuth mengambil otorisasi yang sudah divalidasi dari context.
func GetRequestAuth(r *http.Request) (RequestAuth, bool) {
	auth, ok := r.Context().Value(authKey).(RequestAuth)
	return auth, ok
}

// WithRequestAuth menyuntikkan RequestAuth ke context request.
// Dipakai oleh handler test untuk meniru hasil middleware tanpa membangun token JWT.
func WithRequestAuth(r *http.Request, auth RequestAuth) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), authKey, auth))
}

// tokenCandidates mengembalikan kandidat token JWT untuk dicoba berurutan.
//
// Urutan: token dari header Authorization (bila format Bearer-nya benar),
// lalu nilai cookie auth_token. Cookie dipakai oleh alur BFF/proxy Next.js,
// di mana browser mengirim cookie httpOnly yang diset oleh route handler.
// Header "Bearer undefined" (sisa kode lama yang membaca cookie httpOnly
// dari JavaScript) sengaja tidak dimasukkan supaya langsung jatuh ke cookie.
func tokenCandidates(r *http.Request) []string {
	var candidates []string

	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") && parts[1] != "" {
			candidates = append(candidates, parts[1])
		}
	}

	if cookie, err := r.Cookie("auth_token"); err == nil && cookie.Value != "" {
		candidates = append(candidates, cookie.Value)
	}

	return candidates
}

// parseClaims memvalidasi satu kandidat token dan mengembalikan klaimnya.
// Fungsi ini tidak menulis response; pemanggil yang memutuskan status error.
func parseClaims(cfg *config.Config, tokenString string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("metode enkripsi tidak sesuai: %v", t.Header["alg"])
		}
		return []byte(cfg.JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("token tidak valid")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, errors.New("gagal membaca data otorisasi")
	}
	return claims, nil
}

// JWTMiddlewareWithResolver memvalidasi token JWT lalu membaca permission/role
// aktual dari database untuk memutuskan akses pada saat request terjadi.
func JWTMiddlewareWithResolver(cfg *config.Config, resolver domain.PermissionResolver, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1048576)

		candidates := tokenCandidates(r)
		if len(candidates) == 0 {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "Akses ditolak: token tidak ditemukan"})
			return
		}

		var claims jwt.MapClaims
		for _, tokenString := range candidates {
			parsed, err := parseClaims(cfg, tokenString)
			if err != nil {
				continue
			}
			claims = parsed
			break
		}

		if claims == nil {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "Token tidak valid atau telah kedaluwarsa"})
			return
		}

		userID := extractUserID(claims)
		if userID <= 0 {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "Token tidak valid"})
			return
		}

		otorisasi, err := resolver.OtorisasiAktif(r.Context(), userID)
		if err != nil {
			if errors.Is(err, domain.ErrAkunTidakBerhak) {
				w.WriteHeader(http.StatusForbidden)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": "Akun pelaku tidak ditemukan atau sudah dinonaktifkan"})
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "Gagal memvalidasi izin akun"})
			return
		}

		auth := RequestAuth{
			UserID:      otorisasi.UserID,
			RoleID:      otorisasi.RoleID,
			App:         otorisasi.App,
			Permissions: otorisasi.Permissions,
		}
		// Claims tetap disimpan supaya handler lama yang membaca nama/email dari
		// UserContextKey tetap bekerja. Yang menentukan wewenang tetap requestAuth,
		// bukan klaim.
		ctx := context.WithValue(r.Context(), UserContextKey, claims)
		ctx = context.WithValue(ctx, authKey, auth)
		next(w, r.WithContext(ctx))
	}
}

func extractUserID(claims jwt.MapClaims) int64 {
	switch v := claims["user_id"].(type) {
	case float64:
		return int64(v)
	case float32:
		return int64(v)
	case int64:
		return v
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case json.Number:
		if parsed, err := v.Int64(); err == nil {
			return parsed
		}
		return 0
	case string:
		var parsed int64
		if _, err := fmt.Sscanf(v, "%d", &parsed); err == nil {
			return parsed
		}
		return 0
	default:
		return 0
	}
}
