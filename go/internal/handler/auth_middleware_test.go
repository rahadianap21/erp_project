package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go_backend/internal/config"
	"go_backend/internal/domain"

	"github.com/golang-jwt/jwt/v5"
)

// testSecret hanya untuk unit test. Nilai ini sengaja tidak sama dengan
// secret yang pernah ditulis hardcoded di config.go, supaya tidak ada yang
// salah mengira ini secret sungguhan.
const testSecret = "secret-khusus-unit-test-erp-batch-1-0000"

var testCfg = &config.Config{JWTSecret: testSecret}

// resolverPalsu meniru domain.PermissionResolver.
//
// Nilai kembaliannya bisa diubah di tengah test untuk mensimulasikan permission
// berubah di database tanpa perlu menerbitkan token baru. Inilah justru inti
// perbaikan FIX #4: token tetap sama, keputusan otorisasi berubah.
type resolverPalsu struct {
	perms     []string
	roleID    int32
	err       error
	dipanggil int
}

func (r *resolverPalsu) OtorisasiAktif(ctx context.Context, userID int64) (*domain.OtorisasiUser, error) {
	r.dipanggil++
	if r.err != nil {
		return nil, r.err
	}
	return &domain.OtorisasiUser{
		UserID:      userID,
		RoleID:      r.roleID,
		Permissions: r.perms,
	}, nil
}

// requestWithClaims membuat request yang membawa claims JWT persis seperti
// yang dilakukan JWTMiddleware, yaitu pada key UserContextKey.
func requestWithClaims(claims jwt.MapClaims) *http.Request {
	r := httptest.NewRequest(http.MethodDelete, "/api/products?id=1", nil)
	return r.WithContext(context.WithValue(r.Context(), UserContextKey, claims))
}

func TestCurrentUserID(t *testing.T) {
	tests := []struct {
		name   string
		claims jwt.MapClaims
		want   int64
	}{
		{
			// Kasus yang paling sering terjadi di produksi: setelah token
			// melewati serialisasi JSON di dalam JWT, angka jadi float64.
			name:   "user_id float64 (hasil round-trip JSON)",
			claims: jwt.MapClaims{"user_id": float64(42)},
			want:   42,
		},
		{
			name:   "user_id float64 besar",
			claims: jwt.MapClaims{"user_id": float64(7)},
			want:   7,
		},
		{
			// Kasus ketika handler dipanggil tanpa JSON round-trip.
			name:   "user_id int64",
			claims: jwt.MapClaims{"user_id": int64(7)},
			want:   7,
		},
		{
			name:   "user_id int",
			claims: jwt.MapClaims{"user_id": 99},
			want:   99,
		},
		{
			name:   "user_id int32",
			claims: jwt.MapClaims{"user_id": int32(5)},
			want:   5,
		},
		{
			name:   "user_id float32",
			claims: jwt.MapClaims{"user_id": float32(12)},
			want:   12,
		},
		{
			name:   "user_id json.Number",
			claims: jwt.MapClaims{"user_id": json.Number("123")},
			want:   123,
		},
		{
			name:   "user_id string numerik",
			claims: jwt.MapClaims{"user_id": "55"},
			want:   55,
		},
		{
			name:   "user_id string non-numerik harus 0",
			claims: jwt.MapClaims{"user_id": "bukan-angka"},
			want:   0,
		},
		{
			name:   "json.Number non-numerik harus 0",
			claims: jwt.MapClaims{"user_id": json.Number("abc")},
			want:   0,
		},
		{
			name:   "key user_id tidak ada",
			claims: jwt.MapClaims{"role_id": float64(1)},
			want:   0,
		},
		{
			name:   "user_id tipe tidak didukung (bool)",
			claims: jwt.MapClaims{"user_id": true},
			want:   0,
		},
		{
			name:   "user_id nil",
			claims: jwt.MapClaims{"user_id": nil},
			want:   0,
		},
		{
			name:   "claims kosong",
			claims: jwt.MapClaims{},
			want:   0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := CurrentUserID(requestWithClaims(tc.claims)); got != tc.want {
				t.Fatalf("CurrentUserID = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestCurrentUserIDTanpaClaims(t *testing.T) {
	// Request yang belum melewati JWTMiddleware tidak punya key
	// UserContextKey, dan harus menghasilkan 0 tanpa panic.
	r := httptest.NewRequest(http.MethodDelete, "/api/products?id=1", nil)
	if got := CurrentUserID(r); got != 0 {
		t.Fatalf("CurrentUserID tanpa claims = %d, want 0", got)
	}
}

func TestCurrentUserIDClaimsSalahTipe(t *testing.T) {
	// Nilai pada key UserContextKey bukan jwt.MapClaims. Type assertion
	// harus gagal dengan aman dan mengembalikan 0.
	r := httptest.NewRequest(http.MethodDelete, "/api/products?id=1", nil)
	bad := r.WithContext(context.WithValue(r.Context(), UserContextKey, "ini bukan claims"))

	if got := CurrentUserID(bad); got != 0 {
		t.Fatalf("CurrentUserID dengan claims salah tipe = %d, want 0", got)
	}
}

func TestCurrentUserIDRequestNil(t *testing.T) {
	if got := CurrentUserID(nil); got != 0 {
		t.Fatalf("CurrentUserID(nil) = %d, want 0", got)
	}
}

func signToken(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	claims["exp"] = jwt.NewNumericDate(time.Now().Add(2 * time.Hour))
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("gagal menandatangani token: %v", err)
	}
	return token
}

// TestCurrentUserIDviaJWTMiddleware menguji jalur end-to-end: token JWT
// asli ditandatangani, dibaca JWTMiddlewareWithResolver, permission dibaca dari
// database, lalu handler menarik user id. Inilah skenario yang tidak bisa
// ditangkap unit test biasa.
func TestCurrentUserIDviaJWTMiddleware(t *testing.T) {
	token := signToken(t, jwt.MapClaims{
		"user_id":     float64(1),
		"role_id":     float64(1),
		"permissions": []string{"products:delete"},
	})

	resolver := &resolverPalsu{perms: []string{"products:delete"}}

	dapat := int64(-1)
	next := func(w http.ResponseWriter, r *http.Request) {
		dapat = CurrentUserID(r)
		w.WriteHeader(http.StatusOK)
	}

	// Bungkus handler dengan RequirePermission supaya alurnya sama dengan
	// router sungguhan: JWT dicek dulu, permission dibaca dari DB, lalu handler.
	handler := JWTMiddlewareWithResolver(testCfg, resolver,
		RequirePermission("products:delete", next))

	req := httptest.NewRequest(http.MethodDelete, "/api/products?id=6", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body.String())
	}
	if dapat != 1 {
		t.Fatalf("CurrentUserID via middleware = %d, want 1", dapat)
	}
}

// TestCurrentUserIDviaJWTMiddlewareTanpaPermission memastikan middleware
// memblokir sebelum handler jalan ketika permission di database tidak cocok.
func TestCurrentUserIDviaJWTMiddlewareTanpaPermission(t *testing.T) {
	token := signToken(t, jwt.MapClaims{
		"user_id":     float64(1),
		"role_id":     float64(5),
		"permissions": []string{"products:read"},
	})

	resolver := &resolverPalsu{perms: []string{"products:read"}}

	dipanggil := false
	next := func(w http.ResponseWriter, r *http.Request) {
		dipanggil = true
		w.WriteHeader(http.StatusOK)
	}

	handler := JWTMiddlewareWithResolver(testCfg, resolver,
		RequirePermission("products:delete", next))

	req := httptest.NewRequest(http.MethodDelete, "/api/products?id=6", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
	if dipanggil {
		t.Fatal("handler tidak boleh dijalankan saat permission tidak cocok")
	}
}

// TestCurrentUserIDTokenTidakValid memastikan token rusak ditolak 401 dan
// handler tidak pernah dipanggil.
func TestCurrentUserIDTokenTidakValid(t *testing.T) {
	dipanggil := false
	next := func(w http.ResponseWriter, r *http.Request) {
		dipanggil = true
		w.WriteHeader(http.StatusOK)
	}

	handler := JWTMiddleware(testCfg, next)

	req := httptest.NewRequest(http.MethodDelete, "/api/products?id=6", nil)
	req.Header.Set("Authorization", "Bearer token-garbage")
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
	if dipanggil {
		t.Fatal("handler tidak boleh dijalankan untuk token tidak valid")
	}
}

// ---------------------------------------------------------------------------
// FIX #4 — wewenang harus dibaca dari database, bukan dari klaim JWT
//
// Token JWT berumur 24 jam. Kalau permission dan role_id dibaca dari klaim,
// mencabut permission atau memindahkan role tidak akan terlihat sampai semua
// token lama kedaluwarsa. Test di bawah memakai token yang SAMA persis di dua
// request, hanya kondisi database yang berubah di antaranya.
// ---------------------------------------------------------------------------

// TestPermissionDicabutBerlakuLangsungTanpaTokenBaru adalah regresi inti FIX #4.
//
// Versi lama membaca claims["permissions"], sehingga pada request kedua token
// yang sama masih sempat memakai permission yang sudah dicabut. Dengan resolver
// dari database, request kedua harus langsung ditolak meski token tidak berubah
// dan belum kedaluwarsa.
func TestPermissionDicabutBerlakuLangsungTanpaTokenBaru(t *testing.T) {
	token := signToken(t, jwt.MapClaims{
		"user_id":     float64(1),
		"role_id":     float64(1),
		"permissions": []string{"products:delete"},
	})

	resolver := &resolverPalsu{perms: []string{"products:delete", "products:read"}}

	dipanggil := 0
	next := func(w http.ResponseWriter, r *http.Request) {
		dipanggil++
		w.WriteHeader(http.StatusOK)
	}
	handler := JWTMiddlewareWithResolver(testCfg, resolver,
		RequirePermission("products:delete", next))

	kirim := func() int {
		req := httptest.NewRequest(http.MethodDelete, "/api/products?id=6", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		handler(rec, req)
		return rec.Code
	}

	if kode := kirim(); kode != http.StatusOK {
		t.Fatalf("permission masih ada, request harus 200, dapat %d", kode)
	}

	// Admin mencabut products:delete. Token BELUM disentuh, belum kedaluwarsa.
	resolver.perms = []string{"products:read"}

	if kode := kirim(); kode != http.StatusForbidden {
		t.Fatalf("permission sudah dicabut, request harus 403, dapat %d", kode)
	}
	if dipanggil != 1 {
		t.Fatalf("handler hanya boleh dipanggil sekali (request pertama), dapat %d", dipanggil)
	}
}

// TestPermissionBaruLangsungBerlaku adalah arah sebaliknya: permission yang
// baru diberikan harus bisa dipakai tanpa logout/login ulang.
func TestPermissionBaruLangsungBerlaku(t *testing.T) {
	token := signToken(t, jwt.MapClaims{
		"user_id":     float64(1),
		"role_id":     float64(5),
		"permissions": []string{"products:read"},
	})

	resolver := &resolverPalsu{perms: []string{"products:read"}}

	next := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
	handler := JWTMiddlewareWithResolver(testCfg, resolver,
		RequirePermission("po:approve", next))

	req := httptest.NewRequest(http.MethodPut, "/api/purchase-orders/approve?id=1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("permission belum ada, harus 403, dapat %d", rec.Code)
	}

	resolver.perms = []string{"products:read", "po:approve"}

	req2 := httptest.NewRequest(http.MethodPut, "/api/purchase-orders/approve?id=1", nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	rec2 := httptest.NewRecorder()
	handler(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("permission baru harus langsung berlaku, dapat %d", rec2.Code)
	}
}

// TestRolePindahLangsungBerlaku ke RequireRole: pindah role di database harus
// mengubah keputusan akses tanpa perlu token baru.
func TestRolePindahLangsungBerlaku(t *testing.T) {
	token := signToken(t, jwt.MapClaims{
		"user_id": float64(1),
		// Klaim masih bilang role 1 (Super Admin).
		"role_id":     float64(1),
		"permissions": []string{"users:create"},
	})

	resolver := &resolverPalsu{perms: []string{"users:create"}, roleID: 1}

	next := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}
	handler := JWTMiddlewareWithResolver(testCfg, resolver,
		RequireRole([]int32{1}, next))

	req := httptest.NewRequest(http.MethodGet, "/api/admin/dashboard", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("role di DB masih Super Admin, harus 200, dapat %d", rec.Code)
	}

	// User dipindahkan ke role lain. Klaim token masih role 1.
	resolver.roleID = 6

	req2 := httptest.NewRequest(http.MethodGet, "/api/admin/dashboard", nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	rec2 := httptest.NewRecorder()
	handler(rec2, req2)
	if rec2.Code != http.StatusForbidden {
		t.Fatalf("role sudah dipindah, harus 403, dapat %d", rec2.Code)
	}
}

// TestAkunNonAktifDitolakSetiapEndpoint menutup celah yang lebih luas dari Fix #1.
//
// Resolve wewenang ada di middleware, jadi penonaktifan akun sekarang berlaku di
// SELURUH endpoint terproteksi — bukan hanya /api/users seperti sebelumnya.
func TestAkunNonAktifDitolakSetiapEndpoint(t *testing.T) {
	token := signToken(t, jwt.MapClaims{
		"user_id":     float64(1),
		"role_id":     float64(1),
		"permissions": []string{"products:delete"},
	})

	resolver := &resolverPalsu{err: domain.ErrAkunTidakBerhak}

	dipanggil := false
	next := func(w http.ResponseWriter, r *http.Request) {
		dipanggil = true
		w.WriteHeader(http.StatusOK)
	}
	handler := JWTMiddlewareWithResolver(testCfg, resolver,
		RequirePermission("products:delete", next))

	req := httptest.NewRequest(http.MethodDelete, "/api/products?id=6", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("akun nonaktif harus 403, dapat %d", rec.Code)
	}
	if dipanggil {
		t.Fatal("handler tidak boleh jalan untuk akun nonaktif")
	}
}

// TestKegagalanDatabaseMenjadi500 menjaga 403 tidak dipakai untuk menutupi
// kegagalan sistem. Kalau database tidak merespon, itu 500 — bukan 403, supaya
// operator tahu ada masalah dan bukan salahnya pengguna.
func TestKegagalanDatabaseMenjadi500(t *testing.T) {
	token := signToken(t, jwt.MapClaims{
		"user_id":     float64(1),
		"role_id":     float64(1),
		"permissions": []string{"products:delete"},
	})

	resolver := &resolverPalsu{err: errors.New("koneksi database terputus")}

	next := func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler tidak boleh jalan ketika database gagal")
	}
	handler := JWTMiddlewareWithResolver(testCfg, resolver,
		RequirePermission("products:delete", next))

	req := httptest.NewRequest(http.MethodDelete, "/api/products?id=6", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("kegagalan database harus 500, dapat %d", rec.Code)
	}
}

// TestTokenTanpaUserIDDitolak memastikan token yang tidak punya user id tidak
// bisa melewati middleware cuma karena-signed dengan benar.
func TestTokenTanpaUserIDDitolak(t *testing.T) {
	token := signToken(t, jwt.MapClaims{
		"role_id":     float64(1),
		"permissions": []string{"products:delete"},
	})

	resolver := &resolverPalsu{perms: []string{"products:delete"}}
	next := func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler tidak boleh jalan untuk token tanpa user id")
	}
	handler := JWTMiddlewareWithResolver(testCfg, resolver,
		RequirePermission("products:delete", next))

	req := httptest.NewRequest(http.MethodDelete, "/api/products?id=6", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("token tanpa user id harus 401, dapat %d", rec.Code)
	}
	if resolver.dipanggil != 0 {
		t.Error("database tidak boleh ditanya untuk token tanpa identitas")
	}
}
